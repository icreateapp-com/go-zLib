package z

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// SendRequestResponse 返回结果结构体
type SendRequestResponse struct {
	StatusCode int
	Body       string
	Headers    http.Header
}

// GetUrl 生成当前服务器的 URL 地址
func GetUrl(params string) string {
	urlCfg, _ := Config.String("_config.url")

	return fmt.Sprintf("%s/%s", strings.Trim(urlCfg, "/"), strings.Trim(params, "/"))
}

// Post 发起 POST 请求
func Post(url string, data map[string]interface{}, headers map[string]string) (string, error) {
	values := ToValues(data)

	// 创建一个新的 HTTP 请求
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return "", err
	}

	// 设置默认的 Content-Type
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 添加可选的 HTTP 头信息
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// PostJson 发起 POST JSON 请求
func PostJson(url string, data map[string]interface{}, headers map[string]string) (string, error) {
	// 将数据序列化为 JSON 字符串
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// 创建一个新的 HTTP 请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	// 设置默认的 Content-Type
	req.Header.Set("Content-Type", "application/json")

	// 添加可选的 HTTP 头信息
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// Get 发起 GET 请求
func Get(url string, headers map[string]string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// 添加可选的 HTTP 头信息
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// Put 发送 PUT 请求
func Put(url string, data map[string]interface{}, headers map[string]string) (string, error) {
	values := ToValues(data)

	req, err := http.NewRequest("PUT", url, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return "", err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(respBody), nil
}

// Delete 发送 DELETE 请求
func Delete(url string, headers map[string]string) (string, error) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return "", err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(respBody), nil
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

// Request 发送请求
func Request(url string, method string, headers map[string]string, paramType string, params map[string]interface{}) (*SendRequestResponse, error) {
	var req *http.Request
	var err error

	// 创建请求体
	var body io.Reader
	switch strings.ToLower(paramType) {
	case "form-data":
		body, err = CreateFormData(params)
	case "x-www-form-urlencoded":
		body, err = CreateFormURLEncoded(params)
	case "json":
		body, err = CreateJSON(params)
	case "xml":
		body, err = CreateXML(params)
	case "raw":
		body = CreateRaw(params)
	case "binary":
		body = CreateBinary(params)
	default:
		return nil, fmt.Errorf("unsupported param type: %s", paramType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request body: %w", err)
	}

	// 创建请求
	req, err = http.NewRequest(strings.ToUpper(method), url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 构建返回结果
	response := &SendRequestResponse{
		StatusCode: resp.StatusCode,
		Body:       string(respBody),
		Headers:    resp.Header,
	}

	return response, nil
}

// CreateFormData 创建 Form-Data 请求体
func CreateFormData(params map[string]interface{}) (io.Reader, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for key, value := range params {
		if err := writer.WriteField(key, ToString(value)); err != nil {
			return nil, fmt.Errorf("failed to write form field %s: %w", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close form writer: %w", err)
	}

	return body, nil
}

// CreateFormURLEncoded 创建 x-www-form-urlencoded 请求体
func CreateFormURLEncoded(params map[string]interface{}) (io.Reader, error) {
	values := url.Values{}
	for key, value := range params {
		values.Add(key, ToString(value))
	}
	return strings.NewReader(values.Encode()), nil
}

// CreateJSON 创建 JSON 请求体
func CreateJSON(params map[string]interface{}) (io.Reader, error) {
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return bytes.NewBuffer(jsonData), nil
}

// CreateXML 创建 XML 请求体
func CreateXML(params map[string]interface{}) (io.Reader, error) {
	xmlData, err := xml.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal XML: %w", err)
	}
	return bytes.NewBuffer(xmlData), nil
}

// CreateRaw 创建 Raw 请求体
func CreateRaw(params map[string]interface{}) io.Reader {
	// 假设 params 是一个包含 raw 数据的 map
	if raw, ok := params["raw"].(string); ok {
		return strings.NewReader(raw)
	}
	return strings.NewReader("")
}

// CreateBinary 创建 Binary 请求体
func CreateBinary(params map[string]interface{}) io.Reader {
	// 假设 params 是一个包含 binary 数据的 map
	if binary, ok := params["binary"].([]byte); ok {
		return bytes.NewBuffer(binary)
	}
	return bytes.NewBuffer(nil)
}
