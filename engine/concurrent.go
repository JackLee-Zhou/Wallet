package engine

import (
	"context"
	"github.com/lmxdawn/wallet/client"
	"github.com/lmxdawn/wallet/config"
	"github.com/lmxdawn/wallet/db"
	"github.com/lmxdawn/wallet/scheduler"
	"github.com/lmxdawn/wallet/types"
	"github.com/rs/zerolog/log"
	"math/big"
	"sync"
	"time"
)

type Worker interface {
	GetNowBlockNum() (uint64, error)
	GetTransaction(num uint64) ([]types.Transaction, uint64, error)
	GetTransactionReceipt(*types.Transaction) error
	GetBalance(address string) (*big.Int, error)
	CreateWallet() (*types.Wallet, error)
	Transfer(privateKeyStr string, fromAddress, toAddress string, value *big.Int, nonce uint64) (string, string, uint64, error)
	GetGasPrice() (string, error)
}

type Scheduler interface {
	BlockWorkerChan() chan uint64
	BlockWorkerReady(chan uint64)
	BlockSubmit(uint64)
	BlockRun()
	ReceiptWorkerChan() chan types.Transaction
	ReceiptWorkerReady(chan types.Transaction)
	ReceiptSubmit(types.Transaction)
	ReceiptRun()
	CollectionSendWorkerChan() chan db.WalletItem
	CollectionSendWorkerReady(c chan db.WalletItem)
	CollectionSendSubmit(c db.WalletItem)
	CollectionSendRun()
}

type ConCurrentEngine struct {
	scheduler Scheduler
	Worker    Worker
	// 添加新币的时候要修改这个
	Config   config.EngineConfig
	Protocol string
	CoinName string
	//DB          db.Database
	http        *client.HttpClient
	TransNotify sync.Map
}

// Run 启动
func (c *ConCurrentEngine) Run() {
	// 关闭连接
	//defer c.DB.Close()
	// 先重新获取一下最新的 BlockNum
	num, err := c.Worker.GetNowBlockNum()
	if err != nil {
		panic("ConCurrentEngine Run Err " + err.Error())
	}
	c.Config.BlockInit = num
	// 监听区块
	go c.blockLoop()
	//go c.collectionLoop()

	log.Info().Msgf("%s Listen Start ", c.Config.CoinName)
	select {}
}

// TODO 监听余额变换

// blockLoop 区块循环监听
func (c *ConCurrentEngine) blockLoop() {
	// 读取当前区块
	blockNumber := c.Config.BlockInit
	// 从 LevelDB 中读取数据 TODO 换掉 LevelDB
	// 不从数据库读取数据 每次启动是直接获取最新的
	//blockNumberStr, err := c.DB.Get("block_number")
	//if err == nil && blockNumberStr != "" {
	//	blockNumber, _ = strconv.ParseUint(blockNumberStr, 10, 64)
	//}

	// 区块信息
	c.scheduler.BlockRun()
	// 交易信息
	c.scheduler.ReceiptRun()
	// 归集信息
	//c.scheduler.CollectionSendRun()

	// 批量创建区块worker
	blockWorkerOut := make(chan types.Transaction)

	// 这里是在不断的读取区块
	c.createBlockWorker(blockWorkerOut)

	// 批量创建交易worker
	// TODO 免费 API 不支持大量的请求
	//for i := uint64(0); i < 5; i++ {
	c.createReceiptWorker()
	//}

	c.scheduler.BlockSubmit(blockNumber)

	go func() {
		for {
			// 不断从区块中 读取其中的所有交易
			transaction := <-blockWorkerOut
			//log.Info().Msgf("交易：%v", transaction)
			// 这里提交之后 ReceiptWorker 中会去读取
			c.scheduler.ReceiptSubmit(transaction)
		}
	}()
}

