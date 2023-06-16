package server

type CreateWalletReq struct {
	Protocol string `json:"protocol"`                    // 协议
	CoinName string `json:"coinName" binding:"required"` // 币种名称
}

type DelWalletReq struct {
	Protocol string `json:"protocol"`                   // 协议
	CoinName string `json:"coinName"`                   // 币种名称
	Address  string `json:"address" binding:"required"` // 地址
}

type WithdrawReq struct {
	Protocol string `json:"protocol" binding:"required"` // 协议
	CoinName string `json:"coinName" binding:"required"` // 币种名称
	OrderId  string `json:"orderId" binding:"required"`  // 订单号
	Address  string `json:"address" binding:"required"`  // 提现地址
	Value    int64  `json:"value" binding:"required"`    // 金额
}

type CollectionReq struct {
	Protocol string `json:"protocol" binding:"required"` // 协议
	CoinName string `json:"coinName" binding:"required"` // 币种名称
	Address  string `json:"address" binding:"required"`  // 地址
	Max      string `json:"max" binding:"required"`      // 最大归集数量（满足当前值才会归集）
}

type TransactionReceiptReq struct {
	Protocol string `json:"protocol" `                   // 协议
	CoinName string `json:"coinName" binding:"required"` // 币种名称
	Hash     string `json:"hash" binding:"required"`     // 交易哈希
}

type GetLinkStatusReq struct {
	Protocol string `json:"protocol" ` // 指定要获取的链名称
	LinkName string `json:"linkName"`  // 链接
	ChainID  uint32 `json:"chainID"`   // 链ID
}

// AddNewCoinReq 增加新的币种
type AddNewCoinReq struct {
	Protocol        string `json:"protocol" `                          // 指定要获取的链名称
	ContractAddress string `json:"contractAddress" binding:"required"` // 指定新币的合约地址
	UserAddress     string `json:"userAddress" binding:"required"`     // 用户的钱包地址
	CoinName        string `json:"coinName" `                          // 币种名称
}

// GetBalanceReq 获取账户余额信息
type GetBalanceReq struct {
	Protocol    string `json:"protocol" `                      // 指定要获取的链名称 应该用这个给 要知道现在这个用户要查哪条链上的数据
	UserAddress string `json:"userAddress" binding:"required"` // 用户的钱包地址
	CoinName    string `json:"coinName" `                      // 币种名称
	//ChainID     uint32 `json:"chainID"`                        // 链ID
}

// GetWalletActivity 获取钱包活动信息 交易记录
type GetWalletActivity struct {
	Protocol    string `json:"protocol" `                      // 指定要获取的链名称 应该用这个给 要知道现在这个用户要查哪条链上的数据
	UserAddress string `json:"userAddress" binding:"required"` // 用户的钱包地址
	CoinName    string `json:"coinName" `                      // 币种名称
}

// SendTransaction 发起一笔交易
type SendTransaction struct {
	Protocol string `json:"protocol"`                // 指定要获取的链名称 应该用这个给 要知道现在这个用户要查哪条链上的数据
	From     string `json:"from" binding:"required"` // 用户的钱包地址
	CoinName string `json:"coinName"`                // 币种名称 为空表示原生币
	To       string `json:"to" binding:"required"`   // 接收者
	Num      string `json:"num" binding:"required"`  // 数量
}

// NftTransaction NFT交易
type NftTransaction struct {
	From            string `json:"from" binding:"required"`            // 用户的钱包地址
	To              string `json:"to" binding:"required"`              // 接收者
	ContractAddress string `json:"contractAddress" binding:"required"` // NFT合约地址
	TokenID         string `json:"tokenID" binding:"required"`         // NFT的ID
}

// CheckTransReq 检查交易是否成功
type CheckTransReq struct {
	Protocol string `json:"protocol" `                  // 指定要获取的链名称 应该用这个给 要知道现在这个用户要查哪条链上的数据
	Address  string `json:"address" `                   // 用户的钱包地址
	CoinName string `json:"coinName"`                   // 币种名称 为空表示原生币
	TxHash   string `json:"txHash"  binding:"required"` // 交易Hash
}

// CheckTransResp 检查交易是否成功回执
type CheckTransResp struct {
	TxHash  string `json:"txHash"`  // 交易Hash
	Status  int    `json:"status"`  // 交易状态 0 等待 1 成功 2 失败
	Message string `json:"message"` // 交易状态描述
}

// ChangSignTypeReq 改变签名方式
type ChangSignTypeReq struct {
	WalletAddress string   `json:"walletAddress" binding:"required"` // 需要改变的钱包地址
	SignType      int32    `json:"signType" binding:"required"`      // 签名模式
	SingGroup     []string `json:"singGroup"`                        // 若是多签则要传入管理的用户钱包地址
}

// ExportWalletReq 导出钱包
type ExportWalletReq struct {
	Address string `json:"address" binding:"required"` // 导出地址
}

type ImportWalletReq struct {
	PrivateKey string `json:"privateKey" binding:"required"` // 私钥
}

// LoginReq 登录请求
type LoginReq struct {
	Account string `json:"account" binding:"required"` // 登录账户
	PassWD  string `json:"passWD" binding:"required"`  // 传入的密码
}

// RegisterReq 注册请求
type RegisterReq struct {
	Account string `json:"account" binding:"required"` // 登录账户
	PassWD  string `json:"passWD" binding:"required"`  // 传入的密码
}

type SpeedUpReq struct {
	Address string `json:"address" binding:"required"` // 钱包地址
	TxHash  string `json:"txHash" binding:"required"`  // 交易哈希
	//几倍加速?
}

type CallContractReq struct {
	From                 string `json:"from" binding:"required"` // 钱包地址
	To                   string `json:"to" binding:"required"`   // 合约地址
	Data                 string `json:"data" `                   // 数据
	Value                string `json:"value"`                   // 金额
	Gas                  uint64 `json:"gas" `                    // gas
	GasPrice             string `json:"gasPrice" `               // gasPrice
	MaxFeePerGas         string `json:"maxFeePerGas" `           // maxFeePerGas
	MaxPriorityFeePerGas string `json:"maxPriorityFeePerGas" `   // maxProfitGas
}

type CancelReq struct {
	Address string `json:"address" binding:"required"` // 钱包地址
	TxHash  string `json:"txHash" binding:"required"`  // 交易哈希
}

// AddNFTReq 向钱中加入 NFT
type AddNFTReq struct {
	UserAddress     string `json:"userAddress" binding:"required"` // 用户的钱包地址
	ContractAddress string `json:"contractAddress" binding:"required"`
	TokenID         string `json:"tokenID" binding:"required"`
}
