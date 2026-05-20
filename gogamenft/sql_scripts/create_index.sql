-- 用户表索引
CREATE INDEX idx_users_eth_address ON users(eth_address);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_last_login ON users(last_login);

-- 物品表索引
CREATE INDEX idx_items_owner_address ON items(owner_address);
CREATE INDEX idx_items_creator_address ON items(creator_address);
CREATE INDEX idx_items_is_listed ON items(is_listed);
CREATE INDEX idx_items_rarity ON items(rarity);
CREATE INDEX idx_items_level ON items(level);
CREATE INDEX idx_items_listed_at ON items(listed_at);
CREATE INDEX idx_items_created_at ON items(created_at);
CREATE INDEX idx_items_list_price ON items(list_price);

-- 对JSONB字段创建GIN索引以支持查询
CREATE INDEX idx_items_attributes ON items USING gin(attributes);

-- 订单表索引
CREATE INDEX idx_orders_seller_address ON orders(seller_address);
CREATE INDEX idx_orders_buyer_address ON orders(buyer_address);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_item_id ON orders(item_id);
CREATE INDEX idx_orders_created_at ON orders(created_at);
CREATE INDEX idx_orders_tx_hash ON orders(tx_hash);

-- 交易记录表索引
CREATE INDEX idx_transactions_from_address ON transactions(from_address);
CREATE INDEX idx_transactions_to_address ON transactions(to_address);
CREATE INDEX idx_transactions_block_number ON transactions(block_number);
CREATE INDEX idx_transactions_tx_type ON transactions(tx_type);
CREATE INDEX idx_transactions_created_at ON transactions(created_at);
CREATE INDEX idx_transactions_nft_token ON transactions(nft_contract_address, token_id);
