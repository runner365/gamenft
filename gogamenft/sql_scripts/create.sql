-- 创建表结构
-- 注意：在 psql 中执行以下语句

-- 启用UUID扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 用户表
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    eth_address VARCHAR(42) NOT NULL UNIQUE,
    username VARCHAR(50),
    nonce VARCHAR(64) NOT NULL,
    avatar TEXT,
    bio TEXT,
    email VARCHAR(100),
    is_active BOOLEAN DEFAULT true,
    last_login TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 物品表
CREATE TABLE items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token_id BIGINT NOT NULL,
    nft_contract_address VARCHAR(42) NOT NULL,
    owner_address VARCHAR(42) NOT NULL,
    creator_address VARCHAR(42) NOT NULL,
    name VARCHAR(100),
    description TEXT,
    image_url TEXT,
    metadata_url TEXT,
    token_uri TEXT,
    rarity VARCHAR(20),
    level INTEGER DEFAULT 1,
    attributes JSONB,
    is_listed BOOLEAN DEFAULT false,
    list_price NUMERIC(36, 18),
    listed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- 唯一约束：同一个合约下的token_id必须唯一
    UNIQUE(nft_contract_address, token_id)
);

-- 订单表
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id VARCHAR(36) NOT NULL UNIQUE,
    item_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    seller_address VARCHAR(42) NOT NULL,
    buyer_address VARCHAR(42),
    price NUMERIC(36, 18) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    paid_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    tx_hash VARCHAR(66),
    cancel_reason VARCHAR(200),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 交易记录表
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tx_hash VARCHAR(66) NOT NULL UNIQUE,
    block_number BIGINT,
    block_timestamp BIGINT,
    nft_contract_address VARCHAR(42) NOT NULL,
    token_id BIGINT NOT NULL,
    from_address VARCHAR(42),
    to_address VARCHAR(42),
    price NUMERIC(36, 18),
    tx_type VARCHAR(20),
    status VARCHAR(20) DEFAULT 'pending',
    gas_used BIGINT,
    gas_price NUMERIC(36, 18),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);