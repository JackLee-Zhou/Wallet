package server

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/lmxdawn/wallet/db"
	"github.com/lmxdawn/wallet/engine"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"strconv"
	"time"
)

// 这里放的是 所有被添加的代币 需要监听并过滤其是否被交易的消息

// Coin 币种结构
type Coin struct {
	CoinName        string // 根据链上的不同 默认为 ETH 或者 MATIC
	ContractAddress string // 合约地址 为空表示主币
	IsNFT           bool   // 是否是 NFT
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
		if !AddCoin(temp.CoinName, temp.ContractAddress, true, temp.IsNFT) {
			log.Fatal().Msgf("init AddCoin err")
		}
	}

	iListenHttp, err := ethclient.Dial(url)
	if err != nil {
		log.Fatal().Msgf("engineServer init err is %s ", err.Error())
		return
	}
	ListenHttp = iListenHttp
	timerUpDataBalance(120 * time.Second)
	timerUpDataNFTOwner(120 * time.Second)
	log.Info().Msgf("engineServer init success ")
}

// AddCoin 添加代币 进行过滤监听
func AddCoin(coinName, contractAddress string, isInit, isNFT bool) bool {
	if CoinList == nil {
		log.Fatal().Msgf("AddCoin CoinList is nil ")
		return false
	}
	if _, ok := CoinList.Mapping[contractAddress]; ok {
		log.Info().Msgf("AddCoin contractAddress is exist ")
		return false
	}
	coin := &Coin{
		CoinName:        coinName,
		ContractAddress: contractAddress,
		IsNFT:           isNFT,
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

// timerUpDataNFTOwner 定时更新 NFT 所有者
func timerUpDataNFTOwner(dur time.Duration) {
	timer := time.NewTimer(dur)
	log.Info().Msgf("timerUpDataNFTOwner start ")
	go func(t *time.Timer, newDur time.Duration) {
		type ChangeOwner struct {
			address  string
			contract string
			tokenId  int
		}
		for {
			<-t.C
			// 遍历用户 NFT 资产 检查其是否任然是所有者
			usrs := db.GetAllAddress()
			if usrs == nil {
				log.Info().Msgf("timerUpDataNFTOwner usrs is nil")
				return
			}
			nUsrs := []*db.User{}

			// 目标用户 变更的的数据
			changes := []*ChangeOwner{}
			for _, v := range usrs {
				temp := v
				isChange := false
				//newAsset := []*db.NFTAssets{}
				if temp.Assets == nil {
					continue
				}
				for i := len(temp.Assets[temp.CurrentNetWork.NetWorkName].NFT) - 1; i >= 0; i-- {
					t := temp.Assets[temp.CurrentNetWork.NetWorkName].NFT[i]
					// 更新余额
					tokenId, _ := strconv.Atoi(t.TokenID)
					addr, ok := engine.CheckIsOwner(t.ContractAddress, temp.Address, tokenId)

					// 不再是所有者
					if !ok {
						changes = append(changes, &ChangeOwner{
							address:  addr,
							contract: t.ContractAddress,
							tokenId:  tokenId,
						})
						isChange = true
						// 删除
						temp.Assets[temp.CurrentNetWork.NetWorkName].NFT =
							append(temp.Assets[temp.CurrentNetWork.NetWorkName].NFT[:i],
								temp.Assets[temp.CurrentNetWork.NetWorkName].NFT[i+1:]...)
					}

				}
				// 变更后的原始所有者
				if isChange {
					nUsrs = append(nUsrs, temp)
				}
			}
			//	若不是 则检查新的所有者 是否是本服务的用户
			//	若是 则更新目标用户的NFT数据
			for _, v := range changes {
				usr := db.GetUserFromDB(v.address)
				if usr == nil {
					log.Error().Msgf("timerUpDataNFTOwner GetUserFromDB not in db, address is  %s ", v.address)
					continue
				}
				usr.Assets[usr.CurrentNetWork.NetWorkName].NFT = append(usr.Assets[usr.CurrentNetWork.NetWorkName].NFT, &db.NFTAssets{
					ContractAddress: v.contract,
					TokenID:         strconv.Itoa(v.tokenId),
				})
				nUsrs = append(nUsrs, usr)
			}
			// 统一更新所有变更的用户
			for _, v := range nUsrs {
				err := db.UpDataUserInfo(v)
				if err != nil {
					log.Error().Msgf("timerUpDataNFTOwner UpDataUserInfo err is %s ", err.Error())
					continue
				}
			}
			log.Info().Msgf("timerUpDataNFTOwner end at %s ", time.Now().Format("2006-01-02 15:04:05"))
			t.Reset(newDur)
		}
	}(timer, dur)
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

				if len(newAsset) > 0 {
					temp.Assets[temp.CurrentNetWork.NetWorkName].Coin = newAsset
					nUsrs = append(nUsrs, temp)
				}

			}

			for _, v := range nUsrs {
				err := db.UpDataUserInfo(v)
				if err != nil {
					log.Error().Msgf("timerUpDataBalance UpDataUserInfo address not in db address is %s ", v.Address)
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
