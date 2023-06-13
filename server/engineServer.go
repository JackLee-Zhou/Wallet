package server

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/lmxdawn/wallet/db"
	"github.com/lmxdawn/wallet/engine"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"time"
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

// ListenHttp 区别于 Worker 的 http  ListenHttp 只用于监听区块
var ListenHttp *ethclient.Client

func CoinInit(url string) {
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
	//for _, v := range CoinList.List {
	//	temp := v
	//	takeCoinListen(temp)
	//}

	iListenHttp, err := ethclient.Dial(url)
	if err != nil {
		log.Fatal().Msgf("engineServer init err is %s ", err.Error())
		return
	}
	ListenHttp = iListenHttp
	timerUpDataBalance(30 * time.Second)
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

// timerUpDataBalance 定时更新用户的余额
func timerUpDataBalance(dur time.Duration) {
	timer := time.NewTimer(dur)
	log.Info().Msgf("timerUpDataBalance start ")
	go func(t *time.Timer, newDur time.Duration) {
		for {
			<-t.C
			usrs := db.GetAllAddress()
			if usrs == nil {
				log.Info().Msgf("timerUpDataBalance usrs is nil")
				return
			}
			nUsrs := []*db.User{}
			for _, v := range usrs {
				temp := v
				newAsset := []*db.CoinAssets{}
				if temp.Assets == nil {
					continue
				}
				for _, uV := range temp.Assets[temp.CurrentNetWork.NetWorkName].Coin {
					t := uV
					// 更新余额
					balance, err := engine.EWorker.GetBalance(temp.Address, t.ContractAddress)
					if err != nil {
						log.Error().Msgf("timerUpDataBalance GetBalance err is %s ", err.Error())
						continue
					}
					t.Num = balance
					newAsset = append(newAsset, t)
				}
				temp.Assets[temp.CurrentNetWork.NetWorkName].Coin = newAsset
				nUsrs = append(nUsrs, temp)
			}

			for _, v := range nUsrs {
				err := db.UpDataUserInfo(v)
				if err != nil {
					log.Error().Msgf("timerUpDataBalance UpDataUserInfo err is %s ", err.Error())
					continue
				}
			}
			log.Info().Msgf("timerUpDataBalance end at %s ", time.Now().Format("2006-01-02 15:04:05"))
			t.Reset(newDur)
		}

	}(timer, dur)

}

func GetTransFromLink(address string) []byte {
	response, err := http.Get("https://api-testnet.polygonscan.com/api?" +
		"module=account&action=txlist&address=" + address + "&startblock=0&endblock=99999999&page=1&offset=10" +
		"&sort=asc&apikey=432F174RDZHNVM81M4JT8UJAWFW87DKUBV")
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	//res := []byte{}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Info().Msgf("GetTransFromLink err is %s ", err.Error())
		return nil
	}
	return body
}

func takeCoinListen(coin *Coin) {
	// 是否需要拉去这个代币是记录
}
