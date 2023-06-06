package server

type CreateWalletReq struct {
	Protocol string `json:"protocol" binding:"required"` // 协议
	CoinName string `json:"coinName" binding:"required"` // 币种名称
}

type DelWalletReq struct {
	Protocol string `json:"protocol" binding:"required"` // 协议
	CoinName string `json:"coinName" binding:"required"` // 币种名称
	Address  string `json:"address" binding:"required"`  // 地址
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
	Protocol string `json:"protocol" binding:"required"` // 协议
	CoinName string `json:"coinName" binding:"required"` // 币种名称
	Hash     string `json:"hash" binding:"required"`     // 交易哈希
}

type GetLinkStatusReq struct {
	Protocol string `json:"protocol" binding:"required"` // 指定要获取的链名称
	Rpc      string `json:"rpc" binding:"required"`      // 链接地址
	ChainID  uint32 `json:"chainID"`                     // 链ID
}

// AddNewCoinReq 增加新的币种
type AddNewCoinReq struct {
	Protocol        string `json:"protocol" binding:"required"`        // 指定要获取的链名称
	ContractAddress string `json:"contractAddress" binding:"required"` // 指定新币的合约地址
	UserAddress     string `json:"userAddress" binding:"required"`     // 用户的钱包地址
	CoinName        string
}

// GetBalanceReq 获取账户余额信息
type GetBalanceReq struct {
	Protocol    string `json:"protocol" binding:"required"`    // 指定要获取的链名称 应该用这个给 要知道现在这个用户要查哪条链上的数据
	UserAddress string `json:"userAddress" binding:"required"` // 用户的钱包地址
	CoinName    string `json:"coinName" `                      // 币种名称
	//ChainID     uint32 `json:"chainID"`                        // 链ID
}

// GetWalletActivity 获取钱包活动信息 交易记录
type GetWalletActivity struct {
	Protocol    string `json:"protocol" binding:"required"`    // 指定要获取的链名称 应该用这个给 要知道现在这个用户要查哪条链上的数据
	UserAddress string `json:"userAddress" binding:"required"` // 用户的钱包地址
	CoinName    string `json:"coinName" `                      // 币种名称
}

// SendTransaction 发起一笔交易
type SendTransaction struct {
	Protocol string `json:"protocol" binding:"required"` // 指定要获取的链名称 应该用这个给 要知道现在这个用户要查哪条链上的数据
	From     string `json:"from" binding:"required"`     // 用户的钱包地址
	CoinName string `json:"coinName" binding:"required"` // 币种名称 为空表示原生币
	To       string `json:"to" binding:"required"`       // 接收者
	Num      string `json:"num" binding:"required"`      // 数量
}

// CheckTransReq 检查交易是否成功
type CheckTransReq struct {
	Protocol string `json:"protocol" binding:"required"` // 指定要获取的链名称 应该用这个给 要知道现在这个用户要查哪条链上的数据
	Address  string `json:"address" binding:"required"`  // 用户的钱包地址
	CoinName string `json:"coinName" binding:"required"` // 币种名称 为空表示原生币
	TxHash   string `json:"txHash"  binding:"required"`  // 交易Hash
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
