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
	"github.com/rs/zerolog/log"
	"math/big"
	"strings"
	"sync"
)

type MulSignCounter struct {
	Counter int32
	Lock    sync.Locker
}

var signCounter *MulSignCounter

type Worker struct {
	confirms               uint64 // 需要的确认数
	http                   *ethclient.Client
	tokenTransferEventHash common.Hash
	tokenAbi               abi.ABI // 合约的abi
	// pendingList 未完成的交易列表
	pending *sync.Map
	//Pending                map[string]struct{} // 待执行的交易
	nonceLock sync.Mutex
	//TransHistory           map[string][]*types.Transaction // 交易历史记录
}

// EWorker 全局的worker 考虑切链 加Map
var EWorker *Worker

// IsContract 判断是否是合约地址
func (w *Worker) IsContract(address string) bool {
	byteCode, err := w.http.CodeAt(context.Background(), common.HexToAddress(address), nil)
	if err != nil {
		log.Error().Msgf("IsContract err is %s", err.Error())
		return false
	}

	// 存在 code
	if len(byteCode) > 0 {
		return true
	}
	return false
}

// GetPendingByHex 通过交易的hex获取处于Pending状态的交易
func (w *Worker) GetPendingByHex(txHex string) *types.Transaction {
	pend, ok := w.pending.Load(txHex)
	if !ok {
		log.Info().Msgf("GetPendingByHex target is not exist %s ", txHex)
		return nil
	}
	return pend.(*types.Transaction)

}

// RemovePendingByHex 通过交易的hex删除处于Pending状态的交易
func (w *Worker) RemovePendingByHex(txHex string) {

	// 检查一下是否存在
	_, ok := w.pending.Load(txHex)
	if !ok {
		log.Info().Msgf("RemovePendingByHex target is not exist %s ", txHex)
		return
	}
	w.pending.Delete(txHex)
	log.Info().Msgf("RemovePendingByHex target is %s ", txHex)
}

func NewWorker(confirms uint64, url string) error {
	http, err := ethclient.Dial(url)
	if err != nil {
		return err
	}
	var tokenTransferEventHashSig []byte
	var tokenTransferEventHash common.Hash
	var tokenAbiStr string

	tokenTransferEventHashSig = []byte("Transfer(address,uint256)")
	tokenTransferEventHash = crypto.Keccak256Hash(tokenTransferEventHashSig)
	tokenAbiStr = "[\n\t{\n\t\t\"inputs\": [],\n\t\t\"stateMutability\": \"nonpayable\",\n\t\t\"type\": \"constructor\"\n\t},\n\t{\n\t\t\"anonymous\": false,\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"indexed\": true,\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"owner\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"indexed\": true,\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"spender\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"indexed\": false,\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"value\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"Approval\",\n\t\t\"type\": \"event\"\n\t},\n\t{\n\t\t\"anonymous\": false,\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"indexed\": true,\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"from\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"indexed\": true,\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"to\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"indexed\": false,\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"value\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"Transfer\",\n\t\t\"type\": \"event\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"owner\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"spender\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"allowance\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"view\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"spender\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"amount\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"approve\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"bool\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"bool\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"nonpayable\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"account\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"balanceOf\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"view\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [],\n\t\t\"name\": \"decimals\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint8\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"uint8\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"view\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"spender\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"subtractedValue\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"decreaseAllowance\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"bool\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"bool\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"nonpayable\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"spender\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"addedValue\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"increaseAllowance\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"bool\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"bool\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"nonpayable\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [],\n\t\t\"name\": \"name\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"string\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"string\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"view\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [],\n\t\t\"name\": \"owner\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"view\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [],\n\t\t\"name\": \"symbol\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"string\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"string\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"view\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [],\n\t\t\"name\": \"totalSupply\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"view\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"to\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"amount\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"transfer\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"bool\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"bool\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"nonpayable\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"from\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"to\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"amount\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"transferFrom\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"bool\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"bool\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"nonpayable\",\n\t\t\"type\": \"function\"\n\t}\n]"

	tokenAbi, err := abi.JSON(strings.NewReader(tokenAbiStr))
	if err != nil {
		return err
	}
	EWorker = &Worker{
		confirms:               confirms,
		http:                   http,
		tokenTransferEventHash: tokenTransferEventHash,
		tokenAbi:               tokenAbi,
		pending:                &sync.Map{},
		//TransHistory:           make(map[string][]*types.Transaction),
	}
	return nil
}
func (w *Worker) GetNowBlockNum() (uint64, error) {
	blockNumber, err := w.http.BlockNumber(context.Background())
	if err != nil {
		return 0, err
	}
	return blockNumber, nil
}

