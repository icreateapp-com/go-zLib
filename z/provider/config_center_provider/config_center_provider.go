package config_center_provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/gin-gonic/gin"
	. "github.com/icreateapp-com/go-zLib/z"
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

// configCenterProvider 配置中心提供者
type configCenterProvider struct {
	version  int64
	address  string
	envId    string
	token    string
	callback string
	clientId string
	mutex    sync.RWMutex // 读写锁保护所有字段
}

// ConfigCenterProvider 全局配置中心提供者实例
var ConfigCenterProvider = &configCenterProvider{}

// Register 注册配置中心
func (c *configCenterProvider) Register() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

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
	// 注意：这里不能加锁，因为Register方法已经加锁了
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
func (c *configCenterProvider) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.URL.Path == "/.well-known/config" {
			var notify ConfigCenterProviderEnvNotify
			if err := ctx.ShouldBindQuery(&notify); err != nil {
				Error.Printf("config center notify error: %s", err.Error())
			} else {
				c.mutex.RLock()
				currentVersion := c.version
				c.mutex.RUnlock()

				if notify.Version == 0 || notify.Version != currentVersion {
					c.mutex.Lock()
					if err := c.Sync(); err != nil {
						Error.Printf("config center sync error: %s", err.Error())
					} else {
						Info.Println("config center sync success")
					}
					c.mutex.Unlock()
				}
			}
		}
		ctx.Next()
	}
}

// GetVersion 获取当前配置版本
func (c *configCenterProvider) GetVersion() int64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.version
}
