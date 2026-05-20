// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";
import "../src/GameItems.sol";

contract GameItemsScript is Script {
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        address deployer = vm.addr(deployerPrivateKey);

        string memory baseURI = vm.envOr("BASE_URI", string(""));

        vm.startBroadcast(deployerPrivateKey);

        GameItems gameItems = new GameItems(baseURI);

        vm.stopBroadcast();

        console.log("GameItems deployed at:", address(gameItems));
        console.log("Owner:", deployer);
        console.log("BaseURI:", baseURI);
        console.log("Token IDs: KNIFE=1, PISTOL=2, BOMB=3");
    }
}
