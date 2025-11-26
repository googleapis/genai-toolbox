#!/usr/bin/env node
const { spawn } = require('child_process');
const path = require('path');
const os = require('os');

const PLATFORMS = {
  'darwin-arm64': 'toolbox-darwin-arm64',
  'darwin-x64': 'toolbox-darwin-x64',
  'linux-x64': 'toolbox-linux-x64',
  'win32-x64': 'toolbox-win32-x64'
};

const currentKey = `${os.platform()}-${os.arch()}`;
const pkgName = PLATFORMS[currentKey];

if (!pkgName) {
  console.error(`Unsupported platform: ${currentKey}`);
  process.exit(1);
}

let binPath;
try {
  const pkgJsonPath = require.resolve(`${pkgName}/package.json`);
  const pkgDir = path.dirname(pkgJsonPath);
  const binName = os.platform() === 'win32' ? 'toolbox.exe' : 'toolbox';
  binPath = path.join(pkgDir, 'bin', binName);
} catch (e) {
  console.error(`Binary for ${currentKey} not found. Installation failed?`);
  process.exit(1);
}

spawn(binPath, process.argv.slice(2), { stdio: 'inherit' })
  .on('exit', process.exit);
