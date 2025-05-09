package provider

import (
	"encoding/json"
	"fmt"
	. "github.com/icreateapp-com/go-zLib/z"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/url"
	"strings"
	"time"
)

type ServiceDiscoverServiceInfo struct {
	Name      string            `json:"name"`       // 服务名称
	Host      string            `json:"host"`       // 服务地址
	Port      int               `json:"port"`       // 服务端口
	GrpcHost  string            `json:"grpc_host"`  // GRPC 地址
	GrpcPort  int               `json:"grpc_port"`  // GRPC 端口
	Latency   int               `json:"latency"`    // 响应延迟
	AuthToken map[string]string `json:"auth_token"` // 访问令牌
}

type ServiceDiscoverServicesResponse struct {
	Success bool                         `json:"success"`
	Message []ServiceDiscoverServiceInfo `json:"message"`
	Code    int                          `json:"code"`
}

type ServiceDiscoverServiceResponse struct {
	Success bool                       `json:"success"`
	Message ServiceDiscoverServiceInfo `json:"message"`
	Code    int                        `json:"code"`
}

type ServiceRequestParam struct {
	Path    string
	Method  string
	Query   map[string]interface{}
	Data    map[string]interface{}
	Headers map[string]string
}

type serviceDiscoverProvider struct {
	AutoCheck       bool
	CheckInterval   int
	LostTimeout     int64
	CacheService    map[string]ServiceDiscoverServiceInfo
	CacheExpiration map[string]time.Time
	CacheTTL        int
}

var ServiceDiscoverProvider serviceDiscoverProvider

// Register 服务注册
// 示例：
// 在配置文件中添加以下配置项：
// config:
//
//	service_discover:
//	  address: "http://localhost:8080"
//	  apikey: "your_api_key"
//	  cache_ttl: 300
//	  check: true
//	  check_interval: 5
//	  lost_timeout: 10
//	name: "your_service_name"
//	port: 8081
//	auth:
//	  token1: "value1"
//	  token2: "value2"
func (s *serviceDiscoverProvider) Register() {
	// 初始化缓存
	s.CacheService = make(map[string]ServiceDiscoverServiceInfo)
	s.CacheExpiration = make(map[string]time.Time)

	if ttl, err := Config.Int("config.service_discover.cache_ttl"); err != nil {
		s.CacheTTL = 0
	} else {
		s.CacheTTL = ttl
	}

	// 注册服务
	if err := s.registerService(); err != nil {
		Error.Fatalf("service discover register failure: %s", err.Error())
	}

	// 自动检查服务健康状态
	check, err := Config.Bool("config.service_discover.check")
	s.AutoCheck = err == nil && check

	checkInterval, err := Config.Int("config.service_discover.check_interval")
	if err != nil {
		checkInterval = 5
	}
	s.CheckInterval = checkInterval

	lostTimeout, err := Config.Int64("config.service_discover.lost_timeout")
	if err != nil {
		lostTimeout = 10
	}
	s.LostTimeout = lostTimeout

	if s.AutoCheck {
		go s.startHealthCheck()
	}
}

