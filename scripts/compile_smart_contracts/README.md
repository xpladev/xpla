# Compiling Smart Contracts

`make contracts-compile` compiles the precompile interfaces and all suites under
`tests/solidity/`. Each project uses its existing compiler and dependency versions.
This command only compiles contracts; it does not start a chain or run tests.
Node.js 24 and pnpm 9.15.0 are used by CI for these builds.

## Usage

To compile the smart contracts, run the following command:

```bash
make contracts-compile
```

The generated files are written to:

- `contracts/artifacts/`: Hardhat artifacts for the precompile interfaces. The
  Python script also extracts their ABI arrays into `precompile/`.
- `tests/solidity/suites/{precompiles,revert_cases}/artifacts/`: Hardhat test
  artifacts, including each contract's ABI and deployment bytecode.
- `tests/solidity/suites/{basic,eip1559,exception,opcode}/build/contracts/`:
  Truffle test artifacts, when the suite contains Solidity contracts.

To compile only the Solidity test suites:

```bash
make test-contracts-compile
```

`make test` runs this compilation before the Go tests. The ICS20 integration and
multichain tests read the generated Hardhat artifacts directly. When running
`go test` manually, run `make test-contracts-compile` first, including after
changing Solidity sources. Missing artifacts fail the tests with this command
in the error message.

To clean up the `contracts/` project's generated artifacts, installed dependencies
and cached files, run:

```bash
make contracts-clean
```
