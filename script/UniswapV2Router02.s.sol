// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";

contract UniswapV2Router02Script is Script {
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");

        // WETH on Sepolia: 0xfFf9976782d46CC05630D1f6eBAb18b2324d6B14
        address weth = vm.envOr("WETH_ADDRESS", address(0xfFf9976782d46CC05630D1f6eBAb18b2324d6B14));
        address factory = vm.envOr("FACTORY_ADDRESS", address(0));
        require(factory != address(0), "FACTORY_ADDRESS must be set (deploy UniswapV2Factory first)");

        bytes memory args = abi.encode(factory, weth);
        bytes memory bytecode = abi.encodePacked(
            vm.getCode("UniswapV2Router02.sol:UniswapV2Router02"),
            args
        );

        vm.startBroadcast(deployerPrivateKey);

        address router;
        assembly {
            router := create(0, add(bytecode, 0x20), mload(bytecode))
        }
        require(router != address(0), "deployment failed");

        vm.stopBroadcast();

        console.log("UniswapV2Router02 deployed at:", router);
        console.log("Factory:", factory);
        console.log("WETH:", weth);
    }
}
