import type { HardhatUserConfig } from "hardhat/config";

const config: HardhatUserConfig = {
  solidity: '0.8.22',
  paths: {
    sources: './solidity'
  }
}

export default config;