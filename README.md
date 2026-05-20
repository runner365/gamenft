# Solidity contracts
项目功能：
* 俄罗斯方块游戏，玩家可以在Sepolia网络上进行游戏，获得奖励NFT，包括刀、手枪、炸弹。
* 游戏奖励NFT的合约地址在Sepolia网络上部署，玩家可以在Sepolia网络上进行游戏，获得奖励NFT。
* 玩家可以将获得的NFT上架到市场合约进行交易，市场合约支持ERC1155类型的NFT。
* 市场合约支持ERC1155类型的NFT，玩家可以将获得的NFT上架到市场合约进行交易。

## 主要合约
* GameToken.sol:GameToken是游戏的代币，基于ERC20标准，玩家需要用ETH在Uniswap上兑换GameToken。GameToken可以在Uniswap上兑换为ETH。
1）玩家可以在Uniswap上兑换GameToken
2）玩家可以用GameToken购买独特NFT
3）玩家可以在市场合约上架NFT进行交易
* GameItems.sol:GameItems是游戏的奖励NFT合约，基于ERC1155标准，玩家可以在Sepolia网络上进行游戏，获得奖励NFT。
玩家可以在Sepolia网络上进行游戏，获得奖励NFT。如：某address用户在游戏中获得了一把刀，立刻上链到GameItems合约中。
* NFTMarketplace.sol:NFTMarketplace是游戏的NFT市场合约，基于ERC1155标准，玩家可以在市场合约上架NFT进行交易。
1）玩家可以在市场合约上架NFT进行交易，如：某address用户上架了3把刀，价格为100GameToken。
2）玩家可以在市场合约上购买NFT进行交易，如：某address用户购买了3把刀，支付了100GameToken。
* UniswapLiquiditySetup.sol:UniswapLiquiditySetup是游戏的Uniswap流动性合约，玩家可以在Uniswap上兑换GameToken。



