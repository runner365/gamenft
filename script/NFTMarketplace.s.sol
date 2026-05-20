// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";
import "../src/NFTMarketplace.sol";

contract NFTMarketplaceScript is Script {
    function run() external {
        uint256 privateKey = vm.envUint("PRIVATE_KEY");
        address gameToken = vm.envOr(
            "GAMETOKEN_ADDRESS",
            address(0xCFF4475F8D171449899E983292c463314aBdF79c)
        );

        vm.startBroadcast(privateKey);

        NFTMarketplace marketplace = new NFTMarketplace(gameToken);

        vm.stopBroadcast();

        console.log("NFTMarketplace deployed at:", address(marketplace));
        console.log("  GameToken:", gameToken);
    }
}
