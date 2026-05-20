// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";
import {IERC20} from "openzeppelin-contracts/contracts/token/ERC20/IERC20.sol";

interface IUniswapLiquiditySetup {
    function createPoolAndAddLiquidity(address tokenA, uint256 amount) external payable;
}

contract AddLiquidityScript is Script {
    function run() external {
        uint256 privateKey = vm.envUint("PRIVATE_KEY");
        address deployer = vm.addr(privateKey);

        address liquiditySetup = vm.envOr(
            "LIQUIDITY_SETUP_ADDRESS",
            address(0x3d58314c60e2f768a3Bb83eC0903B7b562951bDd)
        );
        address gameToken = vm.envOr(
            "GAMETOKEN_ADDRESS",
            address(0xCFF4475F8D171449899E983292c463314aBdF79c)
        );
        uint256 tokenAmount = vm.envOr("TOKEN_AMOUNT", uint256(100 ether));
        uint256 ethAmount = vm.envOr("ETH_AMOUNT", uint256(1 ether));

        vm.startBroadcast(privateKey);

        // transfer GameToken to LiquiditySetup so it can approve the Router
        IERC20(gameToken).transfer(liquiditySetup, tokenAmount);

        // create pool (if needed) and add liquidity with ETH
        IUniswapLiquiditySetup(liquiditySetup).createPoolAndAddLiquidity{value: ethAmount}(
            gameToken,
            tokenAmount
        );

        vm.stopBroadcast();

        console.log("Pool created / liquidity added:");
        console.log("  Deployer:", deployer);
        console.log("  LiquiditySetup:", liquiditySetup);
        console.log("  GameToken:", gameToken);
        console.log("  Token amount:", tokenAmount);
        console.log("  ETH amount:", ethAmount);
    }
}
