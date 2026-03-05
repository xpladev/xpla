# @xpla/contracts

A collection of smart contracts for the CONX Chain.  
The published package includes precompile interface sources (`.sol`) and ABIs as typed TypeScript (`.ts`).

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
| `@xpla/contracts/abi/precompiles/` | ABI as typed ESM (`.ts`) |

Included precompiles: `auth`, `bank`, `wasm`.

## Usage

### Loading ABI with TypeScript / viem (typed)

Import the named ABI constant so that `functionName`, `args`, and return types are inferred:

```typescript
import { createPublicClient, http } from "viem";
import { IAuth_ABI } from "@xpla/contracts/abi/precompiles/auth/IAuth";

const client = createPublicClient({ transport: http() });

// functionName and args are type-checked and autocompleted
const result = await client.readContract({
  address: "0x...",
  abi: IAuth_ABI,
  functionName: "accounts",
  args: ["0x..."],
});
```

Use the same pattern for other precompiles, e.g. `IBank_ABI` from `abi/precompiles/bank/IBank`, `IWasm_ABI` from `abi/precompiles/wasm/IWasm`, etc.

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

- Interface ABI: `@xpla/contracts/abi/precompiles/{module}/{Interface}.ts`  
  e.g. `abi/precompiles/auth/IAuth`
- Sources: `@xpla/contracts/precompiles/{module}/{Interface}.sol`
