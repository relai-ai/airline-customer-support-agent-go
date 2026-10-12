"""Local Docker-only routing; hosted HTTPS gateways do not need this bridge."""
import http.client
import http.server
import ipaddress
import json
import socket
import subprocess
import sys
import time
from urllib.parse import urlsplit


def local_target(base):
    url = urlsplit(base)
    if url.scheme != 'http' or url.hostname != 'host.docker.internal':
        return None
    if url.username or url.password or url.query or url.fragment or url.port in (None, 8080):
        raise ValueError('Invalid local gateway address')
    return socket.gethostbyname(url.hostname), url.port


def ensure_bridge(base):
    target = local_target(base)
    if target is None:
        return
    network = json.loads(subprocess.check_output(['docker', 'network', 'inspect', 'bridge']))[0]
    bind = network['IPAM']['Config'][0]['Gateway']
    if not ipaddress.ip_address(bind).is_private:
        raise ValueError('Expected private inner Docker bridge')
    try:
        with socket.create_connection((bind, target[1]), timeout=1):
            return
    except OSError:
        pass
    child = subprocess.Popen([sys.executable, __file__, base],
                             stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                             stderr=subprocess.DEVNULL, start_new_session=True)
    for _ in range(50):
        if child.poll() is not None:
            raise RuntimeError('Local gateway bridge failed to start')
        try:
            with socket.create_connection((bind, target[1]), timeout=.2):
                return
        except OSError:
            time.sleep(.1)
    child.terminate()
    raise RuntimeError('Local gateway bridge did not become ready')


def serve(base):
    target = local_target(base)
    if not target:
        return
    host, port = target
    network = json.loads(subprocess.check_output(['docker', 'network', 'inspect', 'bridge']))[0]
    bind = network['IPAM']['Config'][0]['Gateway']
    if not ipaddress.ip_address(bind).is_private:
        raise ValueError('Expected private inner Docker bridge')

    class Handler(http.server.BaseHTTPRequestHandler):
        def log_message(self, *args):
            pass

        def forward(self):
            if not self.path.startswith('/') or self.path.startswith('//'):
                self.send_error(400)
                return
            try:
                length = int(self.headers.get('Content-Length', '0'))
                if length < 0 or length > 2 * 1024 * 1024 or self.headers.get('Transfer-Encoding'):
                    self.send_error(413)
                    return
                headers = {k: v for k, v in self.headers.items()
                           if k.lower() not in ('host', 'connection', 'transfer-encoding')}
                headers['Host'] = f'host.docker.internal:{port}'
                upstream = http.client.HTTPConnection(host, port, timeout=125)
                upstream.request(self.command, self.path, self.rfile.read(length), headers)
                result = upstream.getresponse()
                body = result.read()
                self.send_response(result.status)
                if result.getheader('Content-Type'):
                    self.send_header('Content-Type', result.getheader('Content-Type'))
                self.send_header('Content-Length', str(len(body)))
                self.end_headers()
                self.wfile.write(body)
                upstream.close()
            except (ValueError, OSError, http.client.HTTPException):
                self.send_error(502, 'Local gateway unavailable')

        do_GET = do_POST = do_DELETE = do_PATCH = forward

    # No wildcard/interface binding and no Docker port publication. Only the
    # isolated workspace's inner trial network can connect to this listener.
    http.server.ThreadingHTTPServer((bind, port), Handler).serve_forever()


if __name__ == '__main__':
    serve(sys.argv[1])
