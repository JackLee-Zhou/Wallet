package server

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/lmxdawn/wallet/db"
	"github.com/rs/zerolog/log"
)

// 这里放的是 所有被添加的代币 需要监听并过滤其是否被交易的消息

// Coin 币种结构
type Coin struct {
	CoinName        string // 根据链上的不同 默认为 ETH 或者 MATIC
	ContractAddress string // 合约地址 为空表示主币
}

// ListenCoinList 所有需要监听的代币列表
type ListenCoinList struct {
	List    []*Coin          // 遍历列表
	Mapping map[string]*Coin // map 直接查询
}

var CoinList *ListenCoinList

var ListenHttp *ethclient.Client

func CoinInit() {
	CoinList = &ListenCoinList{
		Mapping: make(map[string]*Coin),
	}
	// 读取数据库 把所有需要监听的 打入内存中
	tokens := db.GetAll20TokenFromDB()
	for _, v := range tokens {
		temp := v
		if !AddCoin(temp.CoinName, temp.ContractAddress, true) {
			log.Fatal().Msgf("init AddCoin err")
		}
	}
	for _, v := range CoinList.List {
		temp := v
		takeCoinListen(temp)
	}
	// TODO 取消硬编码
	iListenHttp, err := ethclient.Dial("https://polygon-mumbai.blockpi.network/v1/rpc/public")
	if err != nil {
		log.Fatal().Msgf("engineServer init err is %s ", err.Error())
		return
	}
	ListenHttp = iListenHttp
	log.Info().Msgf("engineServer init success ")
}

// AddCoin 添加代币 进行过滤监听
func AddCoin(coinName, contractAddress string, isInit bool) bool {
	if CoinList == nil {
		log.Fatal().Msgf("AddCoin CoinList is nil ")
		return false
	}
	coin := &Coin{
		CoinName:        coinName,
		ContractAddress: contractAddress,
	}
	CoinList.List = append(CoinList.List, coin)
	CoinList.Mapping[contractAddress] = coin

	// 初始化的时候集中处理
	if isInit {
		return true
	}
	// 开始单个的处理添加处理监听
	takeCoinListen(coin)
	return true
}

func takeCoinListen(coin *Coin) {

}
