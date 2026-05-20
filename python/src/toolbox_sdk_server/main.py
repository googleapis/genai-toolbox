import os
import platform
import subprocess
import sys
from importlib import resources

def get_binary_path():
    """Locates the embedded Go binary for the current platform."""
    system = platform.system()
    machine = platform.machine()
    package_path = "toolbox_sdk_server.bin"

    if system == "Windows":
        bin_name = "toolbox.exe"
    else:
        # Linux and Darwin
        bin_name = "toolbox"

    try:
        # Use resources.files() for a more modern approach if available,
        # but resources.path() is fine.
        with resources.path(package_path, bin_name) as path:
            return str(path)
    except FileNotFoundError:
        raise FileNotFoundError(f"Could not find binary {bin_name} for {system}-{machine} in {package_path}")

def run():
    """Executes the embedded Go binary."""
    try:
        binary_path = get_binary_path()
    except FileNotFoundError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

    if not os.access(binary_path, os.X_OK):
        try:
            os.chmod(binary_path, 0o755)
        except OSError as e:
            print(f"Error setting execute permission on {binary_path}: {e}", file=sys.stderr)
            sys.exit(1)

    try:
        process = subprocess.Popen([binary_path] + sys.argv[1:])
        process.wait()
        sys.exit(process.returncode)
    except KeyboardInterrupt:
        print("Interrupted by user", file=sys.stderr)
        sys.exit(1)
    except OSError as e:
        print(f"Error executing binary {binary_path}: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    run()

