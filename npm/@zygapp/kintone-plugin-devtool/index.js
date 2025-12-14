#!/usr/bin/env node

const { spawn } = require("child_process");
const path = require("path");
const fs = require("fs");

const binaryPathFile = path.join(__dirname, "bin", ".binary-path");

let binaryPath;
if (fs.existsSync(binaryPathFile)) {
  binaryPath = fs.readFileSync(binaryPathFile, "utf-8").trim();
} else {
  // フォールバック: 現在のプラットフォームから推測
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
  binaryPath = path.join(__dirname, "bin", platformDir, binaryName);
}

if (!fs.existsSync(binaryPath)) {
  console.error(`Binary not found: ${binaryPath}`);
  console.error("Please reinstall the package.");
  process.exit(1);
}

const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: "inherit",
});

child.on("error", (err) => {
  console.error(`Failed to start: ${err.message}`);
  process.exit(1);
});

child.on("close", (code) => {
  process.exit(code || 0);
});
