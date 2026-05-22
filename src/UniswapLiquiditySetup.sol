// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {IUniswapV2Router02} from "lib/v2-periphery/contracts/interfaces/IUniswapV2Router02.sol";
import {IUniswapV2Factory} from "lib/v2-core/contracts/interfaces/IUniswapV2Factory.sol";
import {IUniswapV2Pair} from "lib/v2-core/contracts/interfaces/IUniswapV2Pair.sol";
import {IERC20} from "lib/openzeppelin-contracts/contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "lib/openzeppelin-contracts/contracts/token/ERC20/utils/SafeERC20.sol";
import {Ownable} from "lib/openzeppelin-contracts/contracts/access/Ownable.sol";
import {ReentrancyGuard} from "lib/openzeppelin-contracts/contracts/security/ReentrancyGuard.sol";

contract UniswapLiquiditySetup is Ownable, ReentrancyGuard {
    using SafeERC20 for IERC20;
    address public immutable WETH;
    IUniswapV2Router02 public immutable router;
    IUniswapV2Factory public immutable factory;

    // Mainnet Uniswap V2 Router: 0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D
    // On Sepolia, deploy your own Factory + Router first (see script/).
    constructor(address _router) Ownable(){
        require(_router != address(0), "Router must be a valid address");
        router = IUniswapV2Router02(_router);
        factory = IUniswapV2Factory(router.factory());
        WETH = router.WETH();
    }

    // Accept ETH refunds and plain ETH transfers from router or other contracts
    receive() external payable {}

    function createPoolAndAddLiquidity(address tokenA, uint256 amount) external payable onlyOwner nonReentrant {
        require(tokenA != address(0), "TokenA must be a valid address");
        require(msg.value > 0, "Must send ETH to add liquidity");
        require(amount > 0, "tokenA amount must not be zero");
        address token0;
        address token1;

        (token0, token1) = _sortTokens(tokenA, WETH);
        address pair = factory.getPair(token0, token1);
        if (pair == address(0)) {
            factory.createPair(token0, token1);
        }
        IERC20(tokenA).approve(address(router), 0);
        IERC20(tokenA).approve(address(router), amount);
        uint256 minTokenAmount = amount * 950 / 1000; 
        uint256 minETHAmount = msg.value * 950 / 1000;

        router.addLiquidityETH{value: msg.value}(
            tokenA,
            amount,
            minTokenAmount,
            minETHAmount,
            msg.sender,
            block.timestamp + 600
        );
    }

    function removeLiquidity(address tokenA, uint256 liquidity) external onlyOwner nonReentrant {
        require(tokenA != address(0), "TokenA must be a valid address");
        require(liquidity > 0, "Liquidity amount must be greater than zero");

        address token0;
        address token1;
        (token0, token1) = _sortTokens(tokenA, WETH);
        address pair = factory.getPair(token0, token1);
        require(pair != address(0), "Pool does not exist");

        IERC20(pair).approve(address(router), 0);
        IERC20(pair).approve(address(router), liquidity);
        // get the number of WETH which is same as the number of tokenA in the pool, so we can calculate the minimum amounts to remove
        uint256 totalSupply = IERC20(pair).totalSupply();
        (uint256 reserveToken, uint256 reserveWeth,) = IUniswapV2Pair(pair).getReserves();

        if (reserveToken == 0 || reserveWeth == 0) {
            revert("Pool reserves are empty");
        }
        if (token0 != tokenA) {
            (reserveToken, reserveWeth) = (reserveWeth, reserveToken);
        }
        uint256 minTokenAmount = liquidity * reserveToken / totalSupply * 950 / 1000;
        uint256 minETHAmount = liquidity * reserveWeth / totalSupply * 950 / 1000;

        router.removeLiquidityETH(
            tokenA,
            liquidity,
            minTokenAmount,
            minETHAmount,
            msg.sender,
            block.timestamp + 600
        );
    }

    // input Eth and output tokenA, so the path is WETH -> tokenA, everyone can call this function to swap, not only the owner
    function swapExactETHForTokens(address tokenA, uint256 amountOutMin) external payable nonReentrant {
        require(tokenA != address(0), "TokenA must be a valid address");
        require(msg.value > 0, "Must send ETH to swap");
        address[] memory path = new address[](2);
        path[0] = WETH;
        path[1] = tokenA;

        router.swapExactETHForTokens{value: msg.value}(
            amountOutMin,
            path,
            msg.sender,
            block.timestamp + 600
        );
    }

    // input tokenA and output Eth, so the path is tokenA -> WETH, only the owner can call this function to swap
    function swapExactTokensForETH(address tokenA, uint256 amountIn, uint256 amountOutMin) external onlyOwner nonReentrant {
        require(tokenA != address(0), "TokenA must be a valid address");
        require(amountIn > 0, "AmountIn must be greater than zero");
        address[] memory path = new address[](2);
        path[0] = tokenA;
        path[1] = WETH;

        IERC20(tokenA).approve(address(router), 0);
        IERC20(tokenA).approve(address(router), amountIn);

        router.swapExactTokensForETH(
            amountIn,
            amountOutMin,
            path,
            owner(),
            block.timestamp + 600
        );
    }

    // Withdraw any ETH balance held by this contract to the owner
    function withdrawETH() external onlyOwner nonReentrant {
        uint256 balance = address(this).balance;
        require(balance > 0, "No ETH to withdraw");
        (bool sent, ) = owner().call{value: balance}("");
        require(sent, "ETH transfer failed");
    }

    // Sweep ERC20 token balance from this contract to the owner
    function sweepERC20(address token) external onlyOwner nonReentrant {
        require(token != address(0), "Invalid token");
        uint256 bal = IERC20(token).balanceOf(address(this));
        require(bal > 0, "No token balance");
        IERC20(token).safeTransfer(owner(), bal);
    }

    function _sortTokens(address tokenA, address tokenB) private pure returns (address token1, address token2) {
        require(tokenA != address(0), "TokenA must be a valid address");
        require(tokenB != address(0), "TokenB must be a valid address");

        if (tokenA < tokenB) {
            return (tokenA, tokenB);
        } else {
            return (tokenB, tokenA);
        }
    }
}