// registerService 注册服务的具体逻辑
func (s *serviceDiscoverProvider) registerService() error {
	address, err := Config.String("config.service_discover.address")
	if err != nil {
		return err
	}

	address = TernaryString(strings.HasPrefix(address, "http"), address, "http://"+address)
	address = strings.TrimSuffix(address, "/")

	apikey, err := Config.String("config.service_discover.apikey")
	if err != nil {
		return err
	}

	name, _ := Config.String("config.name")
	host, _ := Config.String("config.host")
	port, _ := Config.Int("config.port")
	grpcHost, _ := Config.String("config.grpc.host")
	grpcPort, _ := Config.Int("config.grpc.port")

	if len(host) == 0 || host == "0.0.0.0" {
		host, _ = GetLocalIP()
	}

	if len(grpcHost) == 0 || grpcHost == "0.0.0.0" {
		grpcHost, _ = GetLocalIP()
	}

	tokens, err := Config.StringMap("config.auth")
	if err != nil {
		return err
	}

	maxRetries := 10
	retryInterval := 5

	for attempt := 1; attempt <= maxRetries; attempt++ {
		data := map[string]interface{}{
			"name":       name,
			"port":       port,
			"host":       host,
			"grpc_host":  grpcHost,
			"grpc_port":  grpcPort,
			"auth_token": tokens,
		}
		res, err := PostJson(
			address+"/api/service/register",
			data,
			map[string]string{"Authorization": apikey},
		)
		if err != nil {
			if attempt < maxRetries {
				Warn.Printf("Retrying in %d seconds...", retryInterval)
				time.Sleep(time.Duration(retryInterval) * time.Second)
			} else {
				return err
			}
			continue
		}

		var response Response
		if err := json.Unmarshal([]byte(res), &response); err != nil {
			if attempt < maxRetries {
				Warn.Printf("Retrying in %d seconds...", retryInterval)
				time.Sleep(time.Duration(retryInterval) * time.Second)
			} else {
				return err
			}
			continue
		}

		if !response.Success {
			if attempt < maxRetries {
				Warn.Printf("Retrying in %d seconds..., %s", retryInterval, response.Message)
				time.Sleep(time.Duration(retryInterval) * time.Second)
			} else {
				return fmt.Errorf("registration failed: %s", response.Message)
			}
			continue
		}

		Info.Println("Service register success")
		break
	}

	return nil
}

