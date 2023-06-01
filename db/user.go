package db

import (
	"context"
	"encoding/json"
	"github.com/lmxdawn/wallet/types"
	"github.com/rs/zerolog/log"
	"math/big"
)

var SignTyp int32

const (
	SingerSign int32 = iota
	ThreeTwoSign
	FiveFourSign
)

type NetWork struct {
	NetWorkName string
	RpcUrl      string
	ChainID     uint32
}

type Assets struct {
	ContractAddress string               // 资产合约地址，为空表示主币
	Symbol          string               // 资产符号
	Num             *big.Int             // 拥有的数量
	Trans           []*types.Transaction // 该种资产的交易信息
}

type User struct {
	Address        string     // 用户钱包地址
	PrivateKey     string     // 用户私钥
	PublicKey      string     // 用户公钥
	SingType       int32      // 钱包签名方式 0 单签  1: 2/3 多签 2: 3/5 多签
	SignGroup      []string   // 多签地址
	CurrentNetWork *NetWork   // 用户当前所处的网络
	NetWorks       []*NetWork // 用户添加的网络地址
	UserAssets     []*Assets  // 用户资产数据
}

func (c User) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

func (c User) UnmarshalBinary(data []byte) error {

	return json.Unmarshal(data, &c)
}

// NewWalletUser 新建一个钱包用戶
func NewWalletUser(address, privateKey, publicKey string) *User {
	net := []*NetWork{}
	asset := []*Assets{}
	trans := []*types.Transaction{}
	singGroup := []string{}
	// 设置默认网络
	currentNetWork := &NetWork{
		NetWorkName: "Polygon",
		RpcUrl:      "https://endpoints.omniatech.io/v1/matic/mumbai/public",
		ChainID:     80001,
	}
	defaultAsset := &Assets{
		ContractAddress: "",
		Symbol:          "MATIC",
		Num:             big.NewInt(0),
		Trans:           trans,
	}
	user := &User{
		Address:        address,
		PrivateKey:     privateKey,
		PublicKey:      publicKey,
		SingType:       0, // 默认单签
		CurrentNetWork: currentNetWork,
		NetWorks:       net,
		SignGroup:      singGroup,
		UserAssets:     asset,
	}
	user.NetWorks = append(user.NetWorks, currentNetWork)
	user.UserAssets = append(user.UserAssets, defaultAsset)
	return user
}

// GetUserFromDB 根据地址从数据库中获取
func GetUserFromDB(address string) *User {
	res, err := Rdb.HGet(context.Background(), UserDB, address).Result()
	if err != nil {
		log.Info().Msgf("GetUserFromDB err is %s ", err.Error())
	}
	log.Info().Msgf("GetUserFromDB res is %s ", res)
	usr := &User{}
	err = usr.UnmarshalBinary([]byte(res))
	if err != nil {
		log.Info().Msgf("GetUserFromDB err is %s ", err.Error())
	}
	return usr
}

func UpDataUserTransInfo(address, coinName string, trans []*types.Transaction) {
	usr := GetUserFromDB(address)
	for _, v := range usr.UserAssets {
		if v.Symbol == coinName {
			v.Trans = trans
			break
		}
	}
	_, err := Rdb.HSet(context.Background(), UserDB, address, usr).Result()
	if err != nil {
		log.Info().Msgf("UpDataUserTransInfo toDb err is %s ", err.Error())
	}
}

// CheckWalletIsInDB 检查这个钱包地址是否在数据库中
func CheckWalletIsInDB(address string) bool {
	return true
}

// MulSignMode 多签模式
func (u *User) MulSignMode(to, coinName, num string) bool {
	// 先查一下数据库中 这笔交易那些用户已经签署了
	return false
}
