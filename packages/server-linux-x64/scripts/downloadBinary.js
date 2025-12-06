const fs = require('fs');
const path = require('path');
const https = require('https');
const { execSync } = require('child_process');

// 1. Configuration
const PLATFORM_MAP = {
  'linux': 'linux',
  'darwin': 'darwin',
  'win32': 'windows'
};

const ARCH_MAP = {
  'x64': 'amd64',
  'arm64': 'arm64'
};

const args = process.argv.slice(2);
if (args.length < 2) {
  console.error("Usage: node download-binary.js <platform> <arch>");
  process.exit(1);
}

const [targetPlatform, targetArch] = args;

// 2. Determine Version
const version = fs.readFileSync(path.join(process.cwd(), 'version.txt'), 'utf8').trim(); 

// 3. Construct URL
const gcsPlatform = PLATFORM_MAP[targetPlatform];
const gcsArch = ARCH_MAP[targetArch];

if (!gcsPlatform || !gcsArch) {
  console.error(`Unsupported platform/arch: ${targetPlatform}/${targetArch}`);
  process.exit(1);
}

const extension = targetPlatform === 'win32' ? '.exe' : '';
const binaryName = `toolbox${extension}`;
const url = `https://storage.googleapis.com/genai-toolbox/v${version}/${gcsPlatform}/${gcsArch}/${binaryName}`;

// 4. Prepare Output
const binDir = path.join(process.cwd(), 'bin');
if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}
const destPath = path.join(binDir, binaryName);

if (fs.existsSync(destPath)) {
  console.log(`[Skipped] Binary already exists at ${destPath}`);
  process.exit(0);
}

console.log(`[Prepack] Downloading ${binaryName} for ${targetPlatform}/${targetArch}...`);
console.log(`[Source]  ${url}`);

// 5. Download Function
const file = fs.createWriteStream(destPath);
https.get(url, function(response) {
  if (response.statusCode !== 200) {
    console.error(`❌ Failed to download. Status Code: ${response.statusCode}`);
    fs.unlink(destPath, () => {}); // Delete partial file
    process.exit(1);
  }

  response.pipe(file);

  file.on('finish', () => {
    file.close(() => {
      // 6. Make executable (Unix only)
      if (targetPlatform !== 'win32') {
        try {
          execSync(`chmod +x "${destPath}"`);
        } catch (err) {
          console.warn("⚠️  Could not set executable permissions (chmod failed).");
        }
      }
      console.log(`✅ Success! Binary saved to ${destPath}`);
    });
  });
}).on('error', function(err) {
  fs.unlink(destPath, () => {});
  console.error(`❌ Download Error: ${err.message}`);
  process.exit(1);
});