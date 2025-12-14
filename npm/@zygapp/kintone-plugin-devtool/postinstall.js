#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

const platform = process.platform;
const arch = process.arch;

let platformDir;
if (platform === "darwin") {
  platformDir = arch === "arm64" ? "darwin-arm64" : "darwin-x64";
} else if (platform === "linux") {
  platformDir = arch === "arm64" ? "linux-arm64" : "linux-x64";
} else if (platform === "win32") {
  platformDir = arch === "arm64" ? "win32-arm64" : "win32-x64";
} else {
  console.error(`Unsupported platform: ${platform}`);
  process.exit(1);
}

const binaryName = platform === "win32" ? "kpdev.exe" : "kpdev";
const binaryPath = path.join(__dirname, "bin", platformDir, binaryName);

if (!fs.existsSync(binaryPath)) {
  console.error(`Binary not found for ${platform}-${arch}`);
  console.error(`Expected path: ${binaryPath}`);
  process.exit(1);
}

// バイナリパスを記録
const binaryPathFile = path.join(__dirname, "bin", ".binary-path");
fs.writeFileSync(binaryPathFile, binaryPath);

console.log(`kpdev installed for ${platform}-${arch}`);
