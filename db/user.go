package db

type NetWork struct {
	NetWorkName string
	RpcUrl      string
	ChainID     uint32
}

type Assets struct {
	ContractAddress string // 资产合约地址，为空表示主币
	Symbol          string // 资产符号
	Num             string // 拥有的数量
}

type User struct {
	Address        string     // 用户钱包地址
	PrivateKey     string     // 用户私钥
	CurrentNetWork *NetWork   // 用户当前所处的网络
	NetWorks       []*NetWork // 用户添加的网络地址
	UserAssets     []*Assets  // 用户资产数据
}
