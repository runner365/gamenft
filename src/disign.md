# 合约系统设计
## 1. GameToken.sol - 游戏代币合约ERC20
类型: ERC20 + ERC20Burnable + Ownable

核心功能:
* 创建游戏内交易代币（支持铸造 mint）
* 管理用户代币余额和转账
* 持有者可销毁自己的代币（burn）
* 合约拥有者可增发代币

## 2. NftERC721.sol - 游戏道具合约ERC721
类型: ERC721URIStorage + Ownable

核心功能:
* 铸造游戏道具NFT（safeMint，仅owner）
* 管理道具归属（tokenId → owner）
* 支持道具销毁（burn）
* 支持设置道具元数据URI（_setTokenURI）

## 3. UniswapLiquiditySetup.sol - 流动性管理合约UniswapV2
类型: UniswapV2Router02 + UniSwapV2PairFactory + Ownable + ReentrancyGuard（防重入）

核心功能:
* 📥 添加流动性: 创建GameToken/WETH交易对并注入流动性
* 📤 移除流动性: 销毁LP代币取回资产
* 💱 ETH→GameToken: 任何人可调用，用ETH购买GameToken
* 💱 GameToken→ETH: 仅owner可调用，将GameToken兑换为ETH（可能用于项目方回笼资金）
* 紧急提取功能（withdrawETH、sweepERC20）

## 4. NFTMarketplace.sol - 道具交易市场
类型: 普通合约（依赖IERC20/IERC721/IERC1155接口），支持 ERC721 + ERC1155

核心功能:
* 📦 ERC721 上架: listERC721 - 检查所有权、授权市场合约、创建挂单
* 💰 ERC721 购买: buyERC721 - 检查余额、转移GameToken、转移NFT、自动下架
* 📦 ERC1155 上架: listERC1155 - 按数量上架，需 setApprovalForAll
* 💰 ERC1155 购买: buyERC1155 - 支持部分购买，买完后自动下架，未买完减少挂单数量
* ❌ 取消挂单: cancelListing - 卖家可取消激活的挂单

事件:
* ItemListed(nftContract, tokenId, seller, amount, price)
* ItemBought(nftContract, tokenId, buyer, amount, totalPrice)

## 5. GameItems.sol - 游戏道具合约ERC1155
类型: ERC1155 + Ownable

核心功能:
* 🎮 管理三种游戏道具: knife(id=1) / pistol(id=2) / bomb(id=3)
* 🔨 铸造道具: mint(to, id, amount) — onlyOwner，支持 maxSupply 限量
* 🔥 销毁道具: burn(from, id, amount) — 持有者或授权者可销毁
* 📊 总量追踪: totalSupply[id]、maxSupply[id]
* 🔗 URI 格式: {baseURI}{id}.json

与 ERC721 的区别:
* ERC721 (NftERC721): 1 tokenId → 1 owner，适合独有道具
* ERC1155 (GameItems): 1 tokenId → N 个持有者各有数量，适合可堆叠游戏道具

## 用户购买 ERC721 道具流程
1. 用户 → UniswapLiquiditySetup.swapExactETHForTokens() → 用ETH兑换GameToken
2. 用户 → NFTMarketplace.buyERC721() → 用GameToken购买独有NFT
3. NFTMarketplace → 转移GameToken给卖家
4. NFTMarketplace → 转移NFT给买家

## 用户购买 ERC1155 道具流程
1. 用户 → UniswapLiquiditySetup.swapExactETHForTokens() → 用ETH兑换GameToken
2. 用户 → NFTMarketplace.buyERC1155(nftContract, tokenId, amount) → 按数量购买
3. NFTMarketplace → 转移 GameToken * amount 给卖家
4. NFTMarketplace → 调用 safeTransferFrom 转移 amount 个 ERC1155 token
5. 若买完，自动下架；若部分购买，减少挂单数量

## 用户出售 ERC1155 道具流程
1. 用户 → GameItems.setApprovalForAll(marketplace, true) → 授权市场
2. 用户 → NFTMarketplace.listERC1155(nftContract, tokenId, amount, price) → 上架
3. 买家购买后 → 自动获得 GameToken 收入
4. 卖家 → UniswapLiquiditySetup.swapExactTokensForETH() → 将GameToken兑换为ETH（需owner权限）