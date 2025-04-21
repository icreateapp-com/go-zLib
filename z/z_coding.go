package z

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net/url"
	"regexp"
	"strings"
)

// GetSha1 SHA1 编码
func GetSha1(str string) string {
	hash := sha1.New()
	hash.Write([]byte(str))
	return hex.EncodeToString(hash.Sum(nil))
}

// GetMd5 MD5 编码
func GetMd5(str string) string {
	h := md5.New()
	_, err := io.WriteString(h, str)
	if err != nil {
		log.Println(err)
		return ""
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// EncodeJson JSON 编码
func EncodeJson(v any) (string, error) {
	res, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	return string(res), nil
}

// DecodeJson JSON 解码
func DecodeJson(str string) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(str), &result); err != nil {
		return result, err
	}

	return result, nil
}

// EncodeYaml YAML 编码
func EncodeYaml(v any) (string, error) {
	res, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}

	return string(res), nil
}

// DecodeYaml YAML 解码
func DecodeYaml(str string) (interface{}, error) {
	var result interface{}
	if err := yaml.Unmarshal([]byte(str), &result); err != nil {
		return result, err
	}

	return result, nil
}

// EncodeUrl URL 编码
func EncodeUrl(v string) (string, error) {
	return url.QueryEscape(v), nil
}

// DecodeUrl URL 解码
func DecodeUrl(v string) (interface{}, error) {
	return url.QueryUnescape(v)
}

// DecodeJsonValue JSON 解码并返回其中一个值
func DecodeJsonValue(str string, key string) (string, error) {
	res, err := DecodeJson(str)
	if err != nil {
		return str, err
	}

	if r := res[key]; r != nil {
		return ToString(res[key]), nil
	}

	return str, nil
}

// EncodeBase64 Base64 编码
func EncodeBase64(str string) (string, error) {

	return base64.StdEncoding.EncodeToString([]byte(str)), nil
}

// DecodeBase64 Base64 解码
func DecodeBase64(str string) (string, error) {
	decodeString, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}
	return string(decodeString), nil
}

// IsMd5 判断字符串是否是MD5格式
func IsMd5(s string) bool {
	match, _ := regexp.MatchString("^[a-f0-9]{32}$", s)
	return match
}

// IsCron 判断字符串是否是Cron格式
func IsCron(cronStr string) bool {
	parts := strings.Fields(cronStr)
	if len(parts) != 5 && len(parts) != 6 {
		return false
	}

	// Define regex patterns for each part of the cron expression
	patterns := []string{
		`^(?:\d{1,2}|[*]|(?:\d{1,2}\/\d{1,2})|(?:\d{1,2}\-\d{1,2}))$`, // seconds (optional)
		`^(?:\d{1,2}|[*]|(?:\d{1,2}\/\d{1,2})|(?:\d{1,2}\-\d{1,2}))$`, // minutes
		`^(?:\d{1,2}|[*]|(?:\d{1,2}\/\d{1,2})|(?:\d{1,2}\-\d{1,2}))$`, // hours
		`^(?:\d{1,2}|[*]|(?:\d{1,2}\/\d{1,2})|(?:\d{1,2}\-\d{1,2}))$`, // day of month
		`^(?:\d{1,2}|[*]|(?:\d{1,2}\/\d{1,2})|(?:\d{1,2}\-\d{1,2}))$`, // month
		`^(?:\d{1}|[*]|(?:\d{1}\/\d{1})|(?:\d{1}\-\d{1}))$`,           // day of week (0-7 or * or ?)
	}

	for i, part := range parts {
		matched, _ := regexp.MatchString(patterns[i%6], part)
		if !matched {
			return false
		}
	}

	return true
}

// IsBase64Image 判断字符串是否是base64图片
func IsBase64Image(toTest string) bool {
	if strings.HasPrefix(toTest, "data:image/png;base64") {
		return true
	} else if strings.HasPrefix(toTest, "data:image/jpg;base64") {
		return true
	} else if strings.HasPrefix(toTest, "data:image/jpeg;base64") {
		return true
	} else if strings.HasPrefix(toTest, "data:image/gif;base64") {
		return true
	}

	return false
}

// IsUUID 判断字符串是否是UUID格式
func IsUUID(uuid string) bool {
	re := regexp.MustCompile(`^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}|[0-9a-fA-F]{32})$`)
	return re.MatchString(uuid)
}
