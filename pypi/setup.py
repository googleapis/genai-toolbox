from setuptools import setup, find_packages
import os
import platform
import urllib.request
import stat
import shutil
import sys

try:
    from wheel.bdist_wheel import bdist_wheel as _bdist_wheel
    class bdist_wheel(_bdist_wheel):
        def finalize_options(self):
            _bdist_wheel.finalize_options(self)
            self.root_is_pure = False
            self.root_is_purelib = False
except ImportError:
    print("Warning: wheel package not found, platform tag might be incorrect.")
    bdist_wheel = None

def get_platform_details():
    system = platform.system()
    machine = platform.machine()
    os_part = system.lower()
    arch_part = ""

    if os_part == "darwin" and machine == "x86_64":
        arch_part = "amd64"
    elif os_part == "darwin" and machine == "arm64":
        arch_part = "arm64"
    elif os_part == "linux" and machine == "x86_64":
        arch_part = "amd64"
    elif os_part == "windows" and machine == "AMD64":
        arch_part = "amd64"
    else:
        raise OSError(f"Unsupported platform: {system}-{machine}")
    return os_part, arch_part

def get_version():
    init_py = os.path.join(os.path.dirname(__file__), "src", "toolbox_server", "__init__.py")
    if os.path.exists(init_py):
        with open(init_py, "r") as f:
            for line in f:
                if line.startswith("__version__"):
                    return line.split("=")[1].strip().strip('"').strip("'")
    raise RuntimeError(f"Could not find version in {init_py}")

# Ensure LICENSE is present in the package directory (inherent from root repo)
setup_dir = os.path.dirname(os.path.abspath(__file__))
parent_license = os.path.join(setup_dir, "..", "LICENSE")
local_license = os.path.join(setup_dir, "LICENSE")
if os.path.exists(parent_license):
    shutil.copy2(parent_license, local_license)

def download_binary():
    version = os.environ.get("TOOLBOX_VERSION")
    if not version:
        ver = get_version()
        version = f"v{ver}" if not ver.startswith("v") else ver

    os_part, arch_part = get_platform_details()
    bin_name = "toolbox.exe" if os_part == "windows" else "toolbox"

    url = f"https://storage.googleapis.com/mcp-toolbox-for-databases/{version}/{os_part}/{arch_part}/{bin_name}"
    dest_dir = "src/toolbox_server/bin"

    if os.path.exists(dest_dir):
        shutil.rmtree(dest_dir)
    os.makedirs(dest_dir, exist_ok=True)

    dest_path = os.path.join(dest_dir, bin_name)

    print(f"Downloading {url} to {dest_path}")
    try:
        urllib.request.urlretrieve(url, dest_path)
    except urllib.error.HTTPError as e:
        print(f"ERROR: Failed to download {url}: {e.code} {e.reason}", file=sys.stderr)
        raise SystemExit(f"Failed to download binary from {url}")
    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        raise SystemExit("Binary download failed.")

    st = os.stat(dest_path)
    os.chmod(dest_path, st.st_mode | stat.S_IEXEC)
    return bin_name

binary_name = download_binary()

setup(
    packages=find_packages(where="src"),
    package_dir={"": "src"},
    package_data={
        "toolbox_server": ["bin/*"],
    },
    include_package_data=True,
    cmdclass={'bdist_wheel': bdist_wheel} if bdist_wheel else {},
)
