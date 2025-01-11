package provider

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/icreateapp-com/go-zLib/z"
	"time"
)

func ServiceRegisterProvider(engine *gin.Engine) error {
	address, err := z.Config.String("config.service_discover.address")
	if err != nil {
		z.Error.Println(err.Error())
		return err
	}
	apikey, err := z.Config.String("config.service_discover.apikey")
	if err != nil {
		z.Error.Println(err.Error())
		return err
	}

	name, _ := z.Config.String("config.name")
	port, _ := z.Config.Int("config.port")

	ip, err := z.GetLocalIP()
	if err != nil {
		z.Error.Println(err.Error())
		return err
	}

	maxRetries := 10
	retryInterval := 5

	for attempt := 1; attempt <= maxRetries; attempt++ {
		res, err := z.Post(
			address+"/api/service/register",
			map[string]interface{}{
				"name":    name,
				"port":    port,
				"address": ip,
			},
			map[string]string{"Authorization": apikey},
		)
		if err != nil {
			z.Error.Printf("Attempt %d: Failed to register service: %v", attempt, err)
			if attempt < maxRetries {
				z.Warn.Printf("Retrying in %d seconds...", retryInterval)
				time.Sleep(time.Duration(retryInterval) * time.Second)
			} else {
				return errors.New("maximum retries reached, service registration failed")
			}
			continue
		}

		var response z.Response
		if err := json.Unmarshal([]byte(res), &response); err != nil {
			z.Error.Printf("Attempt %d: Failed to unmarshal response: %v", attempt, err)
			if attempt < maxRetries {
				z.Warn.Printf("Retrying in %d seconds...", retryInterval)
				time.Sleep(time.Duration(retryInterval) * time.Second)
			} else {
				return errors.New("maximum retries reached, service registration failed")
			}
			continue
		}

		if !response.Success {
			z.Error.Printf("Attempt %d: Service registration failed: %s", attempt, z.ToString(response.Message))
			if attempt < maxRetries {
				z.Warn.Printf("Retrying in %d seconds...", retryInterval)
				time.Sleep(time.Duration(retryInterval) * time.Second)
			} else {
				return errors.New("maximum retries reached, service registration failed")
			}
			continue
		}

		z.Info.Println("Service register success")
		break
	}

	return nil
}
