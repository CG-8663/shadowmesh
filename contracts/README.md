# ShadowMesh Smart Contracts

Ethereum smart contracts for the ShadowMesh relay node registry system.

## Overview

This directory contains the Solidity smart contracts that manage the decentralized relay node registry for ShadowMesh. The contracts handle:

- Relay node registration and deregistration
- Node heartbeat monitoring
- Staking mechanisms
- On-chain verification of node status

## Project Structure

```
contracts/
├── contracts/           # Solidity smart contracts
│   └── RelayNodeRegistry.sol
├── scripts/             # Deployment scripts
│   └── deploy.ts
├── test/                # Hardhat test files
│   └── RelayNodeRegistry.test.ts
├── hardhat.config.ts    # Hardhat configuration
└── package.json         # Node.js dependencies
```

## Setup

### Prerequisites

- Node.js v18 or higher
- npm or yarn

### Installation

```bash
npm install
```

### Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

Required environment variables:

- `SEPOLIA_RPC_URL` - Ethereum Sepolia testnet RPC endpoint
- `MAINNET_RPC_URL` - Ethereum mainnet RPC endpoint
- `PRIVATE_KEY` - Deployer wallet private key (DO NOT COMMIT)
- `ETHERSCAN_API_KEY` - Etherscan API key for contract verification

## Development

### Compile Contracts

```bash
npm run compile
```

### Run Tests

```bash
npm test
```

### Test Coverage

```bash
npm run coverage
```

### Clean Build Artifacts

```bash
npm run clean
```

## Deployment

### Local/Hardhat Network

```bash
npm run deploy:local
```

### Sepolia Testnet

```bash
npm run deploy:sepolia
```

### Ethereum Mainnet

```bash
npm run deploy:mainnet
```

## Contract Verification

After deploying to a public network, verify the contract on Etherscan:

```bash
# For Sepolia
npm run verify:sepolia -- <CONTRACT_ADDRESS>

# For Mainnet
npm run verify:mainnet -- <CONTRACT_ADDRESS>
```

## Technology Stack

- **Hardhat**: Development environment and task runner
- **TypeScript**: Type-safe contract interactions and scripts
- **OpenZeppelin**: Security-audited contract libraries
- **Ethers.js v6**: Ethereum library for contract interaction
- **Chai**: Testing assertions

## Smart Contracts

### RelayNodeRegistry

Main contract for managing relay nodes. Current status: **Placeholder**

**Planned Features:**
- `registerNode()` - Register a new relay node with stake
- `submitHeartbeat()` - Submit periodic heartbeat to prove liveness
- `deregisterNode()` - Remove a relay node and return stake
- `getRegisteredNodeCount()` - Query total registered nodes

See contract source code for detailed documentation.

## Development Roadmap

- [x] Epic 1, Story 1.1: Hardhat project initialization
- [ ] Epic 1, Story 1.2: Implement RelayNodeRegistry contract
- [ ] Epic 1, Story 1.3: Deploy to Sepolia testnet
- [ ] Epic 1, Story 1.4: Security audit and mainnet deployment

## License

MIT License - See LICENSE file for details

## Contributing

This is part of the ShadowMesh project. For contribution guidelines, see the main project README.
