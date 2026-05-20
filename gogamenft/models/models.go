// models/models.go
package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// User 用户模型
type User struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	EthAddress string    `gorm:"size:42;uniqueIndex;not null" json:"eth_address"` // 以太坊地址
	Username   string    `gorm:"size:50;index" json:"username"`                   // 用户名
	Nonce      string    `gorm:"size:64;not null" json:"-"`                       // 登录签名随机数
	Avatar     string    `gorm:"type:text" json:"avatar"`                         // 头像URL
	Bio        string    `gorm:"type:text" json:"bio"`                            // 个人简介
	Email      string    `gorm:"size:100;index" json:"email"`                     // 邮箱（可选）
	IsActive   bool      `gorm:"default:true" json:"is_active"`                   // 是否激活
	LastLogin  time.Time `json:"last_login"`                                      // 最后登录时间
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// 关联关系
	Items        []Item        `gorm:"foreignKey:OwnerAddress;references:EthAddress" json:"items,omitempty"`
	Orders       []Order       `gorm:"foreignKey:BuyerAddress;references:EthAddress" json:"orders,omitempty"`
	SellOrders   []Order       `gorm:"foreignKey:SellerAddress;references:EthAddress" json:"sell_orders,omitempty"`
	Transactions []Transaction `gorm:"foreignKey:FromAddress;references:EthAddress" json:"transactions,omitempty"`
	PlayerItems  []PlayerItem  `gorm:"foreignKey:UserAddress;references:EthAddress" json:"player_items,omitempty"`
}

// Item NFT物品模型
type Item struct {
	ID                 uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	TokenID            int64      `gorm:"not null;uniqueIndex:idx_nft_token" json:"token_id"`                      // ERC721/ERC1155 tokenId
	NFTContractAddress string     `gorm:"size:42;not null;uniqueIndex:idx_nft_token" json:"nft_contract_address"`  // 游戏道具合约地址
	OwnerAddress       string     `gorm:"size:42;index;not null" json:"owner_address"`   // 当前所有者
	CreatorAddress     string     `gorm:"size:42;index;not null" json:"creator_address"` // 创建者
	Name               string     `gorm:"size:100" json:"name"`                          // 物品名称
	Description        string     `gorm:"type:text" json:"description"`                  // 描述
	ImageURL           string     `gorm:"type:text" json:"image_url"`                    // 图片URL
	MetadataURL        string     `gorm:"type:text" json:"metadata_url"`                 // 元数据URI
	TokenURI           string     `gorm:"type:text" json:"token_uri"`                    // tokenURI
	Rarity             string     `gorm:"size:20" json:"rarity"`                         // 稀有度
	Level              int        `gorm:"default:1" json:"level"`                        // 等级
	Attributes         JSONB      `gorm:"type:jsonb" json:"attributes"`                  // 扩展属性
	TokenStandard      string     `gorm:"size:10;default:'ERC721'" json:"token_standard"` // ERC721 / ERC1155
	Amount             int        `gorm:"default:1" json:"amount"`                       // 上架数量（ERC1155 >1）
	IsListed           bool       `gorm:"default:false;index" json:"is_listed"`          // 是否在售
	ListPrice          string     `gorm:"type:numeric(36,18)" json:"list_price"`         // 列表价格（单价）
	ListedAt           *time.Time `json:"listed_at"`                                     // 上架时间
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`

	// 唯一约束由 TokenID + NFTContractAddress 组成（使用复合唯一索引）

	// 关联关系
	Owner   *User   `gorm:"foreignKey:OwnerAddress;references:EthAddress" json:"owner,omitempty"`
	Creator *User   `gorm:"foreignKey:CreatorAddress;references:EthAddress" json:"creator,omitempty"`
	Orders  []Order `gorm:"foreignKey:ItemID" json:"orders,omitempty"`
}

// Order 订单模型
type Order struct {
	ID            uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID       string     `gorm:"size:36;uniqueIndex;not null" json:"order_id"`  // 订单号
	ItemID        uint       `gorm:"index" json:"item_id"`                          // 物品ID
	SellerAddress string     `gorm:"size:42;index;not null" json:"seller_address"`  // 卖方
	BuyerAddress  string     `gorm:"size:42;index" json:"buyer_address"`            // 买方
	Price         string     `gorm:"type:numeric(36,18);not null" json:"price"`     // 价格
	Status        string     `gorm:"size:20;default:'pending';index" json:"status"` // 状态
	PaidAt        *time.Time `json:"paid_at"`                                       // 支付时间
	CompletedAt   *time.Time `json:"completed_at"`                                  // 完成时间
	TxHash        string     `gorm:"size:66;index" json:"tx_hash"`                  // 链上交易哈希
	CancelReason  string     `gorm:"size:200" json:"cancel_reason"`                 // 取消原因
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// 关联关系
	Item   *Item `gorm:"foreignKey:ItemID" json:"item,omitempty"`
	Seller *User `gorm:"foreignKey:SellerAddress;references:EthAddress" json:"seller,omitempty"`
	Buyer  *User `gorm:"foreignKey:BuyerAddress;references:EthAddress" json:"buyer,omitempty"`
}

// PlayerItem 游戏道具库存模型（非链上NFT，仅游戏内物品）
type PlayerItem struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserAddress string    `gorm:"size:42;not null;uniqueIndex:idx_user_item_type" json:"user_address"`
	ItemType    string    `gorm:"size:20;not null;uniqueIndex:idx_user_item_type" json:"item_type"` // knife, pistol, bomb
	Quantity    int       `gorm:"default:0;not null" json:"quantity"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	User *User `gorm:"foreignKey:UserAddress;references:EthAddress" json:"user,omitempty"`
}

