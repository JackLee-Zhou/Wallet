package db

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/lmxdawn/wallet/types"
	"github.com/rs/zerolog/log"
	"math/big"
	"strconv"
	"time"
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

// NFTAssets NFT 资产
type NFTAssets struct {
	ContractAddress string // 合约地址
	TokenID         string // NFT ID
}

type CoinAssets struct {
	ContractAddress string               // 资产合约地址，为空表示主币
	Symbol          string               // 资产符号
	Num             *big.Int             // 拥有的数量
	Trans           []*types.Transaction // 该种资产的交易信息
}

type User struct {
	Address        string        // 用户钱包地址
	PrivateKey     string        // 用户私钥
	PublicKey      string        // 用户公钥
	SingType       int32         // 钱包签名方式 0 单签  1: 2/3 多签 2: 3/5 多签
	SignGroup      []string      // 多签地址
	CurrentNetWork *NetWork      // 用户当前所处的网络
	NetWorks       []*NetWork    // 用户添加的网络地址
	UserAssets     []*CoinAssets // 用户资产数据
	NFTAssets      []*NFTAssets  // 用户 NFT 资产数据
}

type Transfer struct {
	Hex       string
	From      string
	To        string
	Value     string
	CoinName  string // 交易的币种 为空表示原生币
	TimeStamp string // 这笔交易的时间戳
}

type Account struct {
	Account    string
	PassWD     string   // 密码
	WalletList []string // 对应的钱包地址列表
}

type LoginInfo struct {
	Account   string
	TimeStamp string
}

func (l LoginInfo) MarshalBinary() ([]byte, error) {
	return json.Marshal(l)
}

func (a Account) MarshalBinary() ([]byte, error) {

	return json.Marshal(a)
}

func (t Transfer) MarshalBinary() ([]byte, error) {

	return json.Marshal(t)
}

