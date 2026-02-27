# @xpla/contracts

A collection of smart contracts for the CONX Chain.  
The published package includes precompile interface sources (`.sol`) and ABIs (`.json`).

## Installation

```bash
# pnpm
pnpm add @xpla/contracts

# npm
npm install @xpla/contracts

# yarn
yarn add @xpla/contracts
```

## Package structure

After installation, use the following paths:

| Path | Description |
|------|-------------|
| `@xpla/contracts/precompiles/` | Solidity sources (`.sol`) |
| `@xpla/contracts/abi/precompiles/` | ABI-only JSON (`.json`) |

Included precompiles: `auth`, `bank`, `wasm`.

## Usage

### Loading ABI (ethers / viem / web3, etc.)

```javascript
import IAuthAbi from "@xpla/contracts/abi/precompiles/auth/IAuth.sol/IAuth.json" assert { type: "json" };

// or Node
const IAuthAbi = require("@xpla/contracts/abi/precompiles/auth/IAuth.sol/IAuth.json");
```

### Using interfaces in Hardhat

Import by package path in your contract:

```solidity
import "@xpla/contracts/precompiles/auth/IAuth.sol";
import "@xpla/contracts/precompiles/bank/IBank.sol";
import "@xpla/contracts/precompiles/wasm/IWasm.sol";
```

### Using interfaces in Foundry

Add the following to `remappings.txt` for shorter import paths:

```
@xpla/contracts/=node_modules/@xpla/contracts/precompiles/
```

```solidity
import "@xpla/contracts/auth/IAuth.sol";
import "@xpla/contracts/bank/IBank.sol";
import "@xpla/contracts/wasm/IWasm.sol";
```

### Path reference

- Interface ABI: `@xpla/contracts/abi/precompiles/{module}/{Interface}.sol/{Interface}.json`  
  e.g. `abi/precompiles/auth/IAuth.sol/IAuth.json`
- Sources: `@xpla/contracts/precompiles/{module}/{Interface}.sol`