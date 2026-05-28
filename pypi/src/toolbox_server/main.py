import os
import platform
import subprocess
import sys
from importlib import resources
import toolbox_server

def get_binary_path():
    """Locates the embedded Go binary within the wheel."""
    system = platform.system()
    bin_name = "toolbox.exe" if system == "Windows" else "toolbox"

    try:
        binary_resource = resources.files(toolbox_server) / "bin" / bin_name
        with resources.as_file(binary_resource) as executable_path:
            if not executable_path.exists():
                 raise FileNotFoundError(f"Binary not found at {executable_path}")
            return str(executable_path)
    except Exception as e:
        raise FileNotFoundError(f"Could not locate binary {bin_name} for {system}: {e}")

def run():
    """Executes the embedded Go binary."""
    if sys.version_info < (3, 9):
        print("Error: toolbox-server requires Python 3.9 or higher.", file=sys.stderr)
        sys.exit(1)

    try:
        binary_path = get_binary_path()
    except FileNotFoundError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

    if not os.access(binary_path, os.X_OK):
         os.chmod(binary_path, 0o755)

    try:
        # Run the binary and pass through all command-line arguments.
        # We use subprocess.run since it is simpler and handles process wait automatically.
        result = subprocess.run([binary_path] + sys.argv[1:])
        sys.exit(result.returncode)
    except KeyboardInterrupt:
        print("Toolbox execution interrupted.", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"Error running toolbox binary: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    run()
