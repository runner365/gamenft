# 设计思路
```markdown
前端/客户端 → 业务服务端 → 区块链节点
        ↓         ↓
      缓存层    数据库
```
- 前端/客户端：用户界面，提供交互功能。
- 业务服务端：处理用户请求，与区块链节点交互。
- 区块链节点：与以太坊网络交互，执行智能合约。
- 缓存层：缓存频繁访问的数据，提高响应速度。
- 数据库：存储用户数据、交易记录等。

## 数据库设计

### 1. 用户表 (users)
```
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    eth_address VARCHAR(42) UNIQUE NOT NULL,  -- 以太坊地址
    username VARCHAR(50),                     -- 可选的用户名
    nonce VARCHAR(64) NOT NULL,              -- 登录签名随机数
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### 2. NFT物品表 (items)
```
CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    token_id BIGINT NOT NULL,                 -- ERC721 tokenId
    nft_contract_address VARCHAR(42) NOT NULL, -- 游戏道具合约地址
    owner_address VARCHAR(42) NOT NULL,       -- 当前所有者
    creator_address VARCHAR(42) NOT NULL,     -- 创建者
    name VARCHAR(100),                        -- 物品名称
    description TEXT,                         -- 描述
    image_url TEXT,                           -- 图片URL
    metadata_url TEXT,                        -- 元数据URI
    token_uri TEXT,                           -- tokenURI
    rarity VARCHAR(20),                       -- 稀有度: 普通/稀有/史诗/传说
    level INT,                               -- 等级
    attributes JSONB,                        -- 扩展属性
    is_listed BOOLEAN DEFAULT FALSE,         -- 是否在售
    list_price DECIMAL(36, 18),              -- 列表价格（以tokenG为单位）
    listed_at TIMESTAMP,                     -- 上架时间
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(nft_contract_address, token_id)
);
```

### 3. 交易记录表 (transactions)
```
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    tx_hash VARCHAR(66) UNIQUE NOT NULL,     -- 交易哈希
    block_number BIGINT,                      -- 区块号
    nft_contract_address VARCHAR(42) NOT NULL,
    token_id BIGINT NOT NULL,
    from_address VARCHAR(42),                 -- 卖方
    to_address VARCHAR(42),                   -- 买方
    price DECIMAL(36, 18),                    -- 交易价格
    tx_type VARCHAR(20),                      -- 类型: MINT/LIST/BUY/CANCEL
    status VARCHAR(20) DEFAULT 'pending',     -- 状态: pending/success/failed
    created_at TIMESTAMP DEFAULT NOW()
);
```

### 4. 订单表 (orders)
```
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    order_id VARCHAR(36) UNIQUE NOT NULL,    -- 订单号
    item_id INT REFERENCES items(id),         -- 物品ID
    seller_address VARCHAR(42) NOT NULL,      -- 卖方
    buyer_address VARCHAR(42),                -- 买方
    price DECIMAL(36, 18) NOT NULL,          -- 价格
    status VARCHAR(20) DEFAULT 'pending',    -- pending/paid/cancelled/completed
    paid_at TIMESTAMP,                       -- 支付时间
    completed_at TIMESTAMP,                  -- 完成时间
    tx_hash VARCHAR(66),                     -- 链上交易哈希
    created_at TIMESTAMP DEFAULT NOW()
);
```

### 5. 游戏道具库存表 (player_items)
```
CREATE TABLE player_items (
    id SERIAL PRIMARY KEY,
    user_address VARCHAR(42) NOT NULL,       -- 用户以太坊地址
    item_type VARCHAR(20) NOT NULL,          -- 道具类型: knife/pistol/bomb
    quantity INT DEFAULT 0 NOT NULL,         -- 持有数量
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_address, item_type)          -- 复合唯一约束，支持 upsert
);
```

## API 设计

### 认证方式
使用以太坊签名认证（EIP-191 personal_sign），登录后返回 JWT Token。
后续请求在 Authorization 头携带 `Bearer <token>`。

### 公开接口（无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/auth/challenge?address=0x...` | 获取签名挑战消息 |
| POST | `/api/v1/auth/login` | 签名验证登录，返回 JWT |
| GET | `/api/v1/items?page=1&size=20` | 获取在售物品列表（分页） |
| GET | `/api/v1/items/:id` | 获取物品详情 |
| POST | `/api/v1/items` | 导入/注册链上 NFT 物品 |

### 认证接口（需 JWT）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/profile` | 获取个人资料和统计 |
| GET | `/api/v1/my-items` | 获取我的 NFT 物品 |
| POST | `/api/v1/items/:id/list` | 上架物品（body: `{"price": "..."}`） |
| POST | `/api/v1/items/:id/buy` | 购买物品（body: `{"price": "..."}`） |
| GET | `/api/v1/orders` | 获取我的订单 |
| **POST** | **`/api/v1/rewards`** | **获得游戏道具（body: `{"item_type": "knife"}`）** |
| **GET** | **`/api/v1/rewards`** | **获取游戏道具库存（返回 `{"knife": N, ...}`）** |
| **POST** | **`/api/v1/rewards/use`** | **消耗游戏道具（body: `{"item_type": "knife"}`）** |

## 游戏道具系统

### 概述
俄罗斯方块游戏中每消除 5 行，玩家随机获得一个游戏道具：

| 道具 | 概率 | 说明 |
|------|------|------|
| knife (刀) | 30% (31-60) | 可消除单行 |
| pistol (手枪) | 30% (61-90) | 可消除单格 |
| bomb (炸弹) | 10% (91-100) | 可消除 3x3 区域 |
| none (无) | 30% (1-30) | 无奖励 |

### 数据流
```
前端游戏获得道具 → 本地计数器 +1 → POST /api/v1/rewards → PostgreSQL upsert (quantity+1)
页面刷新/登录   → GET /api/v1/rewards  → 加载库存 → 恢复本地计数器
前端使用道具     → 本地计数器 -1 → POST /api/v1/rewards/use → PostgreSQL decrement (quantity-1)
```

### 持久化策略
- 采用乐观更新：前端先更新本地计数器，API 调用异步 fire-and-forget
- 未登录用户的道具仅在本地内存中，不发送请求
- player_items 表使用 `(user_address, item_type)` 复合唯一索引，通过 `FirstOrCreate + UPDATE quantity+1` 实现原子累加
- 道具消耗前后端双重校验（本地 count > 0 + 服务端 quantity > 0）

