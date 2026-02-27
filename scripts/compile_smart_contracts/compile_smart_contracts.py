"""
Compile Solidity smart contracts in this repository with Hardhat.

Prerequisite: contracts/solidity directory must exist.

Usage:
    python3 compile_smart_contracts.py --compile   # compile all under precompile/
    python3 compile_smart_contracts.py --clean
"""

import json
import os
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from shutil import copy, rmtree
from typing import List, Union

# The path to the main level of the Cosmos EVM repository.
REPO_PATH = Path(__file__).parent.parent.parent


# Only scan precompile for Solidity contracts.
PRECOMPILE_DIR = "precompile"

# Target directory for Hardhat: contracts/solidity (must exist).
HARDHAT_PROJECT_DIR = "contracts"
SOLIDITY_SOURCE = "solidity"
RELATIVE_TARGET = Path(HARDHAT_PROJECT_DIR) / SOLIDITY_SOURCE
CONTRACTS_TARGET = REPO_PATH / RELATIVE_TARGET


# This list contains all files that should be ignored when scanning the
# repository for Solidity files.
IGNORED_FILES: List[str] = [
    # Ignored because it uses a different OpenZeppelin contracts version to
    # compile
    "ERC20Minter_OpenZeppelinV5.sol",
    # Ignore vendored contracts pulled in through package managers
    r"node_modules/.*\.sol$",
    r"lib/.*\.sol$",
    # Ignore all other test contracts outside of tests/contracts
    r"tests/(?!contracts/).*\.sol$",
    # Ignore Foundry-style script and test contracts
    r".*\.s\.sol$",
    r".*\.t\.sol$",
]


# Solidity files that only contain structs/type definitions and are imported
# by other contracts. They do not produce a standalone compiled artifact.
CONTRACTS_WITHOUT_ARTIFACT: List[str] = [
    "Types",
]

# This list contains all folders that should be ignored when scanning the
# repository for Solidity files.
IGNORED_FOLDERS: List[str] = [
    "nix_tests",
    "node_modules",
    "scripts",
    "tests/solidity",
    # We don't want to copy anything that has already been copied into the
    # contracts subdirectory, but we do want to have the ones stored there originally.
    rf"{RELATIVE_TARGET}/\w+",
]


@dataclass
class Contract:
    """
    Dataclass to store the name and path of a Solidity contract
    as well as the path to where the compiled JSON data is stored.
    """

    compiled_json_path: Union[Path, None]
    filename: str
    path: Path
    relative_path: Path


def find_solidity_contracts(repo_path: Path) -> List[Contract]:
    """Finds all Solidity files under repo_path/precompile. ABI is written back as {ContractName}.json in each folder."""
    scan_root = repo_path / PRECOMPILE_DIR
    if not scan_root.is_dir():
        raise ValueError(
            f"Precompile directory not found: {scan_root}. "
            "This script only compiles contracts under precompile/."
        )

    solidity_files: List[Contract] = []

    for root, _, files in os.walk(scan_root):
        if is_ignored_folder(root):
            continue

        relative_path = Path(root).relative_to(scan_root)

        for file in files:
            if is_ignored_file(Path(root) / file, repo_path):
                continue

            if re.search(r"(?!\.dbg)\.sol$", file):
                filename = os.path.splitext(file)[0]
                root_path = Path(root)
                # Always write compiled ABI back as {ContractName}.json (e.g. IAuth.json, IBank.json)
                compiled_json_path = root_path / f"{filename}.json"

                solidity_files.append(
                    Contract(
                        filename=filename,
                        path=Path(os.path.join(root, file)),
                        relative_path=relative_path,
                        compiled_json_path=compiled_json_path,
                    )
                )

    return solidity_files


def is_ignored_folder(path: str) -> bool:
    """
    Check if the folder is in the list of ignored folders.
    """

    return any(re.search(folder, path) for folder in IGNORED_FOLDERS)


def is_ignored_file(file_path: Path, repo_path: Path) -> bool:
    """
    Check if the file should be ignored based on IGNORED_FILES patterns.
    """
    relative_path = file_path.relative_to(repo_path)
    file_name = file_path.name

    for ignored_pattern in IGNORED_FILES:
        # Check if it's a regex pattern (contains regex metacharacters)
        if any(char in ignored_pattern for char in r".*+?^${}[]|()\\"):
            if re.search(ignored_pattern, str(relative_path)):
                return True
        else:
            # Simple filename match
            if file_name == ignored_pattern:
                return True

    return False


def copy_to_contracts_directory(target_dir: Path, contracts: List[Contract]) -> None:
    """Copy found contracts into target_dir (contracts/solidity). Assumes target_dir exists."""
    if not target_dir.is_dir():
        raise ValueError(
            f"Target directory does not exist: {target_dir}. "
            "Create contracts/solidity first."
        )

    for contract in contracts:
        sub_dir = target_dir / contract.relative_path
        if is_relative_target(contract.relative_path):
            continue
        sub_dir.mkdir(parents=True, exist_ok=True)
        copy(contract.path, sub_dir)