// Transaction 交易记录模型
type Transaction struct {
	ID                 uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TxHash             string    `gorm:"size:66;uniqueIndex;not null" json:"tx_hash"`        // 交易哈希
	BlockNumber        int64     `gorm:"index" json:"block_number"`                          // 区块号
	BlockTimestamp     int64     `json:"block_timestamp"`                                    // 区块时间戳
	NFTContractAddress string    `gorm:"size:42;index;not null" json:"nft_contract_address"` // NFT合约地址
	TokenID            int64     `gorm:"index;not null" json:"token_id"`                     // tokenId
	FromAddress        string    `gorm:"size:42;index" json:"from_address"`                  // 卖方
	ToAddress          string    `gorm:"size:42;index" json:"to_address"`                    // 买方
	Price              string    `gorm:"type:numeric(36,18)" json:"price"`                   // 交易价格
	TxType             string    `gorm:"size:20;index" json:"tx_type"`                       // 交易类型
	Status             string    `gorm:"size:20;default:'pending'" json:"status"`            // 状态
	GasUsed            uint64    `json:"gas_used"`                                           // Gas使用量
	GasPrice           string    `gorm:"type:numeric(36,18)" json:"gas_price"`               // Gas价格
	CreatedAt          time.Time `json:"created_at"`

	// 关联关系
	FromUser *User `gorm:"foreignKey:FromAddress;references:EthAddress" json:"from_user,omitempty"`
	ToUser   *User `gorm:"foreignKey:ToAddress;references:EthAddress" json:"to_user,omitempty"`
	Item     *Item `gorm:"foreignKey:TokenID,NFTContractAddress;references:TokenID,NFTContractAddress" json:"item,omitempty"`
}

// Metadata 元数据模型
type Metadata struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Image       string                 `json:"image"`
	ExternalURL string                 `json:"external_url,omitempty"`
	Attributes  []MetadataAttribute    `json:"attributes,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

type MetadataAttribute struct {
	TraitType string      `json:"trait_type"`
	Value     interface{} `json:"value"`
	Display   interface{} `json:"display_type,omitempty"`
}

// TokenPurchase records an on-chain ETH-to-GMTK purchase.
type TokenPurchase struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserAddress string    `gorm:"size:42;index;not null" json:"user_address"`
	EthAmount   string    `gorm:"type:numeric(36,18);not null" json:"eth_amount"`
	TokenAmount string    `gorm:"type:numeric(36,18);not null" json:"token_amount"`
	Rate        string    `gorm:"type:numeric(36,18);not null" json:"rate"`
	TxHash      string    `gorm:"size:66;uniqueIndex;not null" json:"tx_hash"`
	Status      string    `gorm:"size:20;default:'completed'" json:"status"`
	CreatedAt   time.Time `json:"created_at"`

	User *User `gorm:"foreignKey:UserAddress;references:EthAddress" json:"user,omitempty"`
}

// Task 异步任务模型（链上操作队列）
type Task struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID      string    `gorm:"size:64;uniqueIndex;not null" json:"task_id"`
	TaskType    string    `gorm:"size:32;index;not null" json:"task_type"` // mint_reward
	UserAddress string    `gorm:"size:42;index;not null" json:"user_address"`
	ItemType    string    `gorm:"size:16;not null" json:"item_type"`
	TokenID     int64     `gorm:"not null" json:"token_id"`
	Amount      int64     `gorm:"not null;default:1" json:"amount"`
	Status      string    `gorm:"size:16;index;not null;default:pending" json:"status"` // pending/processing/tx_sent/confirmed/failed
	TxHash      string    `gorm:"size:66" json:"tx_hash"`
	ErrorMsg    string    `gorm:"size:512" json:"error_msg"`
	RetryCount  int       `gorm:"default:0" json:"retry_count"`
	MaxRetries  int       `gorm:"default:3" json:"max_retries"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProcessedEvent 用于多实例事件去重
type ProcessedEvent struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TxHash    string    `gorm:"size:66;uniqueIndex;not null" json:"tx_hash"`
	CreatedAt time.Time `json:"created_at"`
}

// JSONB 用于PostgreSQL的JSONB类型
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONB)
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, j)
}
