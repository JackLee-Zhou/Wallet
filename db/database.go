package db

import (
	"context"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"io"
)

type WalletItem struct {
	Address    string // 地址
	PrivateKey string // 私钥
}

// CoinType 币种的结构
type CoinType struct {
	//Protocol        string `json:"protocol"`
	ContractAddress string `json:"contractAddress"`
	CoinName        string `json:"coinName"`
	IsNFT           bool   `json:"isNFT"`
}

func (c CoinType) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

type Reader interface {
	// Has retrieves if a key is present in the key-value data store.
	Has(key string) (bool, error)

	// Get retrieves the given key if it's present in the key-value data store.
	Get(key string) (string, error)

	// ListWallet retrieves the given key if it's present in the key-value data store.
	ListWallet(prefix string) ([]WalletItem, error)
}

type Writer interface {
	// Put inserts the given value into the key-value data store.
	Put(key string, value string) error

	// Delete removes the key from the key-value data store.
	Delete(key string) error
}

type Database interface {
	Reader
	Writer
	io.Closer
}

// UpDataCoinInfoToDB 想数据库中存入数据 判断这个是否存在
func UpDataCoinInfoToDB(coinName, contractAddress string) bool {
	exit, err := Rdb.HExists(context.Background(), CoinDB, contractAddress).Result()
	if err != nil {
		log.Error().Msgf("UpDataCoinInfoToDB err is %s ", err.Error())
		return false
	}

	if exit {
		log.Info().Msgf("UpDataCoinInfoToDB has same ContractAddress is %s ", contractAddress)
		return false
	}
	ct := &CoinType{
		ContractAddress: contractAddress,
		CoinName:        coinName,
	}
	_, err = Rdb.HSet(context.Background(), CoinDB, contractAddress, ct).Result()
	if err != nil {
		log.Error().Msgf("UpDataCoinInfoToDB Set err is %s ", err.Error())
		return false
	}
	return true
}

// GetAll20TokenFromDB 从数据空读取所有需要监听的 20 Token
func GetAll20TokenFromDB() (data []*CoinType) {
	res, err := Rdb.HGetAll(context.Background(), CoinDB).Result()
	if err != nil {
		log.Error().Msgf("GetAll20TokenFromDB err is %s ", err.Error())
		return nil
	}
	log.Info().Msgf("GetAll20TokenFromDB info %v", res)
	for _, v := range res {
		temp := v
		ct := &CoinType{}
		err := json.Unmarshal([]byte(temp), ct)
		if err != nil {
			log.Error().Msgf("GetAll20TokenFromDB Unmarshal err is %s ", err.Error())
			continue
		}
		data = append(data, ct)
	}
	return
}