// startHealthCheck 启动健康检查定时任务
func (s *serviceDiscoverProvider) startHealthCheck() {
	ticker := time.NewTicker(time.Duration(s.CheckInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		lastHealthCheckTime, has := MemCache.Get("__last_health_check_time__")
		if has {
			if lastTime, ok := lastHealthCheckTime.(int64); ok {
				if time.Now().Unix()-lastTime > s.LostTimeout {
					Warn.Println("health check failed, re-registering service")
					ticker.Stop()
					if err := s.registerService(); err != nil {
						Error.Printf("re-registration failed: %v", err)
					}
					ticker = time.NewTicker(3 * time.Second)
				}
			}
		}
	}
}

// GetAllServiceAddress 获取所有服务地址
func (s *serviceDiscoverProvider) GetAllServiceAddress(name string) (*[]ServiceDiscoverServiceInfo, error) {
	address, err := Config.String("config.service_discover.address")
	if err != nil {
		Error.Fatalf("service register failed: %s", err.Error())
	}

	// 确保地址格式正确
	address = strings.TrimSuffix(address, "/")
	if !strings.HasPrefix(address, "http") {
		address = "http://" + address
	}

	apikey, err := Config.String("config.service_discover.apikey")
	if err != nil {
		Error.Fatalf("service register failed: %s", err.Error())
	}

	urlStr := address + "/api/service/instances?name=" + name
	res, err := Get(urlStr, map[string]string{"Authorization": apikey})
	if err != nil {
		return nil, err
	}

	var response ServiceDiscoverServicesResponse
	if err := json.Unmarshal([]byte(res), &response); err != nil {
		return nil, err
	}

	return &response.Message, nil
}

// GetBestServiceAddress 获取最佳服务地址
func (s *serviceDiscoverProvider) GetBestServiceAddress(name string) (*ServiceDiscoverServiceInfo, error) {
	// 从缓存中获取服务信息
	if s.CacheTTL > 0 {
		if cachedService, ok := s.CacheService[name]; ok {
			if expiration, expOk := s.CacheExpiration[name]; expOk && time.Now().Before(expiration) {
				return &cachedService, nil
			}
		}
	}

	address, err := Config.String("config.service_discover.address")
	if err != nil {
		Error.Fatalf("service register failed: %s", err.Error())
	}

	// 确保地址格式正确
	address = strings.TrimSuffix(address, "/")
	if !strings.HasPrefix(address, "http") {
		address = "http://" + address
	}

	apikey, err := Config.String("config.service_discover.apikey")
	if err != nil {
		Error.Fatalf("service register failed: %s", err.Error())
	}

	urlStr := address + "/api/service/instance?name=" + name
	res, err := Get(urlStr, map[string]string{"Authorization": apikey})
	if err != nil {
		return nil, err
	}

	var response ServiceDiscoverServiceResponse
	if err := json.Unmarshal([]byte(res), &response); err != nil {
		return nil, err
	}

	if s.CacheTTL > 0 {
		s.CacheService[name] = response.Message
		s.CacheExpiration[name] = time.Now().Add(time.Duration(s.CacheTTL) * time.Second)
	}

	return &response.Message, nil
}

// Call 调用服务
func (s *serviceDiscoverProvider) Call(name string, request ServiceRequestParam, response *interface{}) error {
	// 获取服务
	name = strings.ToLower(name)
	service, err := ServiceDiscoverProvider.GetBestServiceAddress(name)
	if err != nil {
		return err
	}

	// 检查服务地址是否为空
	if service.Host == "" {
		return fmt.Errorf("service address is empty")
	}

	// 检查端口是否有效
	if service.Port < 1 || service.Port > 65535 {
		return fmt.Errorf("invalid service port: %d", service.Port)
	}

	// 解析 request.Path 中的 URL
	parsedURL, urlErr := url.Parse(request.Path)
	if urlErr != nil {
		return fmt.Errorf("failed to parse request path: %v", urlErr)
	}

	// 获取 request.Path 中的查询参数
	queryParams := parsedURL.Query()

	// 合并 request.Query 中的参数
	for key, value := range request.Query {
		queryParams.Add(key, fmt.Sprintf("%v", value))
	}

	// 重新构建请求 URL
	parsedURL.RawQuery = queryParams.Encode()
	fullUrl := fmt.Sprintf("http://%s:%d%s", service.Host, service.Port, parsedURL.String())

	// 构建请求头
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	if len(request.Headers) > 0 {
		for key, value := range request.Headers {
			headers[key] = value
		}
	}

	// 标准化请求路径
	requestPath := strings.TrimPrefix(request.Path, "/")

	// 匹配 AuthToken
	for pathPrefix, token := range service.AuthToken {
		// 标准化路径前缀
		normalizedPrefix := strings.TrimPrefix(pathPrefix, "/")

		if strings.HasPrefix(requestPath, normalizedPrefix) {
			headers["Authorization"] = token
			break
		}
	}

	// 发起请求
	var res string
	switch request.Method {
	case "POST":
		res, err = Post(fullUrl, request.Data, headers)
	case "PUT":
		res, err = Put(fullUrl, request.Data, headers)
	case "DELETE":
		res, err = Delete(fullUrl, headers)
	default:
		res, err = Get(fullUrl, headers)
	}

	if err != nil {
		return err
	}

	// 解析响应
	err = json.Unmarshal([]byte(res), &response)
	if err != nil {
		return err
	}

	return nil
}

// Grpc gRPC 服务
func (s *serviceDiscoverProvider) Grpc(name string, handler func(*ServiceDiscoverServiceInfo, *grpc.ClientConn) error) error {
	// 获取服务
	name = strings.ToLower(name)
	service, err := ServiceDiscoverProvider.GetBestServiceAddress(name)
	if err != nil {
		return err
	}

	// 检查服务地址是否为空
	if service.GrpcHost == "" {
		return fmt.Errorf("service address is empty")
	}

	// 检查端口是否有效
	if service.GrpcPort < 1 || service.GrpcPort > 65535 {
		return fmt.Errorf("invalid service port: %d", service.Port)
	}

	// 连接gRPC服务器
	address := fmt.Sprintf("%s:%d", service.GrpcHost, service.GrpcPort)
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("service %s connection error: %s", service.Name, service.Host)
	}
	defer conn.Close()

	return handler(service, conn)
}
