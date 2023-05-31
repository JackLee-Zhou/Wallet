package server

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/lmxdawn/wallet/engine"
	"math/big"
	"strconv"
)

// CreateWallet ...
// @Tags 钱包
// @Summary 创建钱包地址
// @Produce json
// @Security ApiKeyAuth
// @Param login body CreateWalletReq true "参数"
// @Success 200 {object} Response{data=server.CreateWalletRes}
// @Router /api/createWallet [post]
func CreateWallet(c *gin.Context) {

	var q CreateWalletReq

	if err := c.ShouldBindJSON(&q); err != nil {
		HandleValidatorError(c, err)
		return
	}

	v, ok := c.Get(q.Protocol + q.CoinName)
	if !ok {
		APIResponse(c, ErrEngine, nil)
		return
	}

	currentEngine := v.(*engine.ConCurrentEngine)

	// 创建钱包
	address, err := currentEngine.CreateWallet()
	if err != nil {
		APIResponse(c, ErrCreateWallet, nil)
		return
	}

	res := CreateWalletRes{Address: address}

	APIResponse(c, nil, res)
}

// DelWallet ...
// @Tags 钱包
// @Summary 删除钱包地址
// @Produce json
// @Security ApiKeyAuth
// @Param login body DelWalletReq true "参数"
// @Success 200 {object} Response
// @Router /api/delWallet [post]
func DelWallet(c *gin.Context) {

	var q DelWalletReq

	if err := c.ShouldBindJSON(&q); err != nil {
		HandleValidatorError(c, err)
		return
	}

	v, ok := c.Get(q.Protocol + q.CoinName)
	if !ok {
		APIResponse(c, ErrEngine, nil)
		return
	}

	currentEngine := v.(*engine.ConCurrentEngine)

	q.Address = common.HexToAddress(q.Address).Hex()

	err := currentEngine.DeleteWallet(q.Address)
	if err != nil {
		APIResponse(c, ErrCreateWallet, nil)
		return
	}

	APIResponse(c, nil, nil)
}

// Withdraw ...
// @Tags 钱包
// @Summary 提现
// @Produce json
// @Security ApiKeyAuth
// @Param login body WithdrawReq true "参数"
// @Success 200 {object} Response{data=server.WithdrawRes}
// @Router /api/withdraw [post]
func Withdraw(c *gin.Context) {

	var q WithdrawReq

	if err := c.ShouldBindJSON(&q); err != nil {
		HandleValidatorError(c, err)
		return
	}

	v, ok := c.Get(q.Protocol + q.CoinName)
	if !ok {
		APIResponse(c, ErrEngine, nil)
		return
	}

	currentEngine := v.(*engine.ConCurrentEngine)

	q.Address = common.HexToAddress(q.Address).Hex()

	hash, err := currentEngine.Withdraw(q.OrderId, q.Address, q.Value)
	if err != nil {
		APIResponse(c, err, nil)
		return
	}

	res := WithdrawRes{Hash: hash}

	APIResponse(c, nil, res)
}

// Collection ...
// @Tags 归集某个地址
// @Summary 归集
// @Produce json
// @Security ApiKeyAuth
// @Param login body CollectionReq true "参数"
// @Success 200 {object} Response{data=server.CollectionRes}
// @Router /api/collection [post]
func Collection(c *gin.Context) {

	var q CollectionReq

	if err := c.ShouldBindJSON(&q); err != nil {
		HandleValidatorError(c, err)
		return
	}

	v, ok := c.Get(q.Protocol + q.CoinName)
	if !ok {
		APIResponse(c, ErrEngine, nil)
		return
	}

	currentEngine := v.(*engine.ConCurrentEngine)

	n := new(big.Int)
	max, ok := n.SetString(q.Max, 10)
	if !ok {
		APIResponse(c, InternalServerError, nil)
		return
	}

	q.Address = common.HexToAddress(q.Address).Hex()

	balance, err := currentEngine.Collection(q.Address, max)
	if err != nil {
		APIResponse(c, err, nil)
		return
	}

	res := CollectionRes{Balance: balance.String()}

	APIResponse(c, nil, res)
}

