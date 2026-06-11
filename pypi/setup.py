"""Build configuration for the toolbox-server PyPI package.

Each wheel embeds a single Go binary and is tagged for a single platform. The
caller is responsible for staging the binary before running `python -m build`:

  1. Place the matching binary at src/toolbox_server/bin/toolbox (or
     toolbox.exe for Windows wheels). On Unix wheels it must be marked
     executable (chmod +x).
  2. Optionally set TOOLBOX_PLATFORM to the PEP 425 platform tag for the
     wheel, e.g. "manylinux2014_x86_64", "macosx_11_0_arm64",
     "macosx_10_14_x86_64", "win_amd64", "win_arm64". If unset, the wheel is
     tagged for the host platform (useful for local development).

The wheel is always tagged py3 / none / <plat> since it ships no Python code
that depends on a specific interpreter ABI.
"""

import os
import platform
import shutil

from setuptools import setup, find_packages

try:
    from wheel.bdist_wheel import bdist_wheel as _bdist_wheel
except ImportError:
    _bdist_wheel = None


def _host_platform_tag():
    """PEP 425 platform tag for the current host."""
    system = platform.system().lower()
    machine = platform.machine().lower()
    if system == "linux" and machine == "x86_64":
        return "manylinux2014_x86_64"
    if system == "darwin" and machine == "arm64":
        return "macosx_11_0_arm64"
    if system == "darwin" and machine == "x86_64":
        return "macosx_10_14_x86_64"
    if system == "windows" and machine in ("amd64", "x86_64"):
        return "win_amd64"
    if system == "windows" and machine == "arm64":
        return "win_arm64"
    raise OSError(f"Unsupported host platform: {system}-{machine}")


if _bdist_wheel is not None:

    class bdist_wheel(_bdist_wheel):
        def finalize_options(self):
            super().finalize_options()
            self.root_is_pure = False

        def get_tag(self):
            plat = os.environ.get("TOOLBOX_PLATFORM") or _host_platform_tag()
            return "py3", "none", plat

else:
    bdist_wheel = None


# Ship the root LICENSE inside the package.
setup_dir = os.path.dirname(os.path.abspath(__file__))
parent_license = os.path.join(setup_dir, "..", "LICENSE")
local_license = os.path.join(setup_dir, "LICENSE")
if os.path.exists(parent_license):
    shutil.copy2(parent_license, local_license)


# Refuse to build a wheel without an embedded binary — the wheel would be
# silently broken at runtime otherwise.
bin_dir = os.path.join(setup_dir, "src", "toolbox_server", "bin")
binaries = []
if os.path.isdir(bin_dir):
    binaries = [
        name for name in ("toolbox", "toolbox.exe")
        if os.path.isfile(os.path.join(bin_dir, name))
    ]
if not binaries:
    raise SystemExit(
        f"No toolbox binary found in {bin_dir}. Stage one before running "
        "`python -m build` (see setup.py docstring)."
    )


setup(
    packages=find_packages(where="src"),
    package_dir={"": "src"},
    package_data={
        "toolbox_server": ["bin/*"],
    },
    include_package_data=True,
    cmdclass={"bdist_wheel": bdist_wheel} if bdist_wheel else {},
)