// collectionLoop 归集循环监听
//func (c *ConCurrentEngine) collectionLoop() {
//	n := new(big.Int)
//	collectionMax, ok := n.SetString(c.Config.CollectionMax, 10)
//	if !ok {
//		panic("setString: error")
//	}
//	// 配置大于0才去自动归集
//	if collectionMax.Cmp(big.NewInt(0)) > 0 {
//		// 启动归集
//		collectionWorkerOut := make(chan db.WalletItem)
//		c.createCollectionWorker(collectionWorkerOut)
//
//		// 启动归集发送worker
//		for i := uint64(0); i < c.Config.CollectionCount; i++ {
//			c.createCollectionSendWorker(collectionMax)
//		}
//
//		go func() {
//			for {
//				collectionSend := <-collectionWorkerOut
//				c.scheduler.CollectionSendSubmit(collectionSend)
//			}
//		}()
//	}
//}

// createBlockWorker 获取最新区块信息
func (c *ConCurrentEngine) createBlockWorker(out chan types.Transaction) {
	in := c.scheduler.BlockWorkerChan()
	go func() {
		for {
			// 意义是啥?
			c.scheduler.BlockWorkerReady(in)
			num := <-in
			log.Info().Msgf("%v，监听区块：%d", c.Config.CoinName, num)
			// 读取了区块中的交易
			transactions, blockNum, err := c.Worker.GetTransaction(num)
			if err != nil || blockNum == num {
				log.Info().Msgf("等待%d秒，当前已是最新区块", c.Config.BlockAfterTime)
				<-time.After(time.Duration(c.Config.BlockAfterTime) * time.Second)
				c.scheduler.BlockSubmit(num)
				continue
			}
			//err = c.DB.Put("block_number", strconv.FormatUint(blockNum, 10))
			if err != nil {
				c.scheduler.BlockSubmit(num)
			} else {
				c.scheduler.BlockSubmit(blockNum)
			}
			for _, transaction := range transactions {
				out <- transaction
			}
		}
	}()
}

// createReceiptWorker 创建获取区块信息的工作
func (c *ConCurrentEngine) createReceiptWorker() {
	// 好像每次返回的是一个新的
	// 返回的 receipt 其实就是 c.scheduler.ReceiptSubmit(transaction) 中每个区块传入的  transaction submit 一个 这边就会读取一个
	in := c.scheduler.ReceiptWorkerChan()
	eWorker := c.Worker.(*EthWorker)
	go func() {
		for {
			c.scheduler.ReceiptWorkerReady(in)
			// 这里的 in 是怎么读出数据的 ?
			transaction := <-in

			// 查询交易的情况 这里查询交易情况处理了 20 币的情况
			err := c.Worker.GetTransactionReceipt(&transaction)
			if err != nil {
				log.Info().Msgf("等待%d秒，收据信息无效, err: %v", c.Config.ReceiptAfterTime, err)
				<-time.After(time.Duration(c.Config.ReceiptAfterTime) * time.Second)
				c.scheduler.ReceiptSubmit(transaction)
			}
			// TODO 在这里比对数据库中需要监听的交易 Hash
			// 检测这里 若里面有存储的数据 则开始根据Hash查最新的区块 看其中有没有交易成功
			if len(eWorker.Pending) == 0 {
				continue
			}

			log.Info().Msgf("Find Transaction %s BlockNum is %v", transaction.Hash, transaction.BlockNumber)

			if _, ok := eWorker.Pending[transaction.Hash]; ok {
				worker := c.Worker.(*EthWorker)
				trans := worker.TransHistory[transaction.From]

				if transaction.Status != 1 {
					log.Error().Msgf("交易失败：%v", transaction.Hash)
					// TODO 发出通知
					for i := 0; i < len(trans); i++ {
						if trans[i].Hash == transaction.Hash {
							trans[i].Status = 0
							break
						}
					}
				} else {
					// TODO 将交易信息存储在中心化的服务器 方便后续的查询
					for i := 0; i < len(trans); i++ {
						if trans[i].Hash == transaction.Hash {
							trans[i].Status = 1
							break
						}
					}
					log.Info().Msgf(" %s 交易完成：%v", worker.token, transaction.Hash)
					// 存入交易成功的元素
					c.TransNotify.Store(transaction.Hash, struct{}{})
					// 是否打入内存中
					db.UpDateTransInfo(transaction.Hash, transaction.From, transaction.To, transaction.Value.String(), c.CoinName)
				}
				// TODO 并发
				delete(eWorker.Pending, transaction.Hash)
			}

		}
	}()
}

