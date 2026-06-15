"""Build configuration for the toolbox-server PyPI package.

Each wheel embeds a single Go binary and is tagged for a single platform. The
caller is responsible for staging the binary and declaring the target platform
before running `python -m build`:

  1. Place the matching binary at src/toolbox_server/bin/toolbox (or
     toolbox.exe for Windows wheels). On Unix wheels it must be marked
     executable (chmod +x).
  2. Set TOOLBOX_PLATFORM to the PEP 425 platform tag for the wheel:
     "manylinux2014_x86_64", "macosx_11_0_arm64", "macosx_10_14_x86_64",
     "win_amd64", or "win_arm64".

The wheel is always tagged py3 / none / <plat> since it ships no Python code
that depends on a specific interpreter ABI.
"""

import os
import shutil

from setuptools import setup, find_packages

# Canonical path as of setuptools 70.1+ (pinned in pyproject.toml). Letting
# this fail loud is intentional — without our bdist_wheel overrides, the wheel
# would be silently mis-tagged as pure-Python and break for every user.
from setuptools.command.bdist_wheel import bdist_wheel as _bdist_wheel


SETUP_DIR = os.path.dirname(os.path.abspath(__file__))


class bdist_wheel(_bdist_wheel):
    def finalize_options(self):
        super().finalize_options()
        # Tell bdist_wheel this package contains a platform-specific binary
        # (not just Python), so the wheel gets a real platform tag instead
        # of the default "any". Required for our get_tag override to take
        # effect.
        self.root_is_pure = False

    def get_tag(self):
        # Override the default "infer the platform tag from the build
        # machine" behavior so one Linux Cloud Build VM can produce wheels
        # for all 5 platforms by varying TOOLBOX_PLATFORM per invocation.
        plat = os.environ.get("TOOLBOX_PLATFORM")
        if not plat:
            raise SystemExit(
                "TOOLBOX_PLATFORM env var is required (e.g., "
                "'manylinux2014_x86_64', 'macosx_11_0_arm64'). "
                "See setup.py docstring."
            )
        return "py3", "none", plat


# Ship the root LICENSE inside the package.
parent_license = os.path.join(SETUP_DIR, "..", "LICENSE")
local_license = os.path.join(SETUP_DIR, "LICENSE")
if os.path.exists(parent_license):
    shutil.copy2(parent_license, local_license)


setup(
    packages=find_packages(where="src"),
    package_dir={"": "src"},
    package_data={
        "toolbox_server": ["bin/*"],
    },
    include_package_data=True,
    cmdclass={"bdist_wheel": bdist_wheel},
)