def is_os_repo(path: Path) -> bool:
    """
    This function checks if the given path is the root of the Cosmos EVM repository,
    where this script is designed to be executed.
    """

    contents = os.listdir(path)

    if "go.mod" not in contents:
        return False

    with open(path / "go.mod", "r", encoding="utf-8") as go_mod:
        while True:
            line = go_mod.readline()
            if not line:
                break

            if "module github.com/xpladev/xpla" in line:
                return True

    return False


def compile_contracts_in_dir(target_dir: Path):
    """
    This function compiles the Solidity contracts in the target directory
    with Hardhat.
    """

    cur_dir = os.getcwd()

    # Change to the root directory of the hardhat setup to compile.
    os.chdir(target_dir.parent)
    if not os.path.exists("hardhat.config.ts"):
        raise ValueError("compilation can only work in a HardHat setup")

    install_failed = os.system("pnpm install")
    if install_failed:
        raise ValueError("Failed to install pnpm packages.")

    compilation_failed = os.system("pnpm exec hardhat compile")
    if compilation_failed:
        raise ValueError("Failed to compile Solidity contracts.")

    os.chdir(cur_dir)


def copy_compiled_contracts_back_to_source(
    contracts: List[Contract],
    compiled_dir: Path,
):
    """
    This function checks if the given contracts have
    been compiled in the compilation target directory
    and copies those back, that have a corresponding JSON
    file found originally.
    """

    for contract in contracts:
        if contract.compiled_json_path is None:
            continue

        # Skip files that only contain structs/imports and do not produce an artifact
        if contract.filename in CONTRACTS_WITHOUT_ARTIFACT:
            continue

        if is_relative_target(contract.relative_path):
            dir_with_json = compiled_dir
        else:
            dir_with_json = compiled_dir / contract.relative_path

        compiled_path = (
            dir_with_json / f"{contract.filename}.sol" / f"{contract.filename}.json"
        )

        if not os.path.exists(compiled_path):
            print(f"Path: {compiled_path}")
            print(f"-> did not find compiled JSON file for {contract.filename}")
            continue

        # Extract ABI only from Hardhat artifacts
        # Hardhat artifacts are objects with an 'abi' field, but we need just the ABI array
        with open(compiled_path, 'r', encoding='utf-8') as f:
            artifact = json.load(f)
            # If it's an artifact object, extract just the ABI
            if isinstance(artifact, dict) and 'abi' in artifact:
                abi_only = artifact['abi']
                with open(contract.compiled_json_path, 'w', encoding='utf-8') as out:
                    json.dump(abi_only, out, indent=2)
            else:
                # If it's already an ABI array, copy as-is
                copy(compiled_path, contract.compiled_json_path)


def clean_up_hardhat_project(hardhat_dir: Path):
    """
    This function removes the build artifacts as well as the downloaded
    node modules from the Hardhat project folder.
    Also, the file that have been copied to the contracts directory are deleted.
    """

    node_modules = hardhat_dir / "node_modules"
    if os.path.exists(node_modules):
        rmtree(hardhat_dir / "node_modules")

    artifacts = hardhat_dir / "artifacts"
    if os.path.exists(artifacts):
        rmtree(artifacts)

    cache = hardhat_dir / "cache"
    if os.path.exists(cache):
        rmtree(cache)

    contracts_dir = hardhat_dir / SOLIDITY_SOURCE
    for entry in contracts_dir.iterdir():
        if entry.is_dir():
            rmtree(entry)


def is_relative_target(path: Path) -> bool:
    """
    Checks if the given path is the target directory,
    where the contracts are copied to.
    """

    return path == RELATIVE_TARGET


def compile_files(repo_path: Path):
    """Compiles all Solidity contracts under precompile/ with Hardhat."""
    found_contracts = find_solidity_contracts(repo_path)
    copy_to_contracts_directory(CONTRACTS_TARGET, found_contracts)
    compile_contracts_in_dir(CONTRACTS_TARGET)
    copy_compiled_contracts_back_to_source(
        found_contracts, CONTRACTS_TARGET.parent / "artifacts" / SOLIDITY_SOURCE
    )


if __name__ == "__main__":
    if not is_os_repo(REPO_PATH):
        raise ValueError(
            f"This script must be run from the xpla repo root. Current path: {REPO_PATH}"
        )

    if len(sys.argv) != 2:
        raise ValueError(
            "Wrong usage. Use --compile or --clean.",
        )

    if sys.argv[1] == "--compile":
        compile_files(REPO_PATH)

    elif sys.argv[1] == "--clean":
        clean_up_hardhat_project(CONTRACTS_TARGET.parent)

    else:
        raise ValueError(
            "Wrong usage, please refer to the README of this script",
        )