// createCollectionWorker 创建归集Worker
func (c *ConCurrentEngine) createCollectionWorker(out chan db.WalletItem) {
	go func() {
		for {
			<-time.After(time.Duration(c.Config.CollectionAfterTime) * time.Second)
			//list, err := c.DB.ListWallet(c.Config.WalletPrefix)
			//if err != nil {
			//	continue
			//}
			//for _, item := range list {
			//	out <- item
			//}
		}
	}()
}

// CreateWallet 创建钱包
func (c *ConCurrentEngine) CreateWallet() (string, error) {
	wallet, err := c.Worker.CreateWallet()
	user := db.NewWalletUser(wallet.Address, wallet.PrivateKey, wallet.PublicKey)
	if user != nil {
		_, err := db.Rdb.HSet(context.Background(), db.UserDB, wallet.Address, user).Result()
		if err != nil {
			log.Info().Msgf("写入钱包失败，地址：%v 异常: %s", wallet.Address, err.Error())
		}
	}
	if err != nil {
		return "", err
	}
	//_ = c.DB.Put(c.Config.WalletPrefix+wallet.Address, wallet.PrivateKey)
	log.Info().Msgf("创建钱包成功，地址：%v，私钥：%v", wallet.Address, wallet.PrivateKey)
	return wallet.Address, nil
}

// DeleteWallet 删除钱包
func (c *ConCurrentEngine) DeleteWallet(address string) error {
	//err := c.DB.Delete(c.Config.WalletPrefix + address)
	//if err != nil {
	//	return err
	//}
	return nil
}

// GetTransactionReceipt 获取交易状态
func (c *ConCurrentEngine) GetTransactionReceipt(hash string) (int, error) {

	t := &types.Transaction{
		Hash:   hash,
		Status: 0,
	}

	err := c.Worker.GetTransactionReceipt(t)
	if err != nil {
		return 0, err
	}

	return int(t.Status), nil
}

// NewEngine 创建ETH
func NewEngine(config config.EngineConfig, isNFT bool) (*ConCurrentEngine, error) {

	// TODO 后面优化掉
	//keyDB, err := db.NewKeyDB(config.File)
	//if err != nil {
	//	return nil, err
	//}

	var iworker Worker
	switch config.Protocol {
	case "eth":
		worker, err := NewEthWorker(config.Confirms, config.Contract, config.Rpc, isNFT)
		iworker = Worker(worker)
		if err != nil {
			return nil, err
		}
	}

	http := client.NewHttpClient(config.Protocol, config.CoinName, config.RechargeNotifyUrl, config.WithdrawNotifyUrl)

	return &ConCurrentEngine{
		//scheduler: scheduler.NewSimpleScheduler(), // 简单的任务调度器
		scheduler: scheduler.NewQueueScheduler(), // 队列的任务调度器
		Worker:    iworker,
		Config:    config,
		Protocol:  config.Protocol,
		CoinName:  config.CoinName,
		//DB:        keyDB,
		http: http,
	}, nil
}

func AddNewCoin(coinName, contractAddress string) (*ConCurrentEngine, error) {

	eng, err := NewEngine(config.EngineConfig{
		CoinName: coinName,
		// 关键是这个 Address 通过这个 Address 来进行合约的标准操作
		Contract: contractAddress,
		Protocol: "eth",
		Rpc:      "https://polygon-mumbai.blockpi.network/v1/rpc/public",
		//File:              "data/" + coinName,
		WalletPrefix:      "wallet-",
		HashPrefix:        "hash-",
		BlockInit:         0,
		BlockAfterTime:    10,
		ReceiptCount:      20,
		ReceiptAfterTime:  10,
		RechargeNotifyUrl: "http://localhost:10001/api/withdraw",
		WithdrawNotifyUrl: "http://localhost:10001/api/withdraw",
	}, false)
	if err != nil {
		log.Info().Msgf("AddNewCoin err is %s ", err.Error())
		return nil, err
	}
	// 开启这个新协议的监听
	go eng.Run()
	return eng, nil
}

// CheckMulSign 检查是否开启多签 是则直接进行多签的逻辑
func CheckMulSign() {

}
