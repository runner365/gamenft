> English | [中文](README.md)

A blockchain-based Tetris game running on the Sepolia testnet.

Core gameplay: Players play Tetris in the browser, earn rewards (knife, pistol, bomb) based on game performance, and list items on the marketplace contract for trading. Earned NFTs can be listed and traded on the on-chain marketplace using GameToken, and GameToken can be swapped with ETH via Uniswap.



# GameNFT — Project Overview

Features:
* A Tetris game where players can play on the Sepolia network and earn reward NFTs, including knife, pistol, and bomb.
* The game reward NFT contract is deployed on the Sepolia network. Players can play on the Sepolia network and earn reward NFTs.
* Players can list their earned NFTs on the marketplace contract for trading. The marketplace contract supports ERC-1155 NFTs.
* The marketplace contract supports ERC-1155 NFTs, and players can list their earned NFTs on the marketplace for trading.

## Main Contracts
* GameToken.sol: The game token based on the ERC-20 standard. Players need to swap ETH for GameToken on Uniswap. GameToken can also be swapped back to ETH on Uniswap.
  1) Players can swap GameToken on Uniswap
  2) Players can use GameToken to purchase unique NFTs
  3) Players can list NFTs on the marketplace contract for trading
* GameItems.sol: The game reward NFT contract based on the ERC-1155 standard. Players can play on the Sepolia network and earn reward NFTs.
  Players can play on the Sepolia network and earn reward NFTs. For example: a user at a certain address earns a knife in the game, and it is immediately minted on-chain to the GameItems contract.
* NFTMarketplace.sol: The game's NFT marketplace contract based on the ERC-1155 standard. Players can list NFTs on the marketplace for trading.
  1) Players can list NFTs on the marketplace for trading. For example: a user lists 3 knives for sale at a price of 100 GameToken.
  2) Players can buy NFTs on the marketplace. For example: a user buys 3 knives, paying 100 GameToken.
* UniswapLiquiditySetup.sol: The game's Uniswap liquidity setup contract. Players can swap GameToken on Uniswap.

## Backend Service (Go)
* Listens to blockchain events: Monitors the TransferSingle event from the GameItems contract, retrieves records of players earning NFTs, and stores them in the database.
* Provides API endpoints: Offers an API for players to query their NFT records.
* Database: Uses SQLite to store player NFT records. The database file is gamenft.db.

## Frontend Application (Vue 3) — GameNFT
* Tetris game: Players can play Tetris in the browser and earn on-chain ERC-1155 NFT rewards (knife, pistol, bomb) based on game performance.
* NFT marketplace: Players can browse all listed NFTs on the marketplace page, make purchases and sales.
* Players can list their own NFTs for trading on the marketplace page.
* Players can buy NFTs listed by other players on the marketplace page.
* Players can view their own NFT records on the marketplace page.

Note: The wallet used is Phantom Wallet.
Players need to create a wallet in Phantom Wallet before they can play the game and trade.
