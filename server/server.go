package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lmxdawn/wallet/config"
	"github.com/lmxdawn/wallet/engine"
	"github.com/rs/zerolog/log"
)

// Start 启动服务
func Start(isSwag bool, configPath string) {
	conf, err := config.NewConfig(configPath)
	CoinInit()
	Init()
	if err != nil || len(conf.Engines) == 0 {
		panic("Failed to load configuration")
	}

	var engines []*engine.ConCurrentEngine

	// 读取配置 会启动很多监听 可同时监听 20 和 原生 配置的监听
	for _, engineConfig := range conf.Engines {
		eth, err := engine.NewEngine(engineConfig, false)
		if err != nil {
			panic(fmt.Sprintf("eth run err：%v", err))
		}
		engines = append(engines, eth)
	}

	// 启动监听器
	//for _, currentEngine := range engines {
	//	go currentEngine.Run()
	//}

	if isSwag {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	err = engine.NewNFTWorker("https://polygon-mumbai-bor.publicnode.com")
	if err != nil {
		log.Fatal().Msgf("NewNFTWorker err is %s ", err.Error())
	}
	server := gin.Default()
	//TODO 这里开始启动除开配置中的 20 Token 监听
	//start20TokenListen(server.)
	// 中间件
	server.Use(gin.Logger())
	server.Use(gin.Recovery())
	server.Use(SetEngine(engines...))
	//server.Use(start20Token())
	//server.Use(AuthRequired())
	//server.Use(sessions.Sessions("Session", store))
	auth := server.Group("/", AuthRequired())
	{
		auth.POST("/createWallet", CreateWallet)
		auth.POST("/delWallet", DelWallet)
		auth.GET("/getTransactionReceipt", GetTransactionReceipt)
		// TODO 发起一笔交易
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

	}
	// 登录检测
	server.POST("/login", Login)
	server.POST("/register", Register)

	//if isSwag {
	//swagHandler := ginSwagger.WrapHandler(swaggerFiles.Handler)
	//server.GET("/swagger/*any", swagHandler)
	//}

	err = server.Run(fmt.Sprintf(":%v", conf.App.Port))
	if err != nil {
		panic("start error")
	}

	log.Info().Msgf("start success at %s ", conf.App.Port)

}
