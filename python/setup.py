from setuptools import setup, find_packages
import os
import platform
import urllib.request
import stat
import shutil
from wheel.bdist_wheel import bdist_wheel as _bdist_wheel

# Mark the wheel as platform-specific
class bdist_wheel(_bdist_wheel):
    def finalize_options(self):
        _bdist_wheel.finalize_options(self)
        self.root_is_purelib = False

def get_platform_details():
    system = platform.system()
    machine = platform.machine()
    if system == "Linux":
        return "linux", "amd64", "toolbox"
    elif system == "Darwin":
        return "darwin", machine, "toolbox"
    elif system == "Windows":
        return "windows", "amd64", "toolbox.exe"
    raise OSError(f"Unsupported platform: {system}-{machine}")

def download_binary():
    version = os.environ.get("TOOLBOX_VERSION", "v0.32.0") # Fixed version for now
    os_part, arch_part, bin_name = get_platform_details()

    url = f"https://storage.googleapis.com/mcp-toolbox-for-databases/{version}/{os_part}/{arch_part}/{bin_name}"
    dest_dir = "src/toolbox_sdk_server/bin"

    if os.path.exists(dest_dir):
        shutil.rmtree(dest_dir)
    os.makedirs(dest_dir, exist_ok=True)

    dest_path = os.path.join(dest_dir, bin_name)

    print(f"Downloading {url} to {dest_path}")
    urllib.request.urlretrieve(url, dest_path)

    st = os.stat(dest_path)
    os.chmod(dest_path, st.st_mode | stat.S_IEXEC)
    return bin_name

# Download the binary before building
binary_name = download_binary()

try:
    from wheel.bdist_wheel import bdist_wheel as _bdist_wheel
    class bdist_wheel(_bdist_wheel):
        def finalize_options(self):
            _bdist_wheel.finalize_options(self)
            self.root_is_pure = False
except ImportError:
    bdist_wheel = None

setup(
    packages=find_packages(where="src"),
    package_dir={"": "src"},
    package_data={
        "toolbox_sdk_server": [f"bin/{binary_name}"],
    },
    include_package_data=True,
    cmdclass={'bdist_wheel': bdist_wheel},
)

