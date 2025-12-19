package z

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/google/uuid"
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

// GetUUID 获取UUID
func GetUUID() string {
	return uuid.New().String()
}

// DecryptRSAOAEP 使用RSA私钥对使用OAEP填充和SHA-256哈希的数据进行解密
func DecryptRSAOAEP(privateKey *rsa.PrivateKey, ciphertext string) (string, error) {
	// 对密文进行Base64解码
	decodedCipher, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 ciphertext: %v", err)
	}

	// 使用RSA-OAEP和SHA-256进行解密
	decrypted, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, decodedCipher, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt with RSA-OAEP: %v", err)
	}

	return string(decrypted), nil
}

// DecryptRSAOAEPByBase64 使用Base64编码的RSA私钥对使用OAEP填充和SHA-256哈希的数据进行解密
func DecryptRSAOAEPByBase64(base64PrivateKey string, ciphertext string) (string, error) {
	// 对Base64编码的私钥进行解码
	privateKeyBytes, err := base64.StdEncoding.DecodeString(base64PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 private key: %v", err)
	}

	// 解析PEM格式的私钥
	block, _ := pem.Decode(privateKeyBytes)
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block containing private key")
	}

	// 解析PKCS8格式的私钥
	pkcs8PrivateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// 如果不是PKCS8格式，尝试使用PKCS1格式
		pkcs1PrivateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("failed to parse private key: %v", err)
		}
		return DecryptRSAOAEP(pkcs1PrivateKey, ciphertext)
	}

	// 类型断言获取RSA私钥
	rsaPrivateKey, ok := pkcs8PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("private key is not an RSA private key")
	}

	return DecryptRSAOAEP(rsaPrivateKey, ciphertext)
}

// Encrypt 加密
func Encrypt(plainText, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 分配足够的空间
	blockSize := block.BlockSize()
	plainText = pkcs7Padding(plainText, blockSize)

	cipherText := make([]byte, len(plainText))
	iv := make([]byte, blockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}

	stream := cipher.NewCBCEncrypter(block, iv)
	stream.CryptBlocks(cipherText, plainText)

	cipherText = append(iv, cipherText...) // 在加密文本前加入IV
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// Decrypt 解密
func Decrypt(cipherTextBase64 string, key []byte) ([]byte, error) {
	cipherText, err := base64.StdEncoding.DecodeString(cipherTextBase64)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	if len(cipherText) < blockSize {
		return nil, errors.New("cipherText too short")
	}

	iv := cipherText[:blockSize]
	cipherText = cipherText[blockSize:]

	stream := cipher.NewCBCDecrypter(block, iv)
	stream.CryptBlocks(cipherText, cipherText)

	cipherText = pkcs7UnPadding(cipherText)
	return cipherText, nil
}

// PKCS7 填充
func pkcs7Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padText...)
}

// PKCS7 去填充
func pkcs7UnPadding(src []byte) []byte {
	length := len(src)
	unPadding := int(src[length-1])
	return src[:(length - unPadding)]
}
