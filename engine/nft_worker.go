package engine

import (
	"context"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"math/big"
	"strings"
	"sync"
)

var NFT *NFTWorker

type NFTWorker struct {
	http                   *ethclient.Client
	tokenTransferEventHash common.Hash
	tokenAbi               abi.ABI             // 合约的abi
	Pending                map[string]struct{} // 待执行的交易
	nonceLock              sync.Mutex
}

// NewNFTWorker 新建 NFT 交易者
func NewNFTWorker(url string) error {
	http, err := ethclient.Dial(url)
	if err != nil {
		return err
	}
	var tokenTransferEventHashSig []byte
	var tokenTransferEventHash common.Hash
	var tokenAbiStr string

	tokenTransferEventHashSig = []byte("transferFrom(address,address,uint256)")
	tokenTransferEventHash = crypto.Keccak256Hash(tokenTransferEventHashSig)
	tokenAbiStr = "[\n\t{\n\t\t\"inputs\": [],\n\t\t\"stateMutability\": \"nonpayable\",\n\t\t\"type\": \"constructor\"\n\t},\n\t{\n\t\t\"anonymous\": false,\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"indexed\": true,\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"owner\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"indexed\": true,\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"approved\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"indexed\": true,\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"tokenId\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"Approval\",\n\t\t\"type\": \"event\"\n\t},\n\t{\n\t\t\"anonymous\": false,\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"indexed\": true,\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"owner\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"indexed\": true,\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"operator\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"indexed\": false,\n\t\t\t\t\"internalType\": \"bool\",\n\t\t\t\t\"name\": \"approved\",\n\t\t\t\t\"type\": \"bool\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"ApprovalForAll\",\n\t\t\"type\": \"event\"\n\t},\n\t{\n\t\t\"anonymous\": false,\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"indexed\": true,\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"from\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"indexed\": true,\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"to\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"indexed\": true,\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"tokenId\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"Transfer\",\n\t\t\"type\": \"event\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"to\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"tokenId\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"approve\",\n\t\t\"outputs\": [],\n\t\t\"stateMutability\": \"nonpayable\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"owner\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"balanceOf\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"view\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"tokenId\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"getApproved\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"view\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"owner\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"operator\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"isApprovedForAll\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"bool\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"bool\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"view\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"player\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"mintNew\",\n\t\t\"outputs\": [],\n\t\t\"stateMutability\": \"nonpayable\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [],\n\t\t\"name\": \"name\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"string\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"string\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"view\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [],\n\t\t\"name\": \"owner\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"view\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"tokenId\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"ownerOf\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"view\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"from\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"to\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"tokenId\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"safeTransferFrom\",\n\t\t\"outputs\": [],\n\t\t\"stateMutability\": \"nonpayable\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"from\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"to\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"tokenId\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"bytes\",\n\t\t\t\t\"name\": \"data\",\n\t\t\t\t\"type\": \"bytes\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"safeTransferFrom\",\n\t\t\"outputs\": [],\n\t\t\"stateMutability\": \"nonpayable\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"operator\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"bool\",\n\t\t\t\t\"name\": \"approved\",\n\t\t\t\t\"type\": \"bool\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"setApprovalForAll\",\n\t\t\"outputs\": [],\n\t\t\"stateMutability\": \"nonpayable\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"bytes4\",\n\t\t\t\t\"name\": \"interfaceId\",\n\t\t\t\t\"type\": \"bytes4\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"supportsInterface\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"bool\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"bool\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"view\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [],\n\t\t\"name\": \"symbol\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"string\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"string\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"view\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"tokenId\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"tokenURI\",\n\t\t\"outputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"string\",\n\t\t\t\t\"name\": \"\",\n\t\t\t\t\"type\": \"string\"\n\t\t\t}\n\t\t],\n\t\t\"stateMutability\": \"view\",\n\t\t\"type\": \"function\"\n\t},\n\t{\n\t\t\"inputs\": [\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"from\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"address\",\n\t\t\t\t\"name\": \"to\",\n\t\t\t\t\"type\": \"address\"\n\t\t\t},\n\t\t\t{\n\t\t\t\t\"internalType\": \"uint256\",\n\t\t\t\t\"name\": \"tokenId\",\n\t\t\t\t\"type\": \"uint256\"\n\t\t\t}\n\t\t],\n\t\t\"name\": \"transferFrom\",\n\t\t\"outputs\": [],\n\t\t\"stateMutability\": \"nonpayable\",\n\t\t\"type\": \"function\"\n\t}\n]"

	tokenAbi, err := abi.JSON(strings.NewReader(tokenAbiStr))
	if err != nil {
		return err
	}
	NFT = &NFTWorker{
		http:                   http,
		tokenTransferEventHash: tokenTransferEventHash,
		tokenAbi:               tokenAbi,
		Pending:                make(map[string]struct{}), // 大小
	}
	return nil
}

// NFTTransfer NFT 转账 在这里直接走NFT的转账交易就行了 特殊处理 不和币种一样 循环监听
func NFTTransfer(contractAddress, from, privateKey string, to, tokenID string) (string, string, uint64, error) {

	contractTransferHashSig := []byte("transferFrom(address,address,uint256)")
	contractTransferHash := crypto.Keccak256Hash(contractTransferHashSig)
	toAddressTmp := common.HexToAddress(to)
	formAddressTmp := common.HexToAddress(from)
	toAddressHex := &toAddressTmp
	formAddressHex := &formAddressTmp
	data, err := makeEthERC721TransferData(contractTransferHash, formAddressHex, toAddressHex, tokenID)
	if err != nil {
		log.Error().Msgf("makeEthERC721TransferData err is %s ", err.Error())
		return "", "", 0, err
	}
	return send721Transaction(contractAddress, privateKey, data)
}

