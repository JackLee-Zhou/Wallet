package server

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/lmxdawn/wallet/db"
	"github.com/lmxdawn/wallet/types"
	"github.com/rs/zerolog/log"
	"math/big"
	"sync"
	"time"
)

// Tran 监听的交易结构
type Tran struct {
	*types.Transaction
}

type ListTrans struct {
	TransMap *sync.Map
}

var TransMap *ListTrans

// listenAllBlock 监听所有区块 不断的监听所有的区块 并将其加入到队列中 等待使用
func listenAllBlock(initNum uint64) {
	startNum := initNum

	// 循环 不阻塞 跳出
	for {
		var toBlock uint64
		// 从最新的节点开始监听
		nowNumber, err := ListenHttp.BlockNumber(context.Background())
		if err != nil {
			log.Error().Msgf("listenAllBlock init err is %s ", err.Error())
			return
		}
		toBlock = startNum + 100
		if startNum == 0 {
			// 从创世区块开始查询
			toBlock = nowNumber
		} else if toBlock > nowNumber {
			// 截至最新的区块
			toBlock = nowNumber
		}

		// 开始监听
		for i := startNum; i < toBlock; i++ {
			if err := listenBlock(int64(i)); err != nil {
				log.Info().Msgf("listenAllBlock listenBlock err is %s ", err.Error())
				log.Info().Msgf("等待%d秒，当前已是最新区块", 10)
				<-time.After(time.Duration(10) * time.Second)
			}
		}
		startNum = toBlock
	}
}

// listenBlock 监听单个区块 并提取其中的交易信息
func listenBlock(blockNum int64) error {
	block, err := ListenHttp.BlockByNumber(context.Background(), big.NewInt(blockNum))
	if err != nil {
		return err
	}
	chainID, err := ListenHttp.NetworkID(context.Background())
	for _, tx := range block.Transactions() {
		// 如果接收方地址为空，则是创建合约的交易，忽略过去
		if tx.To() == nil {
			continue
		}
		msg, err := tx.AsMessage(ethTypes.LatestSignerForChainID(chainID), tx.GasPrice())
		if err != nil {
			continue
		}

		// 若是代币交易 则 to 应该是合约地址
		_, ok := CoinList.Mapping[tx.To().Hex()]

		// 只关心本钱包用户的交易

		// 是所需要监听的代币 或者 原生币交易 则 data 为空
		if ok || db.CheckWalletIsInDB(msg.To().Hex()) ||
			db.CheckWalletIsInDB(msg.From().Hex()) {
			ts := &types.Transaction{
				BlockNumber: big.NewInt(blockNum),
				BlockHash:   block.Hash().Hex(),
				Hash:        tx.Hash().Hex(),
				From:        msg.From().Hex(),
				To:          tx.To().Hex(),
				Value:       tx.Value(),
			}
			TransMap.TransMap.Store(ts.Hash, ts)
			log.Info().Msgf("listenBlock find Trans Hash is %s blockNum is %d", ts.Hash, blockNum)
		}
	}
	return nil
}

// startGetReceipt 开始获取交易情况
func startGetReceipt(maxListenLine int) {
	log.Info().Msgf("startGetReceipt start")
	go func() {
		// 不断监听交易状态
		for {
			TransMap.TransMap.Range(func(key, value interface{}) bool {
				ts := value.(*types.Transaction)
				// 这笔交易已经判断了
				if ts.HasCheck {
					return true
				}
				log.Info().Msgf("listenReceipt %+v ", ts)
				hash := common.HexToHash(ts.Hash)
				receipt, err := ListenHttp.TransactionReceipt(context.Background(), hash)
				if err != nil {
					log.Info().Msgf("startGetReceipt TransactionReceipt err is %s", err.Error())
					return true
				}
				latest, err := ListenHttp.BlockNumber(context.Background())
				if err != nil {
					log.Info().Msgf("startGetReceipt BlockNumber err is %s", err.Error())
					return true
				}
				// 判断确认数
				confirms := latest - receipt.BlockNumber.Uint64() + 1
				if confirms > 5 {
					return true
				}
				status := receipt.Status
				ts.Status = uint(status)
				ts.HasCheck = true
				log.Info().Msgf("startGetReceipt TransactionReceipt %+v", ts)
				return true
			})
		}
	}()
}

// readTransMap 开始过过滤监听
func readTransMap() {
	for {

	}
}

func Init() {
	TransMap = &ListTrans{
		TransMap: &sync.Map{},
	}
	num, _ := ListenHttp.BlockNumber(context.Background())
	go listenAllBlock(num)
	go startGetReceipt(5)
	//readTransMap()
}
