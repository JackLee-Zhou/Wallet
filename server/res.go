package server

import (
	"github.com/gin-gonic/gin"
	"github.com/lmxdawn/wallet/types"
	"net/http"
)

// Response ...
type Response struct {
	Code    int         `json:"code"`    // 错误code码
	Message string      `json:"message"` // 错误信息
	Data    interface{} `json:"data"`    // 成功时返回的对象
}

// APIResponse ....
func APIResponse(Ctx *gin.Context, err error, data interface{}) {
	if err == nil {
		err = OK
	}
	codeNum, message := DecodeErr(err)
	Ctx.JSON(http.StatusOK, Response{
		Code:    codeNum,
		Message: message,
		Data:    data,
	})
}

// CreateWalletRes ...
type CreateWalletRes struct {
	Address string `json:"address"` // 生成的钱包地址
}

// WithdrawRes ...
type WithdrawRes struct {
	Hash string `json:"hash"` // 生成的交易hash
}

// CollectionRes ...
type CollectionRes struct {
	Balance string `json:"balance"` // 实际归集的数量
}

// TransactionReceiptRes ...
type TransactionReceiptRes struct {
	Status int `json:"status"` // 交易状态（0：未成功，1：已成功）
}

// LinkStatus 链上状态 gas gasPrice
type LinkStatus struct {
	GasPrice string `json:"gasPrice"` // gasPrice
}

type GetBalanceRes struct {
	Balance string `json:"balance"`
}

// SendTransactionRes 执行交易回执
type SendTransactionRes struct {
	FromHex string `json:"fromHex"`
	SignHax string `json:"signHax"`
	Nonce   uint64 `json:"nonce"`
}

type WalletActivityRes struct {
	UserAddress string `json:"userAddress"`
	History     []*types.Transaction
}

type History struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Result  []*HistoryRes `json:"result"`
}
type HistoryRes struct {
	BlockNumber       string `json:"blockNumber"`
	TimeStamp         string `json:"timeStamp"`
	Hash              string `json:"hash"`
	Nonce             string `json:"nonce"`
	BlockHash         string `json:"blockHash"`
	TransactionIndex  string `json:"transactionIndex"`
	From              string `json:"from"`
	To                string `json:"to"`
	Value             string `json:"value"`
	Gas               string `json:"gas"`
	GasPrice          string `json:"gasPrice"`
	IsError           string `json:"isError"`
	TxreceiptStatus   string `json:"txreceipt_status"`
	Input             string `json:"input"`
	ContractAddress   string `json:"contractAddress"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	GasUsed           string `json:"gasUsed"`
	Confirmations     string `json:"confirmations"`
}