func send721Transaction(contractAddress string, privateKeyStr string, data []byte) (string, string, uint64, error) {
	var nonce uint64
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
	NFT.nonceLock.Lock()
	defer NFT.nonceLock.Unlock()
	nonce, err = NFT.http.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return "", "", 0, err
	}
	var toAddressHex *common.Address
	toAddressTmp := common.HexToAddress(contractAddress)
	toAddressHex = &toAddressTmp
	var gasLimit uint64
	//gasLimit = uint64(21000) // 在非合约中的转账 21000 是够的 但是在合约中 这个限制太小
	//gasLimit = uint64(200000) //
	// 预估 gasLimit
	gasLimit, err = NFT.http.EstimateGas(context.Background(), ethereum.CallMsg{
		//From: fromAddress,
		To:   toAddressHex,
		Data: data,
	})
	if err != nil {
		log.Error().Msgf("EstimateGas err is %s ", err.Error())
	}

	log.Info().Msgf("gasLimit is %d ", gasLimit)
	gasPrice, err := NFT.http.SuggestGasPrice(context.Background())
	if err != nil {
		return "", "", 0, err
	}
	gasTip, err := NFT.http.SuggestGasTipCap(context.Background())
	if err != nil {
		return "", "", 0, err
	}
	number, err := NFT.http.BlockNumber(context.Background())
	if err != nil {
		return "", "", 0, err
	}

	// 获取 baseFee
	block, err := NFT.http.BlockByNumber(context.Background(), big.NewInt(int64(number)))
	if err != nil {
		log.Info().Msgf("BlockByNumber err is %s ", err.Error())
		return "", "", 0, err
	}
	config := params.MainnetChainConfig
	baseFee := misc.CalcBaseFee(config, block.Header())

	// NFT 转账 polygon 没有计算 baseFee
	txData := &ethTypes.DynamicFeeTx{
		Nonce: nonce,
		To:    toAddressHex,
		// gas 单位上限
		Gas:       gasLimit * 2,
		GasFeeCap: gasPrice.Add(gasPrice, baseFee),
		GasTipCap: gasTip,
		Data:      data,
	}
	// 使用type 0 的方式能够完成交易
	//txData := &ethTypes.LegacyTx{
	//	Nonce: nonce,
	//	To:    toAddressHex,
	//	// gas 单位上限
	//	Gas: gasLimit * 5,
	//	// 设置的最高交易费用
	//	GasPrice: gasPrice,
	//	GasTipCap: gasTip,
	//	Data: data,
	//	// (gasFeeCap+GasTipCap)*Gas = Transaction Fee
	//}
	//tips := txData.GasFeeCap.Add(txData.GasFeeCap, txData.GasTipCap)
	//log.Info().Msgf("tips is %s ", tips.String())
	//log.Info().Msgf("gasFee is %s ", tips.Mul(tips, big.NewInt(int64(txData.Gas))).String())
	//log.Info().Msgf("txData is %+v ", txData)
	tx := ethTypes.NewTx(txData)

	chainID, err := NFT.http.NetworkID(context.Background())
	if err != nil {
		return "", "", 0, err
	}

	// 签名
	signTx, err := ethTypes.SignTx(tx, ethTypes.LatestSignerForChainID(chainID), privateKey)
	if err != nil {
		return "", "", 0, err
	}
	err = NFT.http.SendTransaction(context.Background(), signTx)
	if err != nil {
		return "", "", 0, err
	}

	return fromAddress.Hex(), signTx.Hash().Hex(), nonce, nil
}

// CheckIsOwner 检查是否是 NFT 的拥有者
func CheckIsOwner(contract, usr string, tokenID int) bool {
	address, err := NFT.callContract(contract, "ownerOf", big.NewInt(int64(tokenID)))
	if err != nil {
		log.Error().Msgf("callContract err is %s ", err.Error())
		return false
	}

	log.Info().Msgf("address is %s ", common.BytesToAddress(address).Hex())
	if common.BytesToAddress(address).Hex() != usr {
		return false
	}
	return true
}

func (nw *NFTWorker) callContract(contract string, method string, args ...interface{}) ([]byte, error) {
	input, _ := nw.tokenAbi.Pack(method, args...)
	to := common.HexToAddress(contract)
	msg := ethereum.CallMsg{
		To:   &to,
		Data: input,
	}
	hex, err := nw.http.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Error().Msgf("CallContract err is %s ", err.Error())
		return nil, err
	}
	return hex, nil
}

// makeEthERC721TransferData 构建 nft 交易数据
func makeEthERC721TransferData(contractTransferHash common.Hash, fromAddress, toAddress *common.Address, tokenID string) ([]byte, error) {
	var data []byte
	// 取函数 hash 值的高 4 位字节
	data = append(data, contractTransferHash[:4]...)
	// 依次拼接交易数据
	from := common.LeftPadBytes(fromAddress.Bytes(), 32)
	to := common.LeftPadBytes(toAddress.Bytes(), 32)
	id := new(big.Int)
	id.SetString(tokenID, 10)
	token := common.LeftPadBytes(id.Bytes(), 32)

	data = append(data, from...)
	data = append(data, to...)
	data = append(data, token...)
	log.Info().Msgf("data is %s ", hexutil.Encode(data))
	return data, nil
}
