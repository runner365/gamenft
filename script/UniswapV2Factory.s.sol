// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";

contract UniswapV2FactoryScript is Script {
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        address deployer = vm.addr(deployerPrivateKey);

        address feeToSetter = vm.envOr("FEE_TO_SETTER", deployer);

        bytes memory args = abi.encode(feeToSetter);
        bytes memory bytecode = abi.encodePacked(
            vm.getCode("UniswapV2Factory.sol:UniswapV2Factory"),
            args
        );

        vm.startBroadcast(deployerPrivateKey);

        address factory;
        assembly {
            factory := create(0, add(bytecode, 0x20), mload(bytecode))
        }
        require(factory != address(0), "deployment failed");

        vm.stopBroadcast();

        console.log("UniswapV2Factory deployed at:", factory);
        console.log("feeToSetter:", feeToSetter);
    }
}
