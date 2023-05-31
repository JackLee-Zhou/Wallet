package engine

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/lmxdawn/wallet/types"
	"math/big"
	"strings"
	"sync"
)

type EthWorker struct {
	confirms               uint64 // 需要的确认数
	http                   *ethclient.Client
	token                  string // 代币合约地址，为空表示主币
	tokenTransferEventHash common.Hash
	tokenAbi               abi.ABI     // 合约的abi
	Pending                chan string // 待执行的交易
	nonceLock              sync.Mutex
	TransHistory           map[string][]*types.Transaction // 交易历史记录
}

// GetGasPrice 获取最新的燃料价格
func (e *EthWorker) GetGasPrice() (string, error) {
	// 能这样做 ?
	price, err := e.http.SuggestGasPrice(context.Background())
	if err != nil {
		return "", err
	}
	return price.String(), nil
}

func NewEthWorker(confirms uint64, contract string, url string) (*EthWorker, error) {
	http, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}

	tokenTransferEventHashSig := []byte("Transfer(address,address,uint256)")
	tokenTransferEventHash := crypto.Keccak256Hash(tokenTransferEventHashSig)
	tokenAbiStr := "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"
	tokenAbi, err := abi.JSON(strings.NewReader(tokenAbiStr))
	if err != nil {
		return nil, err
	}

	return &EthWorker{
		confirms:               confirms,
		token:                  contract,
		http:                   http,
		tokenTransferEventHash: tokenTransferEventHash,
		tokenAbi:               tokenAbi,
		Pending:                make(chan string, 64), // 大小
		TransHistory:           make(map[string][]*types.Transaction),
	}, nil
}

// GetNowBlockNum 获取最新块
func (e *EthWorker) GetNowBlockNum() (uint64, error) {

	blockNumber, err := e.http.BlockNumber(context.Background())
	if err != nil {
		return 0, err
	}
	return blockNumber, nil
}

// GetTransactionReceipt 获取交易的票据
func (e *EthWorker) GetTransactionReceipt(transaction *types.Transaction) error {

	hash := common.HexToHash(transaction.Hash)

	receipt, err := e.http.TransactionReceipt(context.Background(), hash)
	if err != nil {
		return err
	}

	// 获取最新区块
	latest, err := e.http.BlockNumber(context.Background())
	if err != nil {
		return err
	}

	// 判断确认数
	confirms := latest - receipt.BlockNumber.Uint64() + 1
	if confirms < e.confirms {
		return errors.New("the number of confirmations is not satisfied")
	}

	status := receipt.Status
	transaction.Status = uint(status)

	return nil
}

// GetTransaction 获取交易信息
func (e *EthWorker) GetTransaction(num uint64) ([]types.Transaction, uint64, error) {

	// 获取的是最新的区块
	nowBlockNumber, err := e.GetNowBlockNum()
	if err != nil {
		return nil, num, err
	}
	toBlock := num + 100
	// 传入的num为0，表示最新块
	if num == 0 {
		toBlock = nowBlockNumber
	} else if toBlock > nowBlockNumber {
		toBlock = nowBlockNumber
	}
	if e.token == "" {
		return e.getBlockTransaction(num)
	} else {
		return e.getTokenTransaction(num, toBlock)
	}

}

// getBlockTransaction 获取主币的交易信息
func (e *EthWorker) getBlockTransaction(num uint64) ([]types.Transaction, uint64, error) {

	block, err := e.http.BlockByNumber(context.Background(), big.NewInt(int64(num)))
	if err != nil {
		return nil, num, err
	}

	chainID, err := e.http.NetworkID(context.Background())
	if err != nil {
		return nil, num, err
	}

	var transactions []types.Transaction
	for _, tx := range block.Transactions() {
		// 如果接收方地址为空，则是创建合约的交易，忽略过去
		if tx.To() == nil {
			continue
		}
		msg, err := tx.AsMessage(ethTypes.LatestSignerForChainID(chainID), tx.GasPrice())
		if err != nil {
			continue
		}
		transactions = append(transactions, types.Transaction{
			BlockNumber: big.NewInt(int64(num)),
			BlockHash:   block.Hash().Hex(),
			Hash:        tx.Hash().Hex(),
			From:        msg.From().Hex(),
			To:          tx.To().Hex(),
			Value:       tx.Value(),
		})
	}
	return transactions, num + 1, nil
}

