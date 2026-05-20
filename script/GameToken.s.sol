// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";
import "../src/GameToken.sol";

contract GameTokenScript is Script {
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        address deployer = vm.addr(deployerPrivateKey);

        uint256 initialSupply = vm.envOr("INITIAL_SUPPLY", uint256(1000 * 10**18));

        vm.startBroadcast(deployerPrivateKey);

        GameToken token = new GameToken("GameToken", "GMTK", initialSupply);

        vm.stopBroadcast();

        console.log("GameToken deployed at:", address(token));
        console.log("Owner:", deployer);
        console.log("Name:", token.name());
        console.log("Symbol:", token.symbol());
        console.log("Initial supply:", token.totalSupply());
    }
}
