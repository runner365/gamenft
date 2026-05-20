-- cleanup_user_items.sql
-- Clean up all game items and related records for a specific user.
-- Replace <USER_ADDRESS> with the target Ethereum address (lowercase, 0x...).

-- Set the target address
\set target_address '0xf47005ae799d3d0c5490e8aa9e9b218196f4c17c'

-- 1. In-game reward items (knife/pistol/bomb)
DELETE FROM player_items WHERE user_address = :'target_address';

-- 2. Marketplace items owned by the user
DELETE FROM items WHERE owner_address = :'target_address';

-- 3. Orders where user is buyer or seller
DELETE FROM orders WHERE buyer_address = :'target_address' OR seller_address = :'target_address';

-- 4. Transactions
DELETE FROM transactions WHERE from_address = :'target_address' OR to_address = :'target_address';

-- 5. Token purchases (GMTK buy records)
DELETE FROM token_purchases WHERE user_address = :'target_address';

-- Verify cleanup
SELECT 'player_items' AS table_name, COUNT(*) AS remaining FROM player_items WHERE user_address = :'target_address'
UNION ALL
SELECT 'items', COUNT(*) FROM items WHERE owner_address = :'target_address'
UNION ALL
SELECT 'orders', COUNT(*) FROM orders WHERE buyer_address = :'target_address' OR seller_address = :'target_address'
UNION ALL
SELECT 'transactions', COUNT(*) FROM transactions WHERE from_address = :'target_address' OR to_address = :'target_address'
UNION ALL
SELECT 'token_purchases', COUNT(*) FROM token_purchases WHERE user_address = :'target_address';
