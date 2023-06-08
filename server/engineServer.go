package server

// 这里放的是 所有被添加的代币 需要监听并过滤其是否被交易的消息

type Coin struct {
	CoinName        string // 根据链上的不同 默认为 ETH 或者 MATIC
	ContractAddress string // 合约地址 为空表示主币

}
