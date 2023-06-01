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
	"strconv"
	"time"
)

type Worker interface {
	GetNowBlockNum() (uint64, error)
	GetTransaction(num uint64) ([]types.Transaction, uint64, error)
	GetTransactionReceipt(*types.Transaction) error
	GetBalance(address string) (*big.Int, error)
	CreateWallet() (*types.Wallet, error)
	Transfer(privateKeyStr string, toAddress string, value *big.Int, nonce uint64) (string, string, uint64, error)
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
	Config      config.EngineConfig
	Protocol    string
	CoinName    string
	DB          db.Database
	http        *client.HttpClient
	TransNotify map[string]struct{}
}

// Run 启动
func (c *ConCurrentEngine) Run() {
	// 关闭连接
	defer c.DB.Close()
	// 先重新获取一下最新的 BlockNum
	num, err := c.Worker.GetNowBlockNum()
	if err != nil {
		panic("ConCurrentEngine Run Err " + err.Error())
	}
	c.Config.BlockInit = num
	// 监听区块
	go c.blockLoop()
	go c.collectionLoop()

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
	for i := uint64(0); i < c.Config.ReceiptCount; i++ {
		c.createReceiptWorker()
	}

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
func (c *ConCurrentEngine) collectionLoop() {
	n := new(big.Int)
	collectionMax, ok := n.SetString(c.Config.CollectionMax, 10)
	if !ok {
		panic("setString: error")
	}
	// 配置大于0才去自动归集
	if collectionMax.Cmp(big.NewInt(0)) > 0 {
		// 启动归集
		collectionWorkerOut := make(chan db.WalletItem)
		c.createCollectionWorker(collectionWorkerOut)

		// 启动归集发送worker
		for i := uint64(0); i < c.Config.CollectionCount; i++ {
			c.createCollectionSendWorker(collectionMax)
		}

		go func() {
			for {
				collectionSend := <-collectionWorkerOut
				c.scheduler.CollectionSendSubmit(collectionSend)
			}
		}()
	}
}

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
			err = c.DB.Put("block_number", strconv.FormatUint(blockNum, 10))
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

			// 查询交易的情况
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
			//log.Info().Msgf("Block Num is %v ", transaction.BlockNumber)

			if _, ok := eWorker.Pending[transaction.Hash]; ok {
				worker := c.Worker.(*EthWorker)
				trans := worker.TransHistory[transaction.From]

				if transaction.Status != 1 {
					log.Error().Msgf("交易失败：%v", transaction.Hash)
					// TODO 发出通知
					for _, v := range trans {
						temp := v
						if temp.Hash == transaction.Hash {
							temp.Status = 0
							break
						}
					}
				} else {
					// TODO 将交易信息存储在中心化的服务器 方便后续的查询
					for _, v := range trans {
						temp := v
						if temp.Hash == transaction.Hash {
							temp.Status = 1
							break
						}
					}
					c.TransNotify[transaction.Hash] = struct{}{}
					log.Info().Msgf("交易完成：%v", transaction.Hash)
					// 删除头部元素
				}
				delete(eWorker.Pending, transaction.Hash)
				//db.UpDataUserTransInfo(transaction.From,transaction.,trans)
			}

		}
	}()
}

// createCollectionWorker 创建归集Worker
func (c *ConCurrentEngine) createCollectionWorker(out chan db.WalletItem) {
	go func() {
		for {
			<-time.After(time.Duration(c.Config.CollectionAfterTime) * time.Second)
			list, err := c.DB.ListWallet(c.Config.WalletPrefix)
			if err != nil {
				continue
			}
			for _, item := range list {
				out <- item
			}
		}
	}()
}

// collectionSendWorker 创建归集发送交易的worker
func (c *ConCurrentEngine) createCollectionSendWorker(max *big.Int) {
	in := c.scheduler.CollectionSendWorkerChan()
	go func() {
		for {
			c.scheduler.CollectionSendWorkerReady(in)
			collectionSend := <-in
			_, err := c.collection(collectionSend.Address, collectionSend.PrivateKey, max)
			if err != nil {
				// 归集失败，重新加入归集队列
				c.scheduler.CollectionSendSubmit(collectionSend)
				continue
			}
		}
	}()
}