// GetTransactionReceipt ...
// @Tags 钱包
// @Summary 获取交易结果
// @Produce json
// @Security ApiKeyAuth
// @Param login body TransactionReceiptReq true "参数"
// @Success 200 {object} Response{data=server.TransactionReceiptRes}
// @Router /api/getTransactionReceipt [get]
func GetTransactionReceipt(c *gin.Context) {

	var q TransactionReceiptReq

	if err := c.ShouldBindJSON(&q); err != nil {
		HandleValidatorError(c, err)
		return
	}

	v, ok := c.Get(q.Protocol + q.CoinName)
	if !ok {
		APIResponse(c, ErrEngine, nil)
		return
	}

	currentEngine := v.(*engine.ConCurrentEngine)

	status, err := currentEngine.GetTransactionReceipt(q.Hash)
	if err != nil {
		APIResponse(c, InternalServerError, nil)
		return
	}

	res := TransactionReceiptRes{
		Status: status,
	}

	APIResponse(c, nil, res)
}

// AddNewCoin 添加新币
func AddNewCoin(c *gin.Context) {
	var newCoin AddNewCoinReq
	if err := c.ShouldBindJSON(&newCoin); err != nil {
		HandleValidatorError(c, err)
		return
	}
}

// GetActivity 获取钱包活动信息 交易记录
func GetActivity(c *gin.Context) {
	var walletActivity GetWalletActivity
	res := WalletActivityRes{}
	if err := c.ShouldBindJSON(&walletActivity); err != nil {
		APIResponse(c, err, nil)
		return
	}

	v, ok := c.Get(walletActivity.Protocol + walletActivity.CoinName)
	if !ok {
		HandleValidatorError(c, ErrNotData)
		return
	}
	currentEngine := v.(*engine.ConCurrentEngine)
	worker := currentEngine.Worker.(*engine.EthWorker)
	for _, v := range worker.TransHistory {
		temp := v
		res.History = append(res.History, temp)
	}
	res.UserAddress = walletActivity.UserAddress
	APIResponse(c, nil, res)
	return
}

// Transaction 发起一笔交易
func Transaction(c *gin.Context) {
	var sT SendTransaction
	var res SendTransactionRes
	if err := c.ShouldBindJSON(&sT); err != nil {
		HandleValidatorError(c, err)
		return
	}
	v, ok := c.Get(sT.Protocol + sT.CoinName)
	if !ok {
		HandleValidatorError(c, ErrNotData)
		return
	}
	currentEngine := v.(*engine.ConCurrentEngine)
	num, err := strconv.Atoi(sT.Num)
	if err != nil {
		APIResponse(c, err, nil)
		return
	}
	// TODO 根据地址 数据库中查询获取到 privateKey
	get, err := currentEngine.DB.Get(currentEngine.Config.WalletPrefix + sT.From)
	if err != nil {
		APIResponse(c, err, nil)
		return
	}
	// 后端签名
	// 这里 返回的仅是放到了交易池里面等到被执行，并没有实际的被真正的执行 还是处于 pending 状态
	fromHex, signHex, nonce, err := currentEngine.Worker.Transfer(get, sT.To, big.NewInt(int64(num)), 0)
	if err != nil {
		APIResponse(c, err, nil)
		return
	}
	res.FromHex = fromHex
	res.SignHax = signHex
	res.Nonce = nonce
	APIResponse(c, nil, res)
}

// GetLinkStatus 获取实时的链上状态
func GetLinkStatus(c *gin.Context) {
	var res LinkStatus
	var linkStatus GetLinkStatusReq
	if err := c.ShouldBindJSON(&linkStatus); err != nil {
		HandleValidatorError(c, err)
		return
	}
	v, ok := c.Get(linkStatus.Rpc)
	if !ok {
		HandleValidatorError(c, ErrNotData)
		return
	}
	currentEngine := v.(*engine.ConCurrentEngine)
	price, err := currentEngine.Worker.GetGasPrice()
	if err != nil {
		HandleValidatorError(c, err)
		return
	}
	res.GasPrice = price
	APIResponse(c, nil, res)
}

// GetBalance 获取账户的余额信息
func GetBalance(c *gin.Context) {
	var balanceReq GetBalanceReq
	if err := c.ShouldBindJSON(&balanceReq); err != nil {
		HandleValidatorError(c, err)
		return
	}
	v, ok := c.Get(balanceReq.Protocol + balanceReq.CoinName)
	if !ok {
		APIResponse(c, ErrEngine, nil)
		return
	}
	currentEngine := v.(*engine.ConCurrentEngine)
	// 代币是 20 币 直接使用20 协议中的 balanceOf
	balance, err := currentEngine.Worker.GetBalance(balanceReq.UserAddress)
	if err != nil {
		APIResponse(c, err, nil)
		return
	}
	res := GetBalanceRes{Balance: balance.String()}
	APIResponse(c, nil, res)
}

// AddNetWork 添加网络
func AddNetWork(c *gin.Context) {

}

// GetWalletInfo 获取钱包基础信息
func GetWalletInfo(c *gin.Context) {

}
