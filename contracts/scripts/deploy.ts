import { ethers } from "hardhat";

async function main() {
  console.log("Deploying RelayNodeRegistry contract...");

  const RelayNodeRegistry = await ethers.getContractFactory("RelayNodeRegistry");
  const relayNodeRegistry = await RelayNodeRegistry.deploy();

  await relayNodeRegistry.waitForDeployment();

  const address = await relayNodeRegistry.getAddress();
  console.log("RelayNodeRegistry deployed to:", address);

  // Verify the deployment
  console.log("Contract deployed successfully!");
  console.log("Network:", (await ethers.provider.getNetwork()).name);
  console.log("Deployer:", (await ethers.getSigners())[0].address);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
