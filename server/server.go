package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lmxdawn/wallet/config"
	"github.com/lmxdawn/wallet/db"
	"github.com/lmxdawn/wallet/engine"
	"github.com/rs/zerolog/log"
)

// Start 启动服务
func Start(isSwag bool, configPath string) {
	db.Init()
	conf, err := config.NewConfig(configPath)

	CoinInit(conf.Engines[0].Rpc)
	Init()
	if err != nil || len(conf.Engines) == 0 {
		panic("Failed to load configuration")
	}
	err = engine.NewWorker(5, conf.Engines[0].Rpc)
	if err != nil {
		log.Fatal().Msgf("NewWorker err is %s ", err.Error())
		return
	}

	if isSwag {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	err = engine.NewNFTWorker(conf.Engines[0].Rpc)
	if err != nil {
		log.Fatal().Msgf("NewNFTWorker err is %s ", err.Error())
	}
	server := gin.Default()
	// 中间件
	server.Use(gin.Logger())
	server.Use(gin.Recovery())
	//server.Use(SetEngine(engines...))
	//server.Use(start20Token())
	//server.Use(AuthRequired())
	//server.Use(sessions.Sessions("Session", store))
	auth := server.Group("/", AuthRequired())
	{
		auth.POST("/createWallet", CreateWallet)
		auth.POST("/delWallet", DelWallet)
		auth.GET("/getTransactionReceipt", GetTransactionReceipt)
		auth.POST("/transaction", Transaction)
		// 添加新币
		auth.POST("/addNewCoin", AddNewCoin)
		// 获取实时的 gas 费用 链上状态
		auth.POST("/getLinkStatus", GetLinkStatus)
		// 获取账户的余额信息
		auth.POST("/getBalance", GetBalance)
		// 添加网络
		auth.POST("/addNetWork", AddNetWork)
		// 获取钱包基础信息
		auth.GET("/getWalletInfo", GetWalletInfo)
		auth.GET("/getWalletList", GetWalletList)
		// 获取账户的活动信息
		auth.GET("/getActivity", GetActivity)
		auth.GET("/getHistoryTrans", GetHistoryTrans)
		auth.POST("/checkTrans", CheckTrans)
		auth.POST("/changSignType", ChangSignType)
		auth.POST("/exportWallet", ExportWallet)
		auth.POST("/nftTransfer", NFTTransfer)
		auth.POST("/addNft", AddNFT)
		auth.POST("/addLink", AddLink)
		auth.POST("/changeLink", ChangeLink)
		auth.POST("/importWallet", ImportWallet)

	}
	// 登录检测
	server.POST("/login", Login)
	server.POST("/register", Register)

	err = server.Run(fmt.Sprintf(":%v", conf.App.Port))
	if err != nil {
		panic("start error")
	}

	log.Info().Msgf("start success at %s ", conf.App.Port)

}
