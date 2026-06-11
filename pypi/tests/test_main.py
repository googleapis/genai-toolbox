import os
import sys
from unittest import mock
import pytest
from toolbox_server import main

def test_get_binary_path_returns_existing_file():
    path = main.get_binary_path()
    assert os.path.exists(path)
    assert os.path.isfile(path)

def test_get_binary_path_returns_executable_file():
    # Catches a missing `chmod +x` in the wheel-build pipeline — the file
    # would still exist but `subprocess.run` would fail with permission denied.
    path = main.get_binary_path()
    assert os.access(path, os.X_OK), f"{path} is not executable"

def test_run_help(monkeypatch):
    # Mock sys.argv to run with --help
    monkeypatch.setattr(sys, "argv", ["toolbox-server", "--help"])
    
    # We expect sys.exit(0)
    with pytest.raises(SystemExit) as excinfo:
        main.run()
    assert excinfo.value.code == 0

def test_run_error_handling(monkeypatch):
    # Mock get_binary_path to raise FileNotFoundError
    with mock.patch("toolbox_server.main.get_binary_path", side_effect=FileNotFoundError("Mocked binary not found")):
        with pytest.raises(SystemExit) as excinfo:
            main.run()
        assert excinfo.value.code == 1
