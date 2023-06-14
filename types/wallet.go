package types

import (
	"math/big"
)

type Wallet struct {
	Address    string
	PublicKey  string
	PrivateKey string
}

// TransferFrom  721 交易的 input 结构
type TransferFrom struct {
	From    string
	To      string
	TokenID *big.Int
}

// Transfer  20 交易的 input 结构
type Transfer struct {
	To    string
	Value *big.Int
}

type Transaction struct {
	BlockNumber *big.Int // 区块号
	BlockHash   string   // 区块哈希
	Hash        string   // 交易hash
	From        string   // 交易者
	To          string   // 接收者
	Value       *big.Int // 交易数量
	Data        []byte   // 交易数据
	Status      uint     // 状态（0：未完成，1：已完成）
	HasCheck    bool     // 是否已经检查过`
	Dirty       bool     // 是否已经写入数据库 false 未写入  true 已写入
}
