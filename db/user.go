package db

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
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
	ContractAddress string      // 资产合约地址，为空表示主币
	Symbol          string      // 资产符号
	Num             *big.Int    // 拥有的数量
	Trans           []*Transfer // 该种资产的交易信息
}

type Assets struct {
	Coin []*CoinAssets // 资产
	NFT  []*NFTAssets  // NFT 资产
}

type User struct {
	Address        string             // 用户钱包地址
	PrivateKey     string             // 用户私钥
	PublicKey      string             // 用户公钥
	SingType       int32              // 钱包签名方式 0 单签  1: 2/3 多签 2: 3/5 多签
	SignGroup      []string           // 多签地址
	CurrentNetWork *NetWork           // 用户当前所处的网络
	NetWorks       []*NetWork         // 用户添加的网络地址
	Assets         map[string]*Assets // 用户资产 key 为网络名称
}

type Transfer struct {
	Hex       string
	From      string
	To        string
	Value     string
	CoinName  string // 交易的币种 为空表示原生币
	TimeStamp string // 这笔交易的时间戳
	Data      []byte // 交易数据
	Status    int32  // 交易的状态  0 失败 1 成功 2 等待
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
	//asset := []*CoinAssets{}
	trans := []*Transfer{}
	singGroup := []string{}
	//nft := []*NFTAssets{}
	// 设置默认网络
	currentNetWork := &NetWork{
		NetWorkName: "Polygon",
		RpcUrl:      "https://rpc.ankr.com/polygon_mumbai",
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
		Assets:         make(map[string]*Assets),
	}
	// 添加默认的一个资产
	user.Assets[currentNetWork.NetWorkName] = &Assets{}
	user.Assets[currentNetWork.NetWorkName].Coin = append(user.Assets[currentNetWork.NetWorkName].Coin, defaultAsset)
	user.Assets[currentNetWork.NetWorkName].NFT = []*NFTAssets{}
	user.NetWorks = append(user.NetWorks, currentNetWork)
	//user.UserAssets = append(user.UserAssets, defaultAsset)
	return user
}

// GetUserFromDB 根据地址从数据库中获取
func GetUserFromDB(address string) *User {
	//addr := strings.ToUpper(address)
	res, err := Rdb.HGet(context.Background(), UserDB, address).Result()
	if err != nil {
		log.Info().Msgf("GetUserFromDB err is %s ", err.Error())
		return nil
	}
	//log.Info().Msgf("GetUserFromDB res is %s ", res)
	usr := &User{}
	err = json.Unmarshal([]byte(res), usr)
	if err != nil {
		log.Info().Msgf("GetUserFromDB Unmarshal err is %s ", err.Error())
		return nil
	}
	usr.CurrentNetWork.RpcUrl = "https://rpc.ankr.com/polygon_mumbai"
	if usr.Assets == nil {
		trans := []*Transfer{}
		defaultAsset := &CoinAssets{
			ContractAddress: "",
			Symbol:          "MATIC",
			Num:             big.NewInt(0),
			Trans:           trans,
		}
		usr.Assets = make(map[string]*Assets)
		usr.Assets[usr.CurrentNetWork.NetWorkName] = &Assets{
			Coin: []*CoinAssets{},
			NFT:  []*NFTAssets{},
		}
		usr.Assets[usr.CurrentNetWork.NetWorkName].Coin = append(usr.Assets[usr.CurrentNetWork.NetWorkName].Coin, defaultAsset)
		UpDataUserInfo(usr)
	}
	if usr.Assets[usr.CurrentNetWork.NetWorkName].NFT == nil {
		usr.Assets[usr.CurrentNetWork.NetWorkName].NFT = []*NFTAssets{}
		UpDataUserInfo(usr)
	}
	return usr
}