// getTokenTransaction 获取代币的交易信息
func (e *EthWorker) getTokenTransaction(num uint64, toBlock uint64) ([]types.Transaction, uint64, error) {
	contractAddress := common.HexToAddress(e.token)
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(num)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: []common.Address{
			contractAddress,
		},
	}
	logs, err := e.http.FilterLogs(context.Background(), query)
	if err != nil {
		return nil, num, err
	}

	var transactions []types.Transaction
	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case e.tokenTransferEventHash:

			tokenTransfer := struct {
				From  common.Address
				To    common.Address
				Value *big.Int
			}{}

			err = e.tokenAbi.UnpackIntoInterface(&tokenTransfer, "Transfer", vLog.Data)
			if err != nil {
				continue
			}

			transactions = append(transactions, types.Transaction{
				BlockNumber: big.NewInt(int64(num)),
				BlockHash:   vLog.BlockHash.Hex(),
				Hash:        vLog.TxHash.Hex(),
				From:        strings.ToLower(common.HexToAddress(vLog.Topics[1].Hex()).Hex()),
				To:          strings.ToLower(common.HexToAddress(vLog.Topics[2].Hex()).Hex()),
				Value:       tokenTransfer.Value,
			})
		}
	}

	return transactions, toBlock, nil
}

// GetBalance 获取余额
func (e *EthWorker) GetBalance(address string) (*big.Int, error) {

	// 如果不是合约
	account := common.HexToAddress(address)
	if e.token == "" {
		balance, err := e.http.BalanceAt(context.Background(), account, nil)
		if err != nil {
			return nil, err
		}
		return balance, nil
	} else {
		res, err := e.callContract(e.token, "balanceOf", account)
		if err != nil {
			return nil, err
		}
		balance := big.NewInt(0)
		balance.SetBytes(res)
		return balance, nil
	}

}

// CreateWallet 创建钱包
func (e *EthWorker) CreateWallet() (*types.Wallet, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	privateKeyBytes := crypto.FromECDSA(privateKey)

	privateKeyString := hexutil.Encode(privateKeyBytes)[2:]

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	publicKeyString := hexutil.Encode(publicKeyBytes)[4:]

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

	return &types.Wallet{
		Address:    address,
		PublicKey:  publicKeyString,
		PrivateKey: privateKeyString,
	}, err
}

