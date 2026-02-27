/**
 * prepack: Transform repo layout (solidity/precompiles/) into package layout (precompiles/)
 * postpack: Restore original layout
 *
 * Published package exposes paths like @xpla/contracts/precompiles/auth/IAuth.sol
 */
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, "..");
const BACKUP_DIR = path.join(ROOT, ".publish-backup");

function getPrecompileSubdirs() {
  const precompilesDir = path.join(ROOT, "solidity", "precompiles");
  if (!fs.existsSync(precompilesDir)) return [];
  return fs.readdirSync(precompilesDir).filter((name) => {
    const full = path.join(precompilesDir, name);
    return fs.statSync(full).isDirectory();
  });
}

function rmrf(dir) {
  if (!fs.existsSync(dir)) return;
  for (const name of fs.readdirSync(dir)) {
    const full = path.join(dir, name);
    if (fs.statSync(full).isDirectory()) rmrf(full);
    else fs.unlinkSync(full);
  }
  fs.rmdirSync(dir);
}

function cpDir(src, dest) {
  if (!fs.existsSync(src)) return;
  fs.mkdirSync(dest, { recursive: true });
  for (const name of fs.readdirSync(src)) {
    const s = path.join(src, name);
    const d = path.join(dest, name);
    if (fs.statSync(s).isDirectory()) cpDir(s, d);
    else fs.copyFileSync(s, d);
  }
}

/** Extract only the abi field from artifact JSON and write to destDir preserving path structure */
function copyArtifactsAsAbiOnly(srcDir, destDir) {
  if (!fs.existsSync(srcDir)) return;
  fs.mkdirSync(destDir, { recursive: true });
  for (const name of fs.readdirSync(srcDir)) {
    const srcPath = path.join(srcDir, name);
    const destPath = path.join(destDir, name);
    if (fs.statSync(srcPath).isDirectory()) {
      copyArtifactsAsAbiOnly(srcPath, destPath);
    } else if (name.endsWith(".json")) {
      const artifact = JSON.parse(fs.readFileSync(srcPath, "utf8"));
      const abi = artifact && typeof artifact.abi !== "undefined" ? artifact.abi : artifact;
      fs.writeFileSync(destPath, JSON.stringify(abi, null, 2));
    }
  }
}

function transform() {
  const solidityDir = path.join(ROOT, "solidity");
  const precompilesSrcDir = path.join(solidityDir, "precompiles");
  const artifactsDir = path.join(ROOT, "artifacts");
  const artifactsPrecompilesSrcDir = path.join(
    artifactsDir,
    "solidity",
    "precompiles"
  );

  if (!fs.existsSync(precompilesSrcDir)) {
    console.error("Run in contracts/ with solidity/precompiles/ present.");
    process.exit(1);
  }
  if (!fs.existsSync(artifactsPrecompilesSrcDir)) {
    console.error(
      "Run pnpm run compile first so artifacts/solidity/precompiles/ exists."
    );
    process.exit(1);
  }

  const DIRS = getPrecompileSubdirs();
  if (DIRS.length === 0) {
    console.error("No subdirs in solidity/precompiles/.");
    process.exit(1);
  }

  fs.mkdirSync(BACKUP_DIR, { recursive: true });
  cpDir(solidityDir, path.join(BACKUP_DIR, "solidity"));
  cpDir(artifactsDir, path.join(BACKUP_DIR, "artifacts"));
  fs.copyFileSync(path.join(ROOT, "package.json"), path.join(BACKUP_DIR, "package.json"));

  const precompilesDestDir = path.join(ROOT, "precompiles");
  fs.mkdirSync(precompilesDestDir, { recursive: true });
  for (const dir of DIRS) {
    const src = path.join(precompilesSrcDir, dir);
    const dest = path.join(precompilesDestDir, dir);
    if (fs.existsSync(src)) cpDir(src, dest);
  }

  // ABI only: copy artifacts/solidity/precompiles → abi/precompiles (store just the abi array, not full artifact)
  const abiPrecompilesDestDir = path.join(ROOT, "abi", "precompiles");
  fs.mkdirSync(abiPrecompilesDestDir, { recursive: true });
  copyArtifactsAsAbiOnly(artifactsPrecompilesSrcDir, abiPrecompilesDestDir);

  const pkgPath = path.join(ROOT, "package.json");
  const pkg = JSON.parse(fs.readFileSync(pkgPath, "utf8"));
  pkg.files = ["precompiles/**/*.sol", "abi/precompiles/**/*.json"];
  pkg.scripts = { ...pkg.scripts, prepack: "true", postpack: "" };
  fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + "\n");
}

function restore() {
  if (!fs.existsSync(BACKUP_DIR)) return;

  const precompilesDir = path.join(ROOT, "precompiles");
  if (fs.existsSync(precompilesDir)) rmrf(precompilesDir);

  const abiDir = path.join(ROOT, "abi");
  if (fs.existsSync(abiDir)) rmrf(abiDir);

  const artifactsDir = path.join(ROOT, "artifacts");
  if (fs.existsSync(artifactsDir)) rmrf(artifactsDir);

  const solidityDir = path.join(ROOT, "solidity");
  if (fs.existsSync(solidityDir)) rmrf(solidityDir);

  cpDir(path.join(BACKUP_DIR, "solidity"), path.join(ROOT, "solidity"));
  cpDir(path.join(BACKUP_DIR, "artifacts"), path.join(ROOT, "artifacts"));
  fs.copyFileSync(path.join(BACKUP_DIR, "package.json"), path.join(ROOT, "package.json"));

  rmrf(BACKUP_DIR);
}

const isRestore = process.argv.includes("--restore");
if (isRestore) restore();
else transform();
