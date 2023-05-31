package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lmxdawn/wallet/config"
	"github.com/lmxdawn/wallet/engine"
	"github.com/rs/zerolog/log"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
)

// Start 启动服务
func Start(isSwag bool, configPath string) {

	conf, err := config.NewConfig(configPath)

	if err != nil || len(conf.Engines) == 0 {
		panic("Failed to load configuration")
	}

	var engines []*engine.ConCurrentEngine

	// 读取配置 会启动很多监听 可同时监听 20 和 原生
	for _, engineConfig := range conf.Engines {
		eth, err := engine.NewEngine(engineConfig)
		if err != nil {
			panic(fmt.Sprintf("eth run err：%v", err))
		}
		engines = append(engines, eth)
	}

	// 启动监听器
	for _, currentEngine := range engines {
		go currentEngine.Run()
	}

	if isSwag {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	server := gin.Default()

	// 中间件
	server.Use(gin.Logger())
	server.Use(gin.Recovery())
	server.Use(SetEngine(engines...))

	auth := server.Group("/api", AuthRequired())
	{
		auth.POST("/createWallet", CreateWallet)
		auth.POST("/delWallet", DelWallet)
		auth.POST("/withdraw", Withdraw)
		auth.POST("/collection", Collection)
		auth.GET("/getTransactionReceipt", GetTransactionReceipt)
	}

	// TODO 发起一笔交易
	server.POST("/transaction", Transaction)

	// 添加新币
	server.POST("/addNewCoin", AddNewCoin)

	// 获取实时的 gas 费用 链上状态
	server.POST("/getLinkStatus", GetLinkStatus)
	// 获取账户的余额信息
	server.POST("/getBalance", GetBalance)
	// 添加网络
	server.POST("/addNetWork", AddNetWork)
	// 获取钱包基础信息
	server.GET("/getWalletInfo", GetWalletInfo)

	// 获取账户的活动信息
	server.GET("/getActivity", GetActivity)

	if isSwag {
		swagHandler := ginSwagger.WrapHandler(swaggerFiles.Handler)
		server.GET("/swagger/*any", swagHandler)
	}

	err = server.Run(fmt.Sprintf(":%v", conf.App.Port))
	if err != nil {
		panic("start error")
	}

	log.Info().Msgf("start success")

}
