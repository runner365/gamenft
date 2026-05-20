#!/usr/bin/env bash
set -euo pipefail

RPC_URL="${RPC_URL:-https://ethereum-sepolia.publicnode.com}"
GAS_LIMIT="${GAS_LIMIT:-6000000}"

usage() {
    echo "Usage: $0 <contract>"
    echo "  contract: GameItems | GameToken | NFTMarketplace | UniswapLiquiditySetup | AddLiquidity | TokenBalance | QueryLiquidity"
    exit 1
}

[ $# -eq 1 ] || usage

case "$1" in
    GameItems)
        echo "=== Deploying GameItems ==="
        forge script script/GameItems.s.sol:GameItemsScript \
            --rpc-url "$RPC_URL" \
            --broadcast --verify --chain sepolia \
            --gas-limit "$GAS_LIMIT"
        ;;
    GameToken)
        echo "=== Deploying GameToken ==="
        forge script script/GameToken.s.sol:GameTokenScript \
            --rpc-url "$RPC_URL" \
            --broadcast --verify --chain sepolia \
            --gas-limit "$GAS_LIMIT"
        ;;
    NFTMarketplace)
        echo "=== Deploying NFTMarketplace ==="
        [ -z "${PRIVATE_KEY:-}" ] && { echo "PRIVATE_KEY must be set"; exit 1; }
        export PRIVATE_KEY
        forge script script/NFTMarketplace.s.sol:NFTMarketplaceScript \
            --rpc-url "$RPC_URL" \
            --broadcast --verify --chain sepolia \
            --gas-limit "$GAS_LIMIT"
        ;;
    UniswapLiquiditySetup)
        echo "=== Deploying UniswapLiquiditySetup ==="
        [ -z "${PRIVATE_KEY:-}" ] && { echo "PRIVATE_KEY must be set"; exit 1; }
        export PRIVATE_KEY
        forge script script/UniswapLiquiditySetup.s.sol:UniswapLiquiditySetupScript \
            --rpc-url "$RPC_URL" \
            --broadcast --verify --chain sepolia \
            --gas-limit "$GAS_LIMIT"
        ;;
    AddLiquidity)
        echo "=== Adding liquidity (GameToken + ETH) ==="
        [ -z "${PRIVATE_KEY:-}" ] && { echo "PRIVATE_KEY must be set"; exit 1; }
        echo "Prerequisite: deployer must hold enough GameToken (mint first via cast send if owner)."
        forge script script/AddLiquidity.s.sol:AddLiquidityScript \
            --rpc-url "$RPC_URL" \
            --broadcast \
            --gas-limit "$GAS_LIMIT"
        ;;
    TokenBalance)
        echo "=== Querying token balance ==="
        forge script script/TokenBalance.s.sol:TokenBalanceScript \
            --rpc-url "$RPC_URL" \
            --gas-limit "$GAS_LIMIT"
        ;;
    query-liquidity)
        echo "=== Querying liquidity pool reserves ==="
        forge script script/QueryLiquidity.s.sol:QueryLiquidityScript \
            --rpc-url "$RPC_URL" \
            --gas-limit "$GAS_LIMIT"
        ;;
    *)
        usage
        ;;
esac
