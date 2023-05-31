package server

import (
	"github.com/gin-gonic/gin"
	"github.com/lmxdawn/wallet/engine"
)

// AuthRequired 认证中间件
func AuthRequired() gin.HandlerFunc {

	return func(c *gin.Context) {

		token := c.GetHeader("x-token")
		if token == "" {
			c.Abort()
			APIResponse(c, ErrToken, nil)
		}

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