// 归集
func (c ConCurrentEngine) collection(address, privateKey string, max *big.Int) (*big.Int, error) {
	balance, err := c.Worker.GetBalance(address)
	if err != nil {
		return nil, err
	}
	if balance.Cmp(max) < 0 {
		return big.NewInt(0), nil
	}

	// 开始归集
	_, _, _, err = c.Worker.Transfer(privateKey, c.Config.CollectionAddress, balance, 0)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

// Collection 归集某个地址
func (c *ConCurrentEngine) Collection(address string, max *big.Int) (*big.Int, error) {

	// 查询地址是否存在
	privateKey, err := c.DB.Get(c.Config.WalletPrefix + address)
	if err != nil {
		return nil, err
	}

	balance, err := c.collection(address, privateKey, max)
	if err != nil {
		return nil, err
	}

	return balance, nil
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
	_ = c.DB.Put(c.Config.WalletPrefix+wallet.Address, wallet.PrivateKey)
	log.Info().Msgf("创建钱包成功，地址：%v，私钥：%v", wallet.Address, wallet.PrivateKey)
	return wallet.Address, nil
}

// DeleteWallet 删除钱包
func (c *ConCurrentEngine) DeleteWallet(address string) error {
	err := c.DB.Delete(c.Config.WalletPrefix + address)
	if err != nil {
		return err
	}
	return nil
}

// Withdraw 提现
func (c *ConCurrentEngine) Withdraw(orderId string, toAddress string, value int64) (string, error) {

	_, hash, _, err := c.Worker.Transfer(c.Config.WithdrawPrivateKey, toAddress, big.NewInt(value), 0)
	if err != nil {
		return "", err
	}
	_ = c.DB.Put(c.Config.HashPrefix+hash, orderId)
	return hash, nil
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
func NewEngine(config config.EngineConfig) (*ConCurrentEngine, error) {
	keyDB, err := db.NewKeyDB(config.File)
	if err != nil {
		return nil, err
	}

	var worker Worker
	switch config.Protocol {
	case "eth":
		worker, err = NewEthWorker(config.Confirms, config.Contract, config.Rpc)
		if err != nil {
			return nil, err
		}
	}

	http := client.NewHttpClient(config.Protocol, config.CoinName, config.RechargeNotifyUrl, config.WithdrawNotifyUrl)

	return &ConCurrentEngine{
		//scheduler: scheduler.NewSimpleScheduler(), // 简单的任务调度器
		scheduler:   scheduler.NewQueueScheduler(), // 队列的任务调度器
		Worker:      worker,
		Config:      config,
		Protocol:    config.Protocol,
		CoinName:    config.CoinName,
		DB:          keyDB,
		http:        http,
		TransNotify: make(map[string]struct{}),
	}, nil
}

func AddNewCoin(coinName, contractAddress string) error {

	eng, err := NewEngine(config.EngineConfig{
		CoinName:          coinName,
		Contract:          contractAddress,
		Protocol:          "eth",
		File:              "data/eth",
		WalletPrefix:      "wallet-",
		HashPrefix:        "hash-",
		BlockInit:         0,
		BlockAfterTime:    1,
		ReceiptCount:      20,
		ReceiptAfterTime:  1,
		RechargeNotifyUrl: "http://localhost:10001/api/withdraw",
		WithdrawNotifyUrl: "http://localhost:10001/api/withdraw",
	})
	if err != nil {
		log.Info().Msgf("AddNewCoin err is %s ", err.Error())
		return err
	}
	// 开启这个新协议的监听
	go eng.Run()
	return nil
}

// CheckMulSign 检查是否开启多签 是则直接进行多签的逻辑
func CheckMulSign() {

}