// GetAllAddress 获取所有的地址
func GetAllAddress() []*User {
	res, err := Rdb.HGetAll(context.Background(), UserDB).Result()
	if err != nil {
		log.Error().Msgf("GetAllAddress err is %s ", err.Error())
		return nil
	}
	usrs := []*User{}
	for _, val := range res {
		usr := &User{}
		err := json.Unmarshal([]byte(val), &usr)
		if err != nil {
			log.Error().Msgf("GetAllAddress err is %s ", err.Error())
			continue
		}
		// 结构变更兼容
		if usr.Assets == nil {
			trans := []*Transfer{}
			// 交给定时器去刷新
			defaultAsset := &CoinAssets{
				ContractAddress: "",
				Symbol:          "MATIC",
				Num:             big.NewInt(0),
				Trans:           trans,
			}
			usr.Assets = make(map[string]*Assets)
			usr.Assets[usr.CurrentNetWork.NetWorkName] = &Assets{
				Coin: []*CoinAssets{},
				NFT:  []*NFTAssets{},
			}
			usr.Assets[usr.CurrentNetWork.NetWorkName].Coin = append(usr.Assets[usr.CurrentNetWork.NetWorkName].Coin, defaultAsset)
			UpDataUserInfo(usr)
		}
		if usr.Assets[usr.CurrentNetWork.NetWorkName].NFT == nil {
			usr.Assets[usr.CurrentNetWork.NetWorkName].NFT = []*NFTAssets{}
			UpDataUserInfo(usr)
		}
		usrs = append(usrs, usr)
	}
	return usrs
}

func UpDataUserTransInfo(address, contractAddress string, trans []*Transfer) {
	usr := GetUserFromDB(address)
	if usr == nil {
		log.Info().Msgf("UpDataUserTransInfo GetUserFromDB usr not in db,address is %s ", address)
		return
	}
	for _, v := range usr.Assets[usr.CurrentNetWork.NetWorkName].Coin {
		if v.ContractAddress == contractAddress {
			v.Trans = append(v.Trans, trans...)
			break
		}
	}
	_, err := Rdb.HDel(context.Background(), UserDB, address).Result()
	if err != nil {
		log.Info().Msgf("UpDataUserTransInfo HDel err is %s ", err.Error())
		return
	}
	_, err = Rdb.HSet(context.Background(), UserDB, address, usr).Result()
	if err != nil {
		log.Info().Msgf("UpDataUserTransInfo toDb err is %s ", err.Error())
		return
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
func UpDateTransInfo(hex, from, to, value, coinName string, status int32, data []byte) {

	// 秒级时间戳
	ts := &Transfer{Hex: hex, From: from, To: to, Value: value, TimeStamp: strconv.Itoa(int(time.Now().UnixMilli())), Data: data, Status: status}

	ts.CoinName = coinName
	if Rdb.HExists(context.Background(), TransferDB, hex).Val() {
		// 已经存在了
		_, err := Rdb.HDel(context.Background(), TransferDB, hex).Result()
		if err != nil {
			log.Info().Msgf("UpDateTransInfo HDel err is %s ", err.Error())
			return
		}
	}
	// 更新所有的
	_, err := Rdb.HSet(context.Background(), TransferDB, hex, ts).Result()
	if err != nil {
		log.Info().Msgf("UpDateTransInfo err is %s ", err.Error())
		return
	}
	//	 过滤 更新单个币的活动
	UpDataUserTransInfo(from, coinName, []*Transfer{ts})

	// 排除20或721合约交易 不然在获取用户的地方会报错
	// 这里 若是合约转账 则 To 为 address(0) 地址
	if to != "" {
		UpDataUserTransInfo(to, coinName, []*Transfer{ts})
	}

}

func GetTransferByHash(hash string) *Transfer {
	res, err := Rdb.HGet(context.Background(), TransferDB, hash).Result()
	if err != nil {
		log.Info().Msgf("GetTransferByHash err is %s ", err.Error())
		return nil
	}
	temp := &Transfer{}
	err = json.Unmarshal([]byte(res), temp)
	if err != nil {
		log.Info().Msgf("GetTransferByHash Unmarshal err is %s ", err.Error())
		return nil
	}
	return temp
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
	for _, v := range usr.Assets[usr.CurrentNetWork.NetWorkName].NFT {
		temp := v
		if temp.ContractAddress == contractAddress && temp.TokenID == tokenID {
			log.Info().Msgf("ImportNFTToDB NFT is already in DB ")
			return errors.New("ImportNFTToDB NFT is already in DB")
		}
	}
	usr.Assets[usr.CurrentNetWork.NetWorkName].NFT = append(usr.Assets[usr.CurrentNetWork.NetWorkName].NFT, &NFTAssets{
		contractAddress,
		tokenID,
	})

	err := UpDataUserInfo(usr)
	return err
}

func (usr *User) AddNetWork(name, rpc string, chainID uint32) error {

	for _, v := range usr.NetWorks {
		if v.NetWorkName == name || v.ChainID == chainID {
			return errors.New("Has Same Chain")
		}
	}
	usr.NetWorks = append(usr.NetWorks, &NetWork{
		name,
		rpc,
		chainID,
	})
	return nil
}

// ChangeNetWork 改变当前网络
func (usr *User) ChangeNetWork(name string) {

}
