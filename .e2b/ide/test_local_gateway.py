import unittest
from unittest.mock import patch
from local_gateway import local_target, ensure_bridge


class LocalGatewayTests(unittest.TestCase):
    def test_hosted_gateway_never_starts_local_listener(self):
        with patch('local_gateway.subprocess.Popen') as start:
            ensure_bridge('https://playground-runtime.example/api/v1/')
            start.assert_not_called()

    def test_only_known_local_host_can_be_bridged(self):
        with patch('local_gateway.socket.gethostbyname', return_value='192.168.65.254'):
            self.assertEqual(local_target('http://host.docker.internal:18102/api/v1/'),
                             ('192.168.65.254', 18102))
        self.assertIsNone(local_target('http://other.example:18102/'))
        for base in ['http://host.docker.internal:8080/',
                     'http://user:password@host.docker.internal:18102/',
                     'http://host.docker.internal:18102/?redirect=evil']:
            with self.assertRaises(ValueError):
                local_target(base)


if __name__ == '__main__':
    unittest.main()