func (t Transfer) UnmarshalBinary(data []byte) error {

	return json.Unmarshal(data, &t)
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
	asset := []*CoinAssets{}
	trans := []*types.Transaction{}
	singGroup := []string{}
	nft := []*NFTAssets{}
	// 设置默认网络
	currentNetWork := &NetWork{
		NetWorkName: "Polygon",
		RpcUrl:      "https://endpoints.omniatech.io/v1/matic/mumbai/public",
		ChainID:     80001,
	}
	defaultAsset := &CoinAssets{
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
		NFTAssets:      nft,
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
	err = json.Unmarshal([]byte(res), usr)
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

// UpDataUserInfo 更新用户数据
func UpDataUserInfo(usr *User) error {
	ok, err := Rdb.HExists(context.Background(), UserDB, usr.Address).Result()
	if err != nil {
		log.Info().Msgf("UpDataUserInfo HExists err is %s ", err.Error())
		return err
	}
	if ok {
		_, err := Rdb.HDel(context.Background(), UserDB, usr.Address).Result()
		if err != nil {
			log.Info().Msgf("UpDataUserInfo HDel err is %s ", err.Error())
			return err
		}
	}

	_, err = Rdb.HSet(context.Background(), UserDB, usr.Address, usr).Result()
	if err != nil {
		log.Info().Msgf("UpDataUserInfo HSet err is %s ", err.Error())
		return err
	}
	return nil
}

// CheckWalletIsInDB 检查这个钱包地址是否在数据库中 多签使用
func CheckWalletIsInDB(address string) bool {
	ok, err := Rdb.HExists(context.Background(), UserDB, address).Result()
	if err != nil {
		log.Info().Msgf("CheckWalletIsInDB err is %s ", err.Error())
		return false
	}
	return ok
}

// MulSignMode 多签模式
func (u *User) MulSignMode(to, coinName, num string) bool {
	// 先查一下数据库中 这笔交易那些用户已经签署了
	return false
}

// UpDateTransInfo 更新交易数据
func UpDateTransInfo(hex, from, to, value string) {

	// 秒级时间戳
	ts := &Transfer{Hex: hex, From: from, To: to, Value: value, TimeStamp: strconv.Itoa(int(time.Now().UnixMilli()))}

	_, err := Rdb.HSet(context.Background(), TransferDB, hex, ts).Result()
	if err != nil {
		log.Info().Msgf("UpDateTransInfo err is %s ", err.Error())
		return
	}
}

// GetTransferFromDB 获取以 from 为目标地址的交易信息
func GetTransferFromDB(address string) (data []*Transfer) {
	type info struct {
		From      string
		To        string
		Value     string
		CoinName  string // 交易的币种 为空表示原生币
		TimeStamp string // 这笔交易
	}
	// TODO 不该这样获取所有
	res, err := Rdb.HGetAll(context.Background(), TransferDB).Result()
	if err != nil {
		log.Info().Msgf("GetTransferFromDB err is %s ", err.Error())
		return
	}
	for key, value := range res {
		tempData := value
		temp := &info{}
		err := json.Unmarshal([]byte(tempData), temp)

		if err != nil {
			log.Info().Msgf("GetTransferFromDB UnMarshal err is %s ", err.Error())
			continue
		}

		// 筛选出 From 和 to 符合条件的
		if temp.From != address && temp.To != address {
			continue
		}

		data = append(data, &Transfer{
			Hex:       key,
			From:      temp.From,
			To:        temp.To,
			Value:     temp.Value,
			CoinName:  temp.CoinName, // 交易的币种 为空表示原生币
			TimeStamp: temp.TimeStamp,
		})
	}
	return
}

func UpDataAccountInfo(account, passwd string) (*Account, error) {

	ac := &Account{
		Account:    account,
		PassWD:     passwd,
		WalletList: nil,
	}
	_, err := Rdb.HSet(context.Background(), AccountDB, account, ac).Result()
	if err != nil {
		log.Info().Msgf("GetAccountInfo get err is %s ", err.Error())
		return nil, err
	}

	return ac, nil
}

func GetAccountInfo(account string) *Account {
	res, err := Rdb.HGet(context.Background(), AccountDB, account).Result()
	if err != nil {
		log.Info().Msgf("GetAccountInfo get err is %s ", err.Error())
		return nil
	}
	ac := &Account{}
	err = json.Unmarshal([]byte(res), ac)
	if err != nil {
		log.Info().Msgf("GetAccountInfo marshal err is %s ", err.Error())
		return nil
	}
	return ac

}

func UpDataLoginInfo(account string) {
	lg := &LoginInfo{
		Account: account,
		// 毫秒级的时间戳
		TimeStamp: strconv.Itoa(int(time.Now().UnixMilli())),
	}

	// TODO 原子性
	_, err := Rdb.HSetNX(context.Background(), LoginDB, account, lg).Result()
	if err != nil {
		log.Info().Msgf("CheckLoginInfo to DB err is %s ", err.Error())
	}
	// 设置过期时间
	isOK, err := Rdb.Expire(context.Background(), LoginDB+":"+account, time.Hour).Result()
	if err != nil {
		log.Info().Msgf("CheckLoginInfo  DB Expire err is %s ", err.Error())
	}
	log.Info().Msgf("CheckLoginInfo  DB Expire isOk %v ", isOK)
}

// CheckLoginInfo 检查登录状态
func CheckLoginInfo(account string) bool {

	res, err := Rdb.HExists(context.Background(), LoginDB, account).Result()
	if err != nil {
		log.Info().Msgf("CheckLoginInfo from DB err is %s ", err.Error())
		return false
	}

	log.Info().Msgf("CheckLoginInfo res is %v ", res)
	if !res {
		return false
	}
	return true
}

// ImportNFTToDB 导入NFT数据到数据库
func (usr *User) ImportNFTToDB(contractAddress, tokenID string) error {
	for _, v := range usr.NFTAssets {
		temp := v
		if temp.ContractAddress == contractAddress && temp.TokenID == tokenID {
			log.Info().Msgf("ImportNFTToDB NFT is already in DB ")
			return errors.New("ImportNFTToDB NFT is already in DB")
		}
	}
	usr.NFTAssets = append(usr.NFTAssets, &NFTAssets{
		contractAddress,
		tokenID,
	})

	err := UpDataUserInfo(usr)
	return err
}
