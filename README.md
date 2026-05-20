> 中文 | [English](README_en.md)

一个基于 Sepolia 区块链俄罗斯方块游戏.

核心玩法： 玩家在浏览器玩俄罗斯方块，根据游戏表现获得奖励（刀、手枪、炸弹）, 并将道具上架到市场合约进行交易。获得的 NFT 可以在链上市场用 GameToken 上架交易，GameToken 与 ETH 之间可通过 Uniswap 互相兑换。


# GameNFT — 项目概述

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

## 后端服务(golang)
* 监听区块链事件：监听GameItems合约的TransferSingle事件，获取玩家获得NFT的记录，并存储到数据库中。
* 提供API接口：提供玩家查询NFT记录的接口，玩家可以通过接口查询自己的NFT记录。
* 数据库：使用SQLite数据库存储玩家NFT记录，数据库文件为gamenft.db。

## 前端应用(Vue3)-GameNFT
* 俄罗斯方块游戏：玩家可以在浏览器玩俄罗斯方块，根据游戏表现获得链上ERC-1155 NFT奖励（刀、手枪、炸弹）。
* NFT市场：玩家可以在市场页面查看所有上架的NFT，进行购买和销售。
* 玩家可以在市场页面上架自己的NFT进行交易。
* 玩家可以在市场页面购买其他玩家上架的NFT进行交易。
* 玩家可以在市场页面查看自己的NFT记录。

注意：钱包采用Phantom Wallet。
玩家需要在Phantom Wallet中创建一个钱包，才能进行游戏和交易。

