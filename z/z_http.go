package z

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// RequestContentType 定义 HTTP 内容类型
type RequestContentType string

const (
	RequestContentTypeJSON      RequestContentType = "application/json"
	RequestContentTypeForm      RequestContentType = "application/x-www-form-urlencoded"
	RequestContentTypeMultipart RequestContentType = "multipart/form-data"
	RequestContentTypeXML       RequestContentType = "application/xml"
	RequestContentTypeBinary    RequestContentType = "application/octet-stream"
	RequestContentTypeRaw       RequestContentType = "raw"
)

// MultipartField 支持普通字段和文件字段
type MultipartField struct {
	FileName string    // 文件名，仅在 IsFile 为 true 时生效
	Reader   io.Reader // 读取内容
	IsFile   bool      // 是否为文件
}

// RequestOptions 请求选项
type RequestOptions struct {
	URL         string
	Method      string
	Headers     map[string]string
	ContentType RequestContentType
	Data        interface{}
	Timeout     time.Duration
}

var (
	defaultClient *http.Client
	once          sync.Once
)

// 获取单例 client（复用连接池）
func getClient() *http.Client {
	once.Do(func() {
		defaultClient = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		}
	})
	return defaultClient
}

// Request 发起请求
func Request(opt RequestOptions) ([]byte, error) {
	if opt.Method == "" {
		opt.Method = http.MethodPost
	}
	if opt.Timeout <= 0 {
		opt.Timeout = 10 * time.Second
	}
	headers := make(http.Header)
	var body io.Reader

	// 构造 body 和 headers
	switch opt.ContentType {
	case RequestContentTypeJSON:
		jsonBytes, err := json.Marshal(opt.Data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonBytes)
		headers.Set("Content-Type", string(RequestContentTypeJSON))

	case RequestContentTypeForm:
		form, ok := opt.Data.(map[string]string)
		if !ok {
			return nil, errors.New("form content-type requires map[string]string")
		}
		values := url.Values{}
		for k, v := range form {
			values.Set(k, v)
		}
		body = strings.NewReader(values.Encode())
		headers.Set("Content-Type", string(RequestContentTypeForm))

	case RequestContentTypeMultipart:
		form, ok := opt.Data.(map[string]MultipartField)
		if !ok {
			return nil, errors.New("multipart content-type requires map[string]MultipartField")
		}
		var b bytes.Buffer
		writer := multipart.NewWriter(&b)
		for key, field := range form {
			if field.IsFile {
				part, err := writer.CreateFormFile(key, field.FileName)
				if err != nil {
					return nil, err
				}
				_, err = io.Copy(part, field.Reader)
				if err != nil {
					return nil, err
				}
			} else {
				err := writer.WriteField(key, readToString(field.Reader))
				if err != nil {
					return nil, err
				}
			}
		}
		writer.Close()
		body = &b
		headers.Set("Content-Type", writer.FormDataContentType())

	case RequestContentTypeXML:
		xmlBytes, err := xml.Marshal(opt.Data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(xmlBytes)
		headers.Set("Content-Type", string(RequestContentTypeXML))

	case RequestContentTypeBinary:
		bin, ok := opt.Data.([]byte)
		if !ok {
			return nil, errors.New("binary content-type requires []byte")
		}
		body = bytes.NewReader(bin)
		headers.Set("Content-Type", string(RequestContentTypeBinary))

	case RequestContentTypeRaw:
		switch v := opt.Data.(type) {
		case string:
			body = strings.NewReader(v)
		case []byte:
			body = bytes.NewReader(v)
		case io.Reader:
			body = v
		default:
			return nil, errors.New("raw content-type requires string, []byte, or io.Reader")
		}

	default:
		return nil, errors.New("unsupported content-type")
	}

	// 构造请求上下文
	ctx, cancel := context.WithTimeout(context.Background(), opt.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, opt.Method, opt.URL, body)
	if err != nil {
		return nil, err
	}

	// 合并 headers
	for k, v := range opt.Headers {
		req.Header.Set(k, v)
	}
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Set(k, v) // 防止覆盖用户自定义的 headers
		}
	}

	// 发起请求
	client := getClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http error: %s\n%s", resp.Status, string(respBody))
	}

	return respBody, nil
}

// RequestSSEChannel 发起 SSE 请求，返回一个只读通道供外部消费事件
func RequestSSEChannel(opt RequestOptions) (<-chan string, <-chan error, context.CancelFunc, error) {
	if opt.Method == "" {
		opt.Method = http.MethodGet
	}
	if opt.Timeout == 0 {
		opt.Timeout = 15 * time.Second
	}

	var body io.Reader

	if opt.Method == http.MethodPost {
		switch opt.ContentType {
		case RequestContentTypeJSON:
			jsonBytes, err := json.Marshal(opt.Data)
			if err != nil {
				return nil, nil, nil, err
			}
			body = bytes.NewBuffer(jsonBytes)
			if opt.Headers == nil {
				opt.Headers = make(map[string]string)
			}
			opt.Headers["Content-Type"] = string(RequestContentTypeJSON)
		default:
			// 如果是其他类型，我们假设 Data 已经是 io.Reader
			if r, ok := opt.Data.(io.Reader); ok {
				body = r
			}
		}
	}

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), opt.Timeout)
	req, err := http.NewRequestWithContext(ctx, opt.Method, opt.URL, body)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}

	// 添加 headers
	for k, v := range opt.Headers {
		req.Header.Set(k, v)
	}
	// SSE 必须为 text/event-stream
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{}

	// 发起请求
	resp, err := client.Do(req)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		cancel()
		return nil, nil, nil, errors.New("unexpected status: " + resp.Status)
	}

	eventChan := make(chan string)
	errChan := make(chan error, 1)

	// 后台读取流
	go func() {
		defer close(eventChan)
		defer close(errChan)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					errChan <- err
				}
				return
			}
			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimPrefix(line, "data:")
			eventChan <- strings.TrimSpace(data)
		}
	}()

	return eventChan, errChan, cancel, nil
}

