-- 活跃用户视图
CREATE VIEW active_users AS
SELECT 
    id,
    eth_address,
    username,
    email,
    last_login
FROM users
WHERE is_active = true;

-- 在售物品视图
CREATE VIEW listed_items AS
SELECT 
    i.id,
    i.token_id,
    i.nft_contract_address,
    i.owner_address,
    i.creator_address,
    i.name,
    i.description,
    i.image_url,
    i.rarity,
    i.level,
    i.list_price,
    i.listed_at,
    u.username as owner_username
FROM items i
LEFT JOIN users u ON i.owner_address = u.eth_address
WHERE i.is_listed = true;

-- 用户资产统计视图
CREATE VIEW user_stats AS
SELECT 
    u.eth_address,
    u.username,
    COUNT(DISTINCT i.id) as total_items,
    COUNT(DISTINCT CASE WHEN i.is_listed THEN i.id END) as listed_items,
    COALESCE(SUM(CASE WHEN o.status = 'completed' THEN o.price ELSE 0 END), 0) as total_volume
FROM users u
LEFT JOIN items i ON u.eth_address = i.owner_address
LEFT JOIN orders o ON u.eth_address = o.seller_address
GROUP BY u.eth_address, u.username;