func (w *Worker) GetTransaction(num uint64) ([]types.Transaction, uint64, error) {
	// 获取的是最新的区块
	nowBlockNumber, err := w.GetNowBlockNum()
	if err != nil {
		return nil, num, err
	}
	toBlock := num + 100
	// 传入的num为0，表示最新块
	if num == 0 {
		// 表示从创世区块开始遍历
		toBlock = nowBlockNumber
	} else if toBlock > nowBlockNumber {
		toBlock = nowBlockNumber
	}
	//if w.token == "" {
	//	return e.getBlockTransaction(num)
	//} else {
	//	return e.getTokenTransaction(num, toBlock)
	//}
	return nil, 0, err
}

func (w *Worker) GetTransactionReceipt(transactionHash string) (int64, error) {
	hash := common.HexToHash(transactionHash)

	receipt, err := w.http.TransactionReceipt(context.Background(), hash)
	if err != nil {
		return 0, err
	}

	// 获取最新区块
	latest, err := w.http.BlockNumber(context.Background())
	if err != nil {
		return 0, err
	}

	// 判断确认数
	confirms := latest - receipt.BlockNumber.Uint64() + 1
	if confirms < w.confirms {
		return 0, errors.New("the number of confirmations is not satisfied")
	}

	status := receipt.Status
	//transaction.Status = uint(status)

	return int64(status), nil
}

func (w *Worker) GetBalance(address string, contractAddress string) (*big.Int, error) {
	//如果不是合约
	account := common.HexToAddress(address)
	if contractAddress == "" {
		balance, err := w.http.BalanceAt(context.Background(), account, nil)
		if err != nil {
			return nil, err
		}
		return balance, nil
	} else {
		res, err := w.callContract(contractAddress, "balanceOf", account)
		if err != nil {
			return nil, err
		}
		balance := big.NewInt(0)
		balance.SetBytes(res)
		return balance, nil
	}
	return nil, nil
}

// GeneratePublicKey 生成公钥
func GeneratePublicKey(private string) (string, error) {
	privateKey, err := crypto.HexToECDSA(private)
	if err != nil {
		return "", err
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	publicKeyStr := hexutil.Encode(publicKeyBytes)[4:]
	return publicKeyStr, nil

}

func (w *Worker) CreateWallet() (*types.Wallet, error) {
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

func (w *Worker) Transfer(privateKeyStr string, toAddress string, value *big.Int, nonce uint64, contractAddress string) (string, string, uint64, error) {
	var data []byte
	var err error
	var value20 *big.Int
	var contractTransferHashSig []byte
	var contractTransferHash common.Hash
	var toAddressTmp common.Address
	var toAddressHex *common.Address

	if contractAddress != "" {

		contractTransferHashSig = []byte("transfer(address,uint256)")
		contractTransferHash = crypto.Keccak256Hash(contractTransferHashSig)
		toAddressTmp = common.HexToAddress(toAddress)
		toAddressHex = &toAddressTmp
		data, err = makeEthERC20TransferData(contractTransferHash, toAddressHex, value)
		if err != nil {
			return "", "", 0, err
		}
		value20 = value
		value = big.NewInt(0)

	}

	// NFT 转账的时候  value 是 tokenID 值
	return w.sendTransaction(contractAddress, privateKeyStr, toAddress, value, value20, nonce, data)
}

func (w *Worker) GetGasPrice() (string, error) {
	// 能这样做 ?
	price, err := w.http.SuggestGasPrice(context.Background())
	if err != nil {
		return "", err
	}
	return price.String(), nil
}

// getBlockTransaction 获取主币的交易信息
func (w *Worker) getBlockTransaction(num uint64) ([]types.Transaction, uint64, error) {

	block, err := w.http.BlockByNumber(context.Background(), big.NewInt(int64(num)))
	if err != nil {
		return nil, num, err
	}

	chainID, err := w.http.NetworkID(context.Background())
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
func (w *Worker) getTokenTransaction(num uint64, toBlock uint64, contract string) ([]types.Transaction, uint64, error) {
	contractAddress := common.HexToAddress(contract)
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(num)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: []common.Address{
			contractAddress,
		},
	}
	logs, err := w.http.FilterLogs(context.Background(), query)
	if err != nil {
		return nil, num, err
	}

	var transactions []types.Transaction
	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case w.tokenTransferEventHash:

			tokenTransfer := struct {
				From  common.Address
				To    common.Address
				Value *big.Int
			}{}

			err = w.tokenAbi.UnpackIntoInterface(&tokenTransfer, "Transfer", vLog.Data)
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

// GetAddressByPrivateKey 根据私钥获取地址
func (w *Worker) GetAddressByPrivateKey(privateKeyStr string) (string, error) {

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

// callContract 调用合约 这里可以传入自定的 method 和 params
func (w *Worker) callContract(contractAddress string, method string, params ...interface{}) ([]byte, error) {
	input, _ := w.tokenAbi.Pack(method, params...)

	to := common.HexToAddress(contractAddress)
	msg := ethereum.CallMsg{
		To:   &to,
		Data: input,
	}

	// 直接 call 合约 用于查询操作 不上链的
	hex, err := w.http.CallContract(context.Background(), msg, nil)

	if err != nil {
		return nil, err
	}

	return hex, nil
}

// SendContractTrans 发送合约交易
func (w *Worker) SendContractTrans(privateKeyStr string, tx *ethTypes.DynamicFeeTx) (string, string, uint64, error) {
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
	gasLimit, err := w.http.EstimateGas(context.Background(), ethereum.CallMsg{
		From:  fromAddress,
		Value: tx.Value,
		To:    tx.To,
		Data:  tx.Data,
	})
	if err != nil {
		log.Error().Msgf("EstimateGas error: %s", err.Error())
		return "", "", 0, err
	}
	gasTip, err := w.http.SuggestGasTipCap(context.Background())
	if err != nil {
		return "", "", 0, err
	}
	gasPrice, err := w.http.SuggestGasPrice(context.Background())
	if err != nil {
		return "", "", 0, err
	}
	w.nonceLock.Lock()
	defer w.nonceLock.Unlock()
	tx.Nonce, err = w.http.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return "", "", 0, err
	}
	tx.GasTipCap = gasTip
	tx.GasFeeCap = gasPrice
	tx.Gas = gasLimit * 2
	txData := ethTypes.NewTx(tx)
	log.Info().Msgf("tx: %+v", tx)
	chainID, err := w.http.NetworkID(context.Background())
	if err != nil {
		return "", "", 0, err
	}

	// 签名
	signTx, err := ethTypes.SignTx(txData, ethTypes.LatestSignerForChainID(chainID), privateKey)
	if err != nil {
		return "", "", 0, err
	}

	err = w.http.SendTransaction(context.Background(), signTx)
	if err != nil {
		return "", "", 0, err
	}

	ts := &types.Transaction{
		Hash:      txData.Hash().String(),
		From:      fromAddress.String(),
		To:        tx.To.String(),
		Value:     tx.Value,
		Status:    uint(0),
		Data:      tx.Data,
		Nonce:     tx.Nonce,
		Gas:       tx.Gas,
		GasFeeCap: tx.GasFeeCap,
		GasTipCap: tx.GasTipCap,
	}
	w.pending.Store(ts.Hash, ts)
	return fromAddress.Hex(), signTx.Hash().Hex(), tx.Nonce, nil
}

func (w *Worker) sendTransaction(contractAddress string, privateKeyStr string,
	toAddress string, value *big.Int, value20 *big.Int, nonce uint64, data []byte, trans ...*types.Transaction) (string, string, uint64, error) {
	//var trueValue *big.Int
	//trueValue = value
	txData := &ethTypes.DynamicFeeTx{}
	var toAddressHex *common.Address
	var gasLimit uint64
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
	gasTip, err := w.http.SuggestGasTipCap(context.Background())
	if err != nil {
		return "", "", 0, err
	}
	gasPrice, err := w.http.SuggestGasPrice(context.Background())
	if err != nil {
		return "", "", 0, err
	}

	// 加速减速或者取消
	if len(trans) != 0 {
		nonce, err = w.http.PendingNonceAt(context.Background(), fromAddress)
		// 暂时只支持单一的操作
		pend := trans[0]
		toAddressTmp := common.HexToAddress(pend.To)
		toAddressHex = &toAddressTmp
		//txData.Data = pend.Data
		txData.Gas = uint64(28000)
		txData.GasFeeCap = gasPrice.Mul(gasPrice, big.NewInt(2))
		txData.GasTipCap = gasTip.Mul(gasTip, big.NewInt(2))
		txData.To = toAddressHex
		//txData.Value = pend.Value
		txData.Nonce = nonce
	} else {
		if nonce <= 0 {
			// 解决 nonce 竞争问题
			w.nonceLock.Lock()
			defer w.nonceLock.Unlock()
			nonce, err = w.http.PendingNonceAt(context.Background(), fromAddress)
			if err != nil {
				return "", "", 0, err
			}

		}

		//gasLimit = uint64(21000) // 在非合约中的转账 21000 是够的 但是在合约中 这个限制太小
		gasLimit = uint64(28000) //

		if contractAddress != "" {
			toAddressTmp := common.HexToAddress(contractAddress)
			toAddressHex = &toAddressTmp
		} else {
			toAddressTmp := common.HexToAddress(toAddress)
			toAddressHex = &toAddressTmp
		}
		// 20 币和 NFT 交易都可以通过这里 做处理 防止 value 传值错误
		//if contractAddress != "" {
		//	value = big.NewInt(0)
		//
		//	// 这里 因为是转移20代币 转账的操作是发送给合约的，由合约内部进行 transfer 操作，所以这笔交易是发送给 合约的
		//	contractAddressHex := common.HexToAddress(contractAddress)
		//	toAddressHex = &contractAddressHex
		//}
		txData = &ethTypes.DynamicFeeTx{
			Nonce: nonce,
			To:    toAddressHex,
			Value: value,
			Gas:   gasLimit,
			// 最高的 gas 费
			GasFeeCap: gasPrice,
			// 最高小费单价
			GasTipCap: gasTip,
			Data:      data,
		}
	}

	//ethTypes.DynamicFeeTx{}

	tx := ethTypes.NewTx(txData)

	chainID, err := w.http.NetworkID(context.Background())
	if err != nil {
		return "", "", 0, err
	}

	// 签名
	signTx, err := ethTypes.SignTx(tx, ethTypes.LatestSignerForChainID(chainID), privateKey)
	if err != nil {
		return "", "", 0, err
	}

	err = w.http.SendTransaction(context.Background(), signTx)
	if err != nil {
		return "", "", 0, err
	}

	//w.Pending[signTx.Hash().Hex()] = struct{}{}

	// 要落地
	ts := &types.Transaction{
		Hash: tx.Hash().String(),
		From: fromAddress.String(),
		To:   txData.To.String(),
		//Value:  trueValue,
		Status:    uint(0),
		Data:      data,
		Nonce:     nonce,
		Gas:       txData.Gas,
		GasFeeCap: txData.GasFeeCap,
		GasTipCap: txData.GasTipCap,
	}
	//e.TransHistory[fromAddress.String()] = append(e.TransHistory[fromAddress.String()], &types.Transaction{
	//	Hash: tx.Hash().String(),
	//	From: fromAddress.String(),
	//	To:   txData.To.String(),
	//	//Value:  trueValue,
	//	Status: uint(0),
	//})
	w.pending.Store(signTx.Hash().Hex(), ts)

	return fromAddress.Hex(), signTx.Hash().Hex(), nonce, nil
}

func (w *Worker) TransactionMethod(hash string) ([]byte, error) {

	tx, _, err := w.http.TransactionByHash(context.Background(), common.HexToHash(hash))
	if err != nil {
		return nil, err
	}

	data := tx.Data()

	return data[0:4], nil
}

// UpPackTransfer 解析出 20 币交易的 input
func (w *Worker) UpPackTransfer(data []byte) *types.Transfer {
	res := &types.Transfer{}
	if method, ok := w.tokenAbi.Methods["transfer"]; ok {
		params, err := method.Inputs.Unpack(data[4:])
		if err != nil {
			log.Info().Msgf("unpack error %s", err.Error())
			return nil
		}

		to := params[0].(common.Address)
		res.To = to.String()
		value := params[1].(*big.Int)
		res.Value = value
		log.Info().Msgf("unpack success %v", res)

	}
	return res

}

// Cancel 取消
func (w *Worker) Cancel(privateKey, from, txHash string) (string, string, uint64, error) {
	//pend := w.GetPendingByHex(txHash)
	//if pend == nil {
	//	return "", "", 0, errors.New("target transaction not in pending")
	//}
	// 收款方为发送方 设置为取消
	//pend.To = pend.From

	//tx, isPend, err := w.http.TransactionByHash(context.Background(), common.HexToHash(txHash))
	//if err != nil {
	//	log.Error().Msgf("get transaction by hash error %s", err.Error())
	//	return "", "", 0, err
	//}
	//if !isPend {
	//	log.Info().Msgf("transaction %s not in pending", txHash)
	//	return "", "", 0, errors.New("target transaction not in pending")
	//}
	pend := &types.Transaction{
		//Data:      tx.Data(),
		//Gas:       tx.Gas(),
		//GasFeeCap: tx.GasFeeCap(),
		//GasTipCap: tx.GasTipCap(),
		To: from, // 收款方为发送方 设置为取消
		//Nonce:     tx.Nonce(),
		//Value:     tx.Value(),
	}

	transaction, s, u, err := w.sendTransaction("", privateKey, "", nil, nil, 0, nil, pend)
	if err != nil {
		return "", "", 0, err
	}
	return transaction, s, u, nil
}

// SpeedUp 加速
func (w *Worker) SpeedUp(privateKey, txHash string) (string, string, uint64, error) {
	pend := w.GetPendingByHex(txHash)
	if pend == nil {
		return "", "", 0, errors.New("target transaction not in pending")
	}
	// 收款方为发送方 设置为取消
	transaction, s, u, err := w.sendTransaction("", privateKey, "", nil, nil, 0, nil, pend)
	if err != nil {
		return "", "", 0, err
	}
	return transaction, s, u, nil
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

// MulSignMode 多签
func (w *Worker) MulSignMode(to, coinName, num, timeStamp string) (int32, string) {
	signCounter.Lock.Lock()
	defer signCounter.Lock.Unlock()

	return 0, ""
}