// PostSSEChannel 发起 SSE 流式 POST 请求
func PostSSEChannel(url string, data interface{}, headers map[string]string) (<-chan string, <-chan error, context.CancelFunc, error) {
	return RequestSSEChannel(RequestOptions{
		URL:         url,
		Method:      http.MethodPost,
		ContentType: RequestContentTypeJSON,
		Data:        data,
		Headers:     headers,
	})
}

// GetSSEChannel 发起 SSE 流式 GET 请求
func GetSSEChannel(url string, headers map[string]string) (<-chan string, <-chan error, context.CancelFunc, error) {
	return RequestSSEChannel(RequestOptions{
		URL:     url,
		Method:  http.MethodGet,
		Headers: headers,
	})
}

// 读取流为字符串（用于 Multipart 普通字段）
func readToString(r io.Reader) string {
	if r == nil {
		return ""
	}
	b, _ := io.ReadAll(r)
	return string(b)
}

// Get 请求
func Get(url string, headers map[string]string) ([]byte, error) {
	return Request(RequestOptions{
		URL:     url,
		Method:  http.MethodGet,
		Headers: headers,
	})
}

// Post 提交
func Post(url string, data interface{}, headers map[string]string, contentType RequestContentType) ([]byte, error) {
	return Request(RequestOptions{
		URL:         url,
		Method:      http.MethodPost,
		Headers:     headers,
		Data:        data,
		ContentType: contentType,
	})
}

// Put 修改
func Put(url string, data interface{}, headers map[string]string, contentType RequestContentType) ([]byte, error) {
	return Request(RequestOptions{
		URL:         url,
		Method:      http.MethodPut,
		Headers:     headers,
		Data:        data,
		ContentType: contentType,
	})
}

// Delete 删除
func Delete(url string, headers map[string]string) ([]byte, error) {
	return Request(RequestOptions{
		URL:     url,
		Method:  http.MethodDelete,
		Headers: headers,
	})
}

// Download 下载文件
func Download(url string, filePath string) error {
	// 发送HTTP请求获取图片数据
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// 检查HTTP响应状态码
	if response.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("HTTP response error: %d", response.StatusCode))
	}

	// 创建目录（如果不存在）
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 将图片数据写入文件
	if _, err := io.Copy(file, response.Body); err != nil {
		return err
	}

	return nil
}

// GetUrl 生成当前服务器的 URL 地址
func GetUrl(params string) string {
	urlCfg, _ := Config.String("_config.url")

	return fmt.Sprintf("%s/%s", strings.Trim(urlCfg, "/"), strings.Trim(params, "/"))
}

// IsUrl 判断是否是有效的URL
func IsUrl(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}

// GetLocalIP 获取本地 IP 地址
func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		// 检查 IP 地址是否为 IPv4 并且不是回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("unable to find local IP address")
}

// AppendQueryParamsToURL 将查询参数字典拼接到 URL 中
func AppendQueryParamsToURL(originalURL string, params map[string]interface{}) (string, error) {
	// 解析原始 URL
	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	queryValues := parsedURL.Query()

	for key, value := range params {
		queryValues.Add(key, ToString(value))
	}

	parsedURL.RawQuery = queryValues.Encode()

	return parsedURL.String(), nil
}

// MatchIP 检查给定的客户端 IP 是否匹配允许的 IP 模式（支持通配符 *）
func MatchIP(clientIP, allowedIP string) bool {
	// 将允许的 IP 和客户端 IP 转换为字符串切片
	allowedParts := strings.Split(allowedIP, ".")
	clientParts := strings.Split(clientIP, ".")

	// 如果两者的点分段数量不同，则直接不匹配
	if len(allowedParts) != len(clientParts) {
		return false
	}

	// 逐段检查匹配情况
	for i := 0; i < len(allowedParts); i++ {
		if allowedParts[i] == "*" {
			continue // 通配符 * 可以匹配任意数字
		}
		if allowedParts[i] != clientParts[i] {
			return false // 当前段不匹配
		}
	}

	return true // 所有段都匹配
}

// IsLocalIP 检查给定的 IP 是否为本地 IP 地址
func IsLocalIP(ip string) bool {
	// 检查常见的本地 IP 地址
	if ip == "127.0.0.1" || ip == "::1" || ip == "0.0.0.0" || ip == "::" {
		return true
	}

	// 解析 IP 地址
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// 检查是否为本地回环地址
	if parsedIP.IsLoopback() {
		return true
	}

	// 检查是否为本地链路地址（169.254.0.0/16）
	if ipv4 := parsedIP.To4(); ipv4 != nil && ipv4[0] == 169 && ipv4[1] == 254 {
		return true
	}

	return false
}
