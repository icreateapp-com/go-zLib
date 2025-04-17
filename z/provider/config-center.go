package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	. "github.com/icreateapp-com/go-zLib/z"
	"net/url"
)

type ConfigCenterProviderEnv struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Version int64  `json:"version"`
}

type ConfigCenterProviderConfigItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ConfigCenterProviderConfig struct {
	Configs []ConfigCenterProviderConfigItem `json:"configs"`
	Version int64                            `json:"version"`
}

type ConfigCenterProviderEnvNotify struct {
	Id      string `json:"id"`
	Code    string `json:"code"`
	Version int64  `json:"version"`
}

type configCenterProvider struct {
	version  int64
	address  string
	envId    string
	token    string
	callback string
	clientId string
}

var ConfigCenterProvider configCenterProvider

// Register 注册配置中心
// 示例：
// 配置文件示例：
// config:
//
//	name: example-service
//	port: 8080
//	config_center:
//	  version: 1
//	  address: http://config-center.example.com
//	  env_id: env123
//	  token: token123
//	  callback: http://example-service.example.com/.well-known/config
func (c *configCenterProvider) Register() {
	var err error
	c.version, err = Config.Int64("config.config_center.version")
	if err != nil {
		Error.Fatal("config.config_center.version not found")
	}
	c.address, err = Config.String("config.config_center.address")
	if err != nil {
		Error.Fatal("config.config_center.address not found")
	}
	c.envId, err = Config.String("config.config_center.env_id")
	if err != nil {
		Error.Fatal("config.config_center.env_id not found")
	}
	c.token, err = Config.String("config.config_center.token")
	if err != nil {
		Error.Fatal("config.config_center.token not found")
	}
	c.callback, err = Config.String("config.config_center.callback")
	if err != nil {
		Error.Fatal("config.config_center.callback not found")
	}

	// 生成客户端ID
	name, err := Config.String("config.name")
	if err != nil {
		Error.Fatal("config.name not found")
	}
	ip, err := GetLocalIP()
	if err != nil {
		Error.Fatalf("get local ip error: %s", err.Error())
	}
	port, err := Config.Int("config.port")
	c.clientId = GetMd5(fmt.Sprintf("%s -> %s:%d", name, ip, port))

	// 配置变更回调地址
	notifyUrl := url.QueryEscape(c.callback)

	headers := map[string]string{"Authorization": c.token}
	path := fmt.Sprintf("/api/client/env/%s/version?notify_url=%s&client_id=%s", c.envId, notifyUrl, c.clientId)

	get, err := Get(fmt.Sprintf("%s%s", c.address, path), headers)
	if err != nil {
		Error.Fatalf("config center response error: %s", err.Error())
	}

	var response Response
	if err := json.Unmarshal([]byte(get), &response); err != nil {
		Error.Fatalf("config center response error: %s", err.Error())
	}

	if !response.Success {
		Error.Fatalf("config center response error: %s", response.Message)
	}

	Info.Println("config center register success")

	var env ConfigCenterProviderEnv

	if err := ToStruct(response.Message, &env); err != nil {
		Error.Fatalf("config center response error: %s", err.Error())
	}

	if env.Version == 0 || env.Version != c.version {
		if err := c.Sync(); err != nil {
			Error.Fatalf("config center sync error: %s", err.Error())
		}

		Info.Println("config center sync success")
	}
}

// Sync 同步配置
func (c *configCenterProvider) Sync() error {
	var err error

	headers := map[string]string{"Authorization": c.token}
	path := fmt.Sprintf("/api/client/env/%s/configs", c.envId)

	get, err := Get(fmt.Sprintf("%s%s", c.address, path), headers)
	if err != nil {
		return err
	}

	var response Response
	if err := json.Unmarshal([]byte(get), &response); err != nil {
		return err
	}

	if !response.Success {
		return errors.New(ToString(response.Message))
	}

	var configs ConfigCenterProviderConfig
	if err := ToStruct(response.Message, &configs); err != nil {
		return err
	}

	// 将配置转为map
	configsMap := map[string]interface{}{}
	for _, config := range configs.Configs {
		configsMap[config.Key] = config.Value
	}

	if err := Config.SetEnvs(configsMap); err != nil {
		return err
	}

	c.version = configs.Version

	return nil
}

// Middleware 提供中间件来处理配置中心的通知
// 示例：
// 当接收到配置中心的通知时，中间件会检查版本并同步配置
// 请求示例：
// GET /.well-known/config?id=123&code=env123&version=2
func (c *configCenterProvider) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.URL.Path == "/.well-known/config" {
			var notify ConfigCenterProviderEnvNotify
			if err := ctx.ShouldBindQuery(&notify); err != nil {
				Error.Println("config center notify error: %s", err.Error())
			} else {
				if notify.Version == 0 || notify.Version != c.version {
					if err := c.Sync(); err != nil {
						Error.Println("config center sync error: %s", err.Error())
					}
					Info.Println("config center sync success")
				}
			}
		}
		ctx.Next()
	}
}
