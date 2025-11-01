import { expect } from "chai";
import { ethers } from "hardhat";
import { RelayNodeRegistry } from "../typechain-types";
import { SignerWithAddress } from "@nomicfoundation/hardhat-ethers/signers";

describe("RelayNodeRegistry", function () {
  let relayNodeRegistry: RelayNodeRegistry;
  let owner: SignerWithAddress;
  let addr1: SignerWithAddress;
  let addr2: SignerWithAddress;

  beforeEach(async function () {
    // Get signers
    [owner, addr1, addr2] = await ethers.getSigners();

    // Deploy the contract
    const RelayNodeRegistryFactory = await ethers.getContractFactory("RelayNodeRegistry");
    relayNodeRegistry = await RelayNodeRegistryFactory.deploy();
    await relayNodeRegistry.waitForDeployment();
  });

  describe("Deployment", function () {
    it("Should set the right owner", async function () {
      expect(await relayNodeRegistry.owner()).to.equal(owner.address);
    });

    it("Should start with zero registered nodes", async function () {
      expect(await relayNodeRegistry.getRegisteredNodeCount()).to.equal(0);
    });
  });

  describe("Node Registration", function () {
    it("Should revert when registering a node (not implemented)", async function () {
      const nodeId = ethers.keccak256(ethers.toUtf8Bytes("node1"));
      await expect(
        relayNodeRegistry.registerNode(nodeId, { value: ethers.parseEther("1.0") })
      ).to.be.revertedWith("Not implemented yet");
    });
  });

  describe("Heartbeat", function () {
    it("Should revert when submitting heartbeat (not implemented)", async function () {
      const nodeId = ethers.keccak256(ethers.toUtf8Bytes("node1"));
      await expect(
        relayNodeRegistry.submitHeartbeat(nodeId)
      ).to.be.revertedWith("Not implemented yet");
    });
  });

  describe("Node Deregistration", function () {
    it("Should revert when deregistering a node (not implemented)", async function () {
      const nodeId = ethers.keccak256(ethers.toUtf8Bytes("node1"));
      await expect(
        relayNodeRegistry.deregisterNode(nodeId)
      ).to.be.revertedWith("Not implemented yet");
    });
  });
});