// GetAddressByPrivateKey 根据私钥获取地址
func (e *EthWorker) GetAddressByPrivateKey(privateKeyStr string) (string, error) {

	privateKey, err := crypto.HexToECDSA(privateKeyStr)
	if err != nil {
		return "", err
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	return fromAddress.Hex(), nil
}

// callContract 查询智能合约
func (e *EthWorker) callContract(contractAddress string, method string, params ...interface{}) ([]byte, error) {

	input, _ := e.tokenAbi.Pack(method, params...)

	to := common.HexToAddress(contractAddress)
	msg := ethereum.CallMsg{
		To:   &to,
		Data: input,
	}

	hex, err := e.http.CallContract(context.Background(), msg, nil)

	if err != nil {
		return nil, err
	}

	return hex, nil
}

// Transfer 转账
func (e *EthWorker) Transfer(privateKeyStr string, toAddress string, value *big.Int, nonce uint64) (string, string, uint64, error) {

	var data []byte
	var err error
	var value20 *big.Int
	if e.token != "" {
		contractTransferHashSig := []byte("transfer(address,uint256)")
		contractTransferHash := crypto.Keccak256Hash(contractTransferHashSig)
		toAddressTmp := common.HexToAddress(toAddress)
		toAddressHex := &toAddressTmp
		data, err = makeEthERC20TransferData(contractTransferHash, toAddressHex, value)
		if err != nil {
			return "", "", 0, err
		}
		value20 = value
		value = big.NewInt(0)
	}

	return e.sendTransaction(e.token, privateKeyStr, toAddress, value, value20, nonce, data)
}

// sendTransaction 创建并发送交易
func (e *EthWorker) sendTransaction(contractAddress string, privateKeyStr string,
	toAddress string, value *big.Int, value20 *big.Int, nonce uint64, data []byte) (string, string, uint64, error) {
	var trueValue *big.Int
	trueValue = value
	privateKey, err := crypto.HexToECDSA(privateKeyStr)
	if err != nil {
		return "", "", 0, err
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", "", 0, errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	if nonce <= 0 {
		// 解决 nonce 竞争问题
		e.nonceLock.Lock()
		defer e.nonceLock.Unlock()
		nonce, err = e.http.PendingNonceAt(context.Background(), fromAddress)
		if err != nil {
			return "", "", 0, err
		}

	}

	var gasLimit uint64
	gasLimit = uint64(21000) // in units
	gasPrice, err := e.http.SuggestGasPrice(context.Background())
	if err != nil {
		return "", "", 0, err
	}
	gasTip, err := e.http.SuggestGasTipCap(context.Background())
	if err != nil {
		return "", "", 0, err
	}
	var toAddressHex *common.Address
	if toAddress != "" {
		toAddressTmp := common.HexToAddress(toAddress)
		toAddressHex = &toAddressTmp
	}

	if contractAddress != "" {
		value = big.NewInt(0)
		trueValue = value20
		contractAddressHex := common.HexToAddress(contractAddress)
		toAddressHex = &contractAddressHex
	}

	//b, err := e.http.BlockByNumber(context.Background(), nil)
	//if err != nil {
	//	return "", "", 0, err
	//}
	//
	//gasPrice = gasPrice.Add(gasPrice, b.BaseFee())
	txData := &ethTypes.DynamicFeeTx{
		Nonce: nonce,
		To:    toAddressHex,
		Value: value,
		Gas:   gasLimit,
		//GasPrice: gasPrice.Add(gasPrice, big.NewInt(100000000)),
		GasFeeCap: gasPrice,
		GasTipCap: gasTip,
		Data:      data,
	}

	//ethTypes.DynamicFeeTx{}

	tx := ethTypes.NewTx(txData)

	chainID, err := e.http.NetworkID(context.Background())
	if err != nil {
		return "", "", 0, err
	}

	// 签名
	signTx, err := ethTypes.SignTx(tx, ethTypes.LatestSignerForChainID(chainID), privateKey)
	if err != nil {
		return "", "", 0, err
	}

	err = e.http.SendTransaction(context.Background(), signTx)
	if err != nil {
		return "", "", 0, err
	}

	// TODO 打入管道中  createReceiptWorker 中去处理监听交易是否成功

	e.Pending <- tx.Hash().String()

	// 要落地
	e.TransHistory[fromAddress.String()] = append(e.TransHistory[fromAddress.String()], &types.Transaction{
		Hash:   tx.Hash().String(),
		From:   fromAddress.String(),
		To:     txData.To.String(),
		Value:  trueValue,
		Status: uint(0),
	})

	return fromAddress.Hex(), signTx.Hash().Hex(), nonce, nil
}

// TransactionMethod 获取某个交易执行的方法
func (e *EthWorker) TransactionMethod(hash string) ([]byte, error) {

	tx, _, err := e.http.TransactionByHash(context.Background(), common.HexToHash(hash))
	if err != nil {
		return nil, err
	}

	data := tx.Data()

	return data[0:4], nil
}

func makeEthERC20TransferData(contractTransferHash common.Hash, toAddress *common.Address, amount *big.Int) ([]byte, error) {
	var data []byte
	data = append(data, contractTransferHash[:4]...)
	paddedAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
	data = append(data, paddedAddress...)
	paddedAmount := common.LeftPadBytes(amount.Bytes(), 32)
	data = append(data, paddedAmount...)
	return data, nil
}
