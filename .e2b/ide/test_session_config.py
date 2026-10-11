import tempfile
import unittest
from pathlib import Path
from session_config import provision_cli


class SessionConfigTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.session = dict(RELAI_API_BASE='https://gateway.example/api/v1/',
                            RELAI_TEAM_API_KEY='temporary-session-test-token',
                            PLAYGROUND_API_BASE='https://gateway.example/playground-api',
                            PLAYGROUND_SESSION_ID='test-session')

    def test_bootstrap_nested_runtime_without_provider_keys(self):
        env = provision_cli(self.root, self.session)
        self.assertEqual(env['AIRLINE_MODEL_PROVIDER'], 'relai')
        self.assertEqual(env['RELAI_API_KEY'], self.session['RELAI_TEAM_API_KEY'])
        for name in ('config.toml', 'simulator.env'):
            file = self.root / '.relai' / name
            self.assertEqual(file.stat().st_mode & 0o777, 0o600)
            self.assertNotIn('OPENAI_API_KEY', file.read_text())
        text = (self.root / '.relai/simulator.env').read_text()
        self.assertIn('AIRLINE_MODEL_PROVIDER=relai\n', text)
        self.assertIn('RELAI_API_URL=https://gateway.example/api/v1/\n', text)
        self.assertNotIn('PLAYGROUND_SESSION_ID', text)
        provision_cli(self.root, {**self.session, 'RELAI_TEAM_API_KEY': 'replacement-session'})
        self.assertNotIn('temporary-session-test-token', (self.root / '.relai/simulator.env').read_text())

    def test_rejects_incomplete_and_injected_environment(self):
        for session in ({}, {**self.session, 'RELAI_TEAM_API_KEY': 'token\nOPENAI_API_KEY=bad'}):
            with self.assertRaises(ValueError):
                provision_cli(self.root, session)

    def test_rejects_private_file_and_parent_symlinks(self):
        target = self.root / 'untouched'
        target.write_text('preserve')
        folder = self.root / '.relai'
        folder.mkdir()
        (folder / 'config.toml').symlink_to(target)
        with self.assertRaises(ValueError):
            provision_cli(self.root, self.session)
        self.assertEqual(target.read_text(), 'preserve')
        (folder / 'config.toml').unlink()
        folder.rmdir()
        folder.symlink_to(self.root, target_is_directory=True)
        with self.assertRaises(ValueError):
            provision_cli(self.root, self.session)


if __name__ == '__main__':
    unittest.main()
