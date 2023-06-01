package server

import (
	"context"
	"encoding/json"
	"github.com/btcsuite/websocket"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/lmxdawn/wallet/db"
	"github.com/lmxdawn/wallet/engine"
	"github.com/rs/zerolog/log"
	"math/big"
	"net/http"
	"strconv"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

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
	err := engine.AddNewCoin(newCoin.CoinName, newCoin.ContractAddress)
	if err != nil {
		APIResponse(c, err, nil)
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
	// 查询历史记录
	res.History = worker.TransHistory[walletActivity.UserAddress]
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
	//ws, _ := upgrader.Upgrade(c.Writer, c.Request, nil)

	currentEngine := v.(*engine.ConCurrentEngine)
	num, err := strconv.Atoi(sT.Num)
	if err != nil {
		APIResponse(c, err, nil)
		return
	}
	// TODO 根据地址 数据库中查询获取到 privateKey
	getRes, err := db.Rdb.HGet(context.Background(), db.UserDB, sT.From).Result()
	if err != nil {
		APIResponse(c, err, nil)
	}
	usr := db.User{}
	json.Unmarshal([]byte(getRes), &usr)
	//err = usr.UnmarshalBinary([]byte(getRes))
	if err != nil {
		APIResponse(c, err, nil)
	}
	// TODO 检查是否是多签 若是则走多签的流程
	if usr.SingType != db.SingerSign {
		usr.MulSignMode(sT.To, sT.CoinName, sT.Num)
	}

	// 后端签名
	// 这里 返回的仅是放到了交易池里面等到被执行，并没有实际的被真正的执行 还是处于 pending 状态
	fromHex, signHex, nonce, err := currentEngine.Worker.Transfer(usr.PrivateKey, sT.To, big.NewInt(int64(num)), 0)
	if err != nil {
		APIResponse(c, err, nil)
		return
	}
	// 这里操作数据库 落地存储
	worker := currentEngine.Worker.(*engine.EthWorker)
	trans := worker.TransHistory[usr.Address]
	for _, value := range usr.UserAssets {
		temp := value
		if sT.CoinName == temp.Symbol {
			temp.Trans = trans
			break
		}
	}

	_, err = db.Rdb.HSet(context.Background(), db.UserDB, usr.Address, usr).Result()
	if err != nil {
		log.Info().Msgf("Trans to DB err is %s ", err.Error())
	}
	res.FromHex = fromHex
	res.SignHax = signHex
	res.Nonce = nonce
	APIResponse(c, nil, res)

	// TODO 设置主动推送的限制时间
	//for {
	//	select {
	//	case finish := <-currentEngine.TransNotify:
	//		err :=ws.WriteJSON(finish)
	//		if
	//	}
	//}
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

// CheckTrans 检查交易是否成功
func CheckTrans(c *gin.Context) {
	var cT CheckTransReq
	if err := c.ShouldBindJSON(&cT); err != nil {
		HandleValidatorError(c, err)
		return
	}
	v, ok := c.Get(cT.Protocol + cT.CoinName)
	if !ok {
		HandleValidatorError(c, ErrNotData)
		return
	}
	currentEngine := v.(*engine.ConCurrentEngine)
	if _, ok := currentEngine.TransNotify[cT.TxHash]; !ok {
		APIResponse(c, ErrNoSuccess, struct {
			isOk bool
		}{
			isOk: false,
		})
	}
	APIResponse(c, nil, struct {
		isOk bool
	}{
		isOk: true,
	})
}

// GetWalletInfo 获取钱包基础信息
func GetWalletInfo(c *gin.Context) {
	address, ok := c.GetQuery("Address")
	if !ok {
		APIResponse(c, ErrNoAddress, nil)
	}

	usr := db.GetUserFromDB(address)
	log.Info().Msgf("GetWalletInfo info is %v ", usr)
	APIResponse(c, nil, usr)
}

// ImportWallet 从外部导入钱包
func ImportWallet(c *gin.Context) {

}

// ExportWallet 导出钱包
func ExportWallet(c *gin.Context) {

}

// ChangSignType 改变签名方式
func ChangSignType(c *gin.Context) {
	var csT ChangSignTypeReq
	if err := c.ShouldBindJSON(&csT); err != nil {
		HandleValidatorError(c, err)
		return
	}
	usr := db.GetUserFromDB(csT.WalletAddress)

	// 不是单签
	if csT.SignType != db.SingerSign {
		switch csT.SignType {
		case db.ThreeTwoSign:
			if len(csT.SingGroup) != 3 {
				APIResponse(c, ErrSignGroupLengthErr, nil)
			}
			for _, v := range csT.SingGroup {
				temp := v
				if !db.CheckWalletIsInDB(temp) {
					APIResponse(c, ErrWalletNotInDB, nil)
				}
			}
			usr.SingType = db.ThreeTwoSign
			usr.SignGroup = csT.SingGroup
		case db.FiveFourSign:
			if len(csT.SingGroup) != 5 {
				APIResponse(c, ErrSignGroupLengthErr, nil)
			}
			for _, v := range csT.SingGroup {
				temp := v
				if !db.CheckWalletIsInDB(temp) {
					APIResponse(c, ErrWalletNotInDB, nil)
				}
			}
			usr.SingType = db.FiveFourSign
			usr.SignGroup = csT.SingGroup
		}
	}
	_, err := db.Rdb.HSet(context.Background(), db.UserDB, csT.WalletAddress, usr).Result()
	if err != nil {
		log.Info().Msgf("ChangSignType UpDate UserInfo Fail err is %s", err.Error())
		APIResponse(c, err, nil)
	}
	APIResponse(c, nil, &struct {
		Message string
	}{
		Message: "调整成功",
	})
}

// Sign 其他用户签名
func Sign(c *gin.Context) {

}
