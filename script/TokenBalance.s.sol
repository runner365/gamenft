// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";

interface IERC20 {
    function balanceOf(address account) external view returns (uint256);
}

contract TokenBalanceScript is Script {
    function run() external {
        address token = vm.envOr(
            "TOKEN_ADDRESS",
            address(0xCFF4475F8D171449899E983292c463314aBdF79c)
        );
        address who = vm.envOr(
            "ADDRESS",
            vm.addr(vm.envUint("PRIVATE_KEY"))
        );

        uint256 bal = IERC20(token).balanceOf(who);

        console.log("Token:", token);
        console.log("Address:", who);
        console.log("Balance (raw):", bal);
        console.log("Balance (ether):", bal / 1e18);
    }
}
