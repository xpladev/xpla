const HDWalletProvider = require('@truffle/hdwallet-provider');

module.exports = {
  networks: {
    // Development network is just left as truffle's default settings
    cosmos: {
      provider: () => new HDWalletProvider({
        privateKeys: [
          '88CBEAD91AEE890D27BF06E003ADE3D4E952427E88F88D31D61D3EF5E5D54305',
          '3B7955D25189C99A7468192FCBC6429205C158834053EBE3F78F4512AB432DB9',
          'e9b1d63e8acd7fe676acb43afb390d4b0202dab61abec9cf2a561e4becb147de'
        ],
        providerOrUrl: 'http://127.0.0.1:8545'
      }),
      network_id: '*', // Any network (default: none)
      gas: 5000000, // Gas sent with each transaction
      gasPrice: 8000000000000, // 8000 gwei (in wei)
      skipDryRun: true
    }
  },
  compilers: {
    solc: {
      version: '0.5.17'
    }
  }
}
