package z

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// GetUrl 生成当前服务器的 URL 地址
func GetUrl(params string) string {
	urlCfg, _ := Config.String("_config.url")

	return fmt.Sprintf("%s/%s", strings.Trim(urlCfg, "/"), strings.Trim(params, "/"))
}

// Post 发起 POST 请求
func Post(url string, data map[string]interface{}) (string, error) {
	values := ToValues(data)
	res, err := http.Post(
		url,
		"application/x-www-form-urlencoded",
		bytes.NewBufferString(values.Encode()),
	)
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
func Get(url string) (string, error) {
	res, err := http.Get(url)
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
