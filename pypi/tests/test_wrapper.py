import os
import sys
import subprocess
from unittest import mock
import pytest
from toolbox_server import main

def test_get_binary_path():
    path = main.get_binary_path()
    assert os.path.exists(path)
    assert os.path.isfile(path)

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
