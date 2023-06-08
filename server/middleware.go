package server

import (
	"github.com/gin-gonic/gin"
	"github.com/lmxdawn/wallet/db"
	"github.com/lmxdawn/wallet/engine"
	"github.com/rs/zerolog/log"
)

// AuthRequired 认证中间件
func AuthRequired() gin.HandlerFunc {

	return func(c *gin.Context) {

		token := c.GetHeader("Account")
		if token == "" {
			c.Abort()
			APIResponse(c, ErrToken, nil)
		}
		if !db.CheckLoginInfo(token) {
			c.Abort()
			APIResponse(c, ErrLoginExpire, nil)
		}
		// 检验有效期
		//session :=sessions.Default(c)
		//session.

	}

}

// SetEngine 设置db数据库
func SetEngine(engines ...*engine.ConCurrentEngine) gin.HandlerFunc {

	return func(c *gin.Context) {
		for _, currentEngine := range engines {
			// TODO 换掉这种处理方式
			// 跨请求取值 方便在其他地方使用
			c.Set(currentEngine.Protocol+currentEngine.CoinName, currentEngine)
			c.Set(currentEngine.Config.Rpc, currentEngine)
		}
	}
}

func start20Token() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokens := db.GetAll20TokenFromDB()
		for _, v := range tokens {
			eng, err := engine.AddNewCoin(v.CoinName, v.ContractAddress)
			if err != nil {
				log.Error().Msgf("start20TokenListen coinName is  %s err is %s ", v.CoinName, err.Error())
			}
			c.Set(v.Protocol+v.CoinName, eng)
		}
	}
}
