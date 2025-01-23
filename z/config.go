package z

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// configs 存储所有配置文件
var configs map[string]*viper.Viper

// config 结构体
type config struct {
	envPrefix string
}

// Config 全局配置对象
var Config config

// LoadDir 加载指定目录下的所有配置文件
func (c *config) LoadDir(dir string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() {
			filename := file.Name()
			if err := c.LoadFile(dir, filename); err != nil {
				return err
			}
		}
	}

	return nil
}

// LoadFile 加载指定文件
func (c *config) LoadFile(dir string, filename string) error {

	c.envPrefix = c.GetEnvPrefix()

	if ".yml" != filepath.Ext(filename) {
		return nil
	}

	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	cfg := viper.New()
	cfg.SetConfigType("yaml")
	cfg.SetConfigName(name)
	cfg.AddConfigPath(dir)

	if err := cfg.ReadInConfig(); err != nil {
		return errors.New("error on parsing configuration file: " + err.Error())
	}

	if configs == nil {
		configs = make(map[string]*viper.Viper)
	}

	configs[name] = cfg

	return nil
}

// SetEnvs 将配置信息写入环境变量
func (c *config) SetEnvs(configs map[string]interface{}) error {
	for key, value := range configs {
		envKey := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
		envKey = fmt.Sprintf("%s_%s", c.envPrefix, envKey)
		if err := os.Setenv(envKey, fmt.Sprintf("%v", value)); err != nil {
			return err
		}
	}

	return nil
}

// GetEnvPrefix 获取 env 前缀
func (c *config) GetEnvPrefix() string {
	name, err := c.String("config.name")
	if err != nil {
		name = "ICREATEAPP"
	} else {
		name = strings.ToUpper(name)
	}

	return name
}

// parseName 解析配置文件名和配置项名
func (c *config) parseName(name string) (v *viper.Viper, valueName string, err error) {
	names := strings.Split(name, ".")

	if len(names) < 2 {
		return nil, "", errors.New("invalid configuration name")
	}

	_fileName := names[0]
	_valueName := strings.Join(names[1:], ".")

	// 优先从环境变量中读取
	envVarName := strings.ToUpper(strings.ReplaceAll(name, ".", "_"))
	envVarName = fmt.Sprintf("%s_%s", c.envPrefix, envVarName)
	if envValue, ok := os.LookupEnv(envVarName); ok {
		tempViper := viper.New()
		tempViper.Set(_valueName, envValue)
		return tempViper, _valueName, nil
	}

	if v := configs[_fileName]; v != nil {
		return v, _valueName, nil
	}

	return nil, "", errors.New("invalid configuration name")
}

// String 获取字符串类型的配置项
func (c *config) String(name string) (value string, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return "", err
	}

	return vv.GetString(vn), nil
}

// Bool 获取布尔类型的配置项
func (c *config) Bool(name string) (value bool, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return false, err
	}

	return vv.GetBool(vn), nil
}

// Int 获取整数类型的配置项
func (c *config) Int(name string) (value int, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetInt(vn), nil
}

// Int32 获取32位整数类型的配置项
func (c *config) Int32(name string) (value int32, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetInt32(vn), nil
}

// Int64 获取64位整数类型的配置项
func (c *config) Int64(name string) (value int64, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetInt64(vn), nil
}

// Uint 获取无符号整数类型的配置项
func (c *config) Uint(name string) (value uint, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetUint(vn), nil
}

// Uint16 获取16位无符号整数类型的配置项
func (c *config) Uint16(name string) (value uint16, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetUint16(vn), nil
}

// Uint32 获取32位无符号整数类型的配置项
func (c *config) Uint32(name string) (value uint32, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetUint32(vn), nil
}

// Uint64 获取64位无符号整数类型的配置项
func (c *config) Uint64(name string) (value uint64, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetUint64(vn), nil
}

// Float64 获取浮点数类型的配置项
func (c *config) Float64(name string) (value float64, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetFloat64(vn), nil
}

// Time 获取时间类型的配置项
func (c *config) Time(name string) (value time.Time, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return time.Now(), err
	}

	return vv.GetTime(vn), nil
}

// Duration 获取时间间隔类型的配置项
func (c *config) Duration(name string) (value time.Duration, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetDuration(vn), nil
}

// IntSlice 获取整数切片类型的配置项
func (c *config) IntSlice(name string) (value []int, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return nil, err
	}

	return vv.GetIntSlice(vn), nil
}

// StringSlice 获取字符串切片类型的配置项
func (c *config) StringSlice(name string) (value []string, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return nil, err
	}

	return vv.GetStringSlice(vn), nil
}

// StringMap 获取字符串映射类型的配置项
func (c *config) StringMap(name string) (value map[string]interface{}, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return nil, err
	}

	return vv.GetStringMap(vn), nil
}

// StringMapStringSlice 获取字符串映射切片类型的配置项
func (c *config) StringMapStringSlice(name string) (value map[string][]string, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return nil, err
	}

	return vv.GetStringMapStringSlice(vn), nil
}

// SizeInBytes 获取字节大小的配置项
func (c *config) SizeInBytes(name string) (value uint, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetSizeInBytes(vn), nil
}
