package db

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var Rdb *redis.Client

const (
	ConfigDB = "Config"
	UserDB   = "User"
)

// init 数据库链接初始化
func init() {
	//clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	//// 连接到MongoDB
	//client, err := mongo.Connect(context.TODO(), clientOptions)
	//MongoClient = client
	//if err != nil {
	//	log.Fatal().Msgf("MongoConnect err is %s ", err.Error())
	//}
	//WalletDB = MongoClient.Database("MyWallet").Collection("wallet")
	//// 检查连接
	//err = MongoClient.Ping(context.TODO(), nil)
	//if err != nil {
	//	log.Fatal().Msgf("MongoConnect err is %s ", err.Error())
	//}
	//fmt.Println("Connected to MongoDB!")
	Rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	res, err := Rdb.Ping(context.Background()).Result()
	log.Info().Msgf("Connection res is %v ", res)
	if err != nil {
		log.Fatal().Msgf("Connection err is %s ", err.Error())
		return
	}
}
