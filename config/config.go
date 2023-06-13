package config

import (
	"context"
	"encoding/json"
	"github.com/jinzhu/configor"
	"github.com/lmxdawn/wallet/db"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type AppConfig struct {
	Port uint `yaml:"port"`
}

type EngineConfig struct {
	Network string `yaml:"network"` // 网络名称（暂时BTC协议有用{MainNet：主网，TestNet：测试网，TestNet3：测试网3，SimNet：测试网}）
	Rpc     string `yaml:"rpc"`     // rpc配置
	User    string `yaml:"user"`    // rpc用户名（没有则为空）
	Pass    string `yaml:"pass"`    // rpc密码（没有则为空）
}

type Config struct {
	App     AppConfig
	Engines []EngineConfig
}

func NewConfig(confPath string) (Config, error) {
	var config Config
	if confPath != "" {
		err := configor.Load(&config, confPath)
		if err != nil {
			return config, err
		}
	} else {
		err := configor.Load(&config, "config/config-example.yml")
		if err != nil {
			return config, err
		}
	}
	// 将配置文件写入数据库
	loadToDB(&config)
	return config, nil
}

func (c Config) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

func (c Config) UnmarshalBinary(data []byte) error {

	return json.Unmarshal(data, &c)
}

func loadToDB(config *Config) {
	_, err := db.Rdb.Get(context.Background(), db.ConfigDB).Result()
	if err == redis.Nil {
		res, err := db.Rdb.Set(context.Background(), db.ConfigDB, *config, 0).Result()
		if err != nil {
			log.Fatal().Msgf("loadToDB err is %s ", err.Error())
			return
		}
		log.Info().Msgf("loadToDB info is %s", res)
	}
	if err != nil {
		log.Fatal().Msgf("loadToDB get err is %s ", err.Error())
		return
	}
}
