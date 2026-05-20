// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";

interface IUniswapV2Factory {
    function getPair(address tokenA, address tokenB) external view returns (address);
}

interface IUniswapV2Pair {
    function getReserves() external view returns (uint112 reserve0, uint112 reserve1, uint32 blockTimestampLast);
    function token0() external view returns (address);
    function token1() external view returns (address);
    function totalSupply() external view returns (uint256);
}

contract QueryLiquidityScript is Script {
    function run() external {
        address factory = vm.envOr(
            "FACTORY_ADDRESS",
            address(0xF62c03E08ada871A0bEb309762E260a7a6a880E6)
        );
        address gameToken = vm.envOr(
            "GAMETOKEN_ADDRESS",
            address(0xCFF4475F8D171449899E983292c463314aBdF79c)
        );
        address weth = vm.envOr(
            "WETH_ADDRESS",
            address(0xfFf9976782d46CC05630D1f6eBAb18b2324d6B14)
        );

        IUniswapV2Factory fac = IUniswapV2Factory(factory);
        address pair = fac.getPair(gameToken, weth);

        console.log("=== Liquidity Pool Info ===");
        console.log("Factory:", factory);
        console.log("GameToken:", gameToken);
        console.log("WETH:", weth);

        if (pair == address(0)) {
            console.log("PAIR: NOT FOUND");
            return;
        }
        console.log("Pair:", pair);

        IUniswapV2Pair p = IUniswapV2Pair(pair);
        address t0 = p.token0();
        address t1 = p.token1();
        (uint256 r0, uint256 r1,) = p.getReserves();
        uint256 supply = p.totalSupply();

        // token0 = WETH (0x7b79... < 0xDD24...)
        uint256 reserveWeth = t0 == weth ? r0 : r1;
        uint256 reserveGmtk = t0 == weth ? r1 : r0;

        console.log("token0:", t0 == weth ? "WETH" : "GMTK");
        console.log("token1:", t1 == weth ? "WETH" : "GMTK");
        console.log("Reserve WETH (wei):", reserveWeth);
        console.log("Reserve GMTK (wei):", reserveGmtk);
        console.log("TotalSupply (LP):", supply);

        // k = x * y
        console.log("k:", reserveWeth * reserveGmtk);

        // spot price: GMTK per 1 ETH
        console.log("Price (GMTK/ETH):", reserveGmtk * 1e18 / reserveWeth);
    }
}
