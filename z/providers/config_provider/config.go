package config_provider

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// Config 配置管理
type Config struct {
	path    string
	configs map[string]*viper.Viper
	isDir   bool
}

type Options struct {
	Path string
}

func ConfigOptions(path string) Options {
	return Options{Path: path}
}

// NewConfigProvider 创建配置管理实例
func NewConfigProvider(opts Options) (*Config, error) {
	c := &Config{
		path:    opts.Path,
		configs: map[string]*viper.Viper{},
	}

	return c.init()
}

func (c *Config) init() (*Config, error) {
	info, err := os.Stat(c.path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		c.isDir = true
		if err := c.LoadDir(c.path); err != nil {
			return nil, err
		}
	} else {
		c.isDir = false
		dir := filepath.Dir(c.path)
		filename := filepath.Base(c.path)
		if filename == "." || filename == string(filepath.Separator) {
			return nil, errors.New("invalid config path")
		}
		if err := c.LoadFile(dir, filename, ""); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// ConfigModule 配置管理模块
var ConfigModule = fx.Options(
	fx.Provide(NewConfigProvider),
)

// LoadDir 加载指定目录下的所有配置文件
func (c *Config) LoadDir(dir string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	appYml := false
	appYaml := false
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		switch strings.ToLower(file.Name()) {
		case "app.yml":
			appYml = true
		case "app.yaml":
			appYaml = true
		}
	}
	if appYml && appYaml {
		return errors.New("both app.yml and app.yaml exist")
	}
	if !appYml && !appYaml {
		return errors.New("config dir must contain app.yml or app.yaml")
	}

	for _, file := range files {
		if !file.IsDir() {
			filename := file.Name()
			if err := c.LoadFile(dir, filename, ""); err != nil {
				return err
			}
		}
	}

	return nil
}

// LoadFile 加载指定文件

func (c *Config) LoadFile(dir string, filename string, namespace string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".yml" && ext != ".yaml" {
		return nil
	}

	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	registerName := namespace
	if registerName == "" {
		registerName = name
	}

	if c.configs == nil {
		c.configs = make(map[string]*viper.Viper)
	}
	if _, exists := c.configs[registerName]; exists {
		return errors.New("duplicate namespace config: " + registerName)
	}

	cfg := viper.New()
	cfg.SetConfigFile(filepath.Join(dir, filename))

	if err := cfg.ReadInConfig(); err != nil {
		return errors.New("error on parsing configuration file: " + err.Error())
	}

	c.configs[registerName] = cfg

	return nil
}

// parseName 解析配置文件名和配置项名
func (c *Config) parseName(name string) (v *viper.Viper, valueName string, err error) {
	names := strings.Split(name, ".")
	if len(names) < 2 {
		return nil, "", errors.New("invalid configuration name")
	}

	ns := names[0]
	key := strings.Join(names[1:], ".")

	if c.isDir {
		vv := c.configs[ns]
		if vv == nil {
			return nil, "", errors.New("missing namespace config: " + ns)
		}
		return vv, key, nil
	}

	if len(c.configs) != 1 {
		return nil, "", errors.New("invalid config state")
	}
	var only *viper.Viper
	for _, vv := range c.configs {
		only = vv
		break
	}
	if only == nil {
		return nil, "", errors.New("invalid config state")
	}
	if ns == "app" {
		return only, key, nil
	}
	return only, name, nil
}

// String 获取字符串类型的配置项
func (c *Config) String(name string) (value string, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return "", err
	}

	return vv.GetString(vn), nil
}

// GetString 获取字符串类型的配置项，出错时返回默认值或空字符串
func (c *Config) GetString(name string, defaultValue ...string) string {
	value, err := c.String(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}
	return value
}

// Bool 获取布尔类型的配置项
func (c *Config) Bool(name string) (value bool, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return false, err
	}

	return vv.GetBool(vn), nil
}

// GetBool 获取布尔类型的配置项，出错时返回默认值或 false
func (c *Config) GetBool(name string, defaultValue ...bool) bool {
	value, err := c.Bool(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}
	return value
}

// Int 获取整数类型的配置项
func (c *Config) Int(name string) (value int, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetInt(vn), nil
}

// GetInt 获取整数类型的配置项，出错时返回默认值或 0
func (c *Config) GetInt(name string, defaultValue ...int) int {
	value, err := c.Int(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return value
}

// Int32 获取32位整数类型的配置项
func (c *Config) Int32(name string) (value int32, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetInt32(vn), nil
}

// GetInt32 获取32位整数类型的配置项，出错时返回默认值或 0
func (c *Config) GetInt32(name string, defaultValue ...int32) int32 {
	value, err := c.Int32(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return value
}

// Int64 获取64位整数类型的配置项
func (c *Config) Int64(name string) (value int64, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetInt64(vn), nil
}

// GetInt64 获取64位整数类型的配置项，出错时返回默认值或 0
func (c *Config) GetInt64(name string, defaultValue ...int64) int64 {
	value, err := c.Int64(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return value
}

// Uint 获取无符号整数类型的配置项
func (c *Config) Uint(name string) (value uint, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetUint(vn), nil
}

// GetUint 获取无符号整数类型的配置项，出错时返回默认值或 0
func (c *Config) GetUint(name string, defaultValue ...uint) uint {
	value, err := c.Uint(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return value
}

// Uint16 获取16位无符号整数类型的配置项
func (c *Config) Uint16(name string) (value uint16, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetUint16(vn), nil
}

// GetUint16 获取16位无符号整数类型的配置项，出错时返回默认值或 0
func (c *Config) GetUint16(name string, defaultValue ...uint16) uint16 {
	value, err := c.Uint16(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return value
}

// Uint32 获取32位无符号整数类型的配置项
func (c *Config) Uint32(name string) (value uint32, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetUint32(vn), nil
}

// GetUint32 获取32位无符号整数类型的配置项，出错时返回默认值或 0
func (c *Config) GetUint32(name string, defaultValue ...uint32) uint32 {
	value, err := c.Uint32(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return value
}

// Uint64 获取64位无符号整数类型的配置项
func (c *Config) Uint64(name string) (value uint64, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetUint64(vn), nil
}

// GetUint64 获取64位无符号整数类型的配置项，出错时返回默认值或 0
func (c *Config) GetUint64(name string, defaultValue ...uint64) uint64 {
	value, err := c.Uint64(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return value
}

// Float64 获取浮点数类型的配置项
func (c *Config) Float64(name string) (value float64, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetFloat64(vn), nil
}

// GetFloat64 获取浮点数类型的配置项，出错时返回默认值或 0
func (c *Config) GetFloat64(name string, defaultValue ...float64) float64 {
	value, err := c.Float64(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return value
}

// Time 获取时间类型的配置项
func (c *Config) Time(name string) (value time.Time, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return time.Time{}, err
	}

	return vv.GetTime(vn), nil
}

// GetTime 获取时间类型的配置项，出错时返回默认值或零值时间
func (c *Config) GetTime(name string, defaultValue ...time.Time) time.Time {
	value, err := c.Time(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return time.Time{}
	}
	return value
}

// Duration 获取时间间隔类型的配置项
func (c *Config) Duration(name string) (value time.Duration, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetDuration(vn), nil
}

// GetDuration 获取时间间隔类型的配置项，出错时返回默认值或 0
func (c *Config) GetDuration(name string, defaultValue ...time.Duration) time.Duration {
	value, err := c.Duration(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return value
}

// IntSlice 获取整数切片类型的配置项
func (c *Config) IntSlice(name string) (value []int, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return nil, err
	}

	return vv.GetIntSlice(vn), nil
}

// GetIntSlice 获取整数切片类型的配置项，出错时返回默认值或 nil
func (c *Config) GetIntSlice(name string, defaultValue ...[]int) []int {
	value, err := c.IntSlice(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return nil
	}
	return value
}

// StringSlice 获取字符串切片类型的配置项
func (c *Config) StringSlice(name string) (value []string, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return nil, err
	}

	return vv.GetStringSlice(vn), nil
}

// GetStringSlice 获取字符串切片类型的配置项，出错时返回默认值或 nil
func (c *Config) GetStringSlice(name string, defaultValue ...[]string) []string {
	value, err := c.StringSlice(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return nil
	}
	return value
}

// StringMap 获取字符串映射类型的配置项
func (c *Config) StringMap(name string) (value map[string]interface{}, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return nil, err
	}

	return vv.GetStringMap(vn), nil
}

// GetStringMap 获取字符串映射类型的配置项，出错时返回默认值或 nil
func (c *Config) GetStringMap(name string, defaultValue ...map[string]interface{}) map[string]interface{} {
	value, err := c.StringMap(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return nil
	}
	return value
}

// StringMapStringSlice 获取字符串映射切片类型的配置项
func (c *Config) StringMapStringSlice(name string) (value map[string][]string, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return nil, err
	}

	return vv.GetStringMapStringSlice(vn), nil
}

// GetStringMapStringSlice 获取字符串映射切片类型的配置项，出错时返回默认值或 nil
func (c *Config) GetStringMapStringSlice(name string, defaultValue ...map[string][]string) map[string][]string {
	value, err := c.StringMapStringSlice(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return nil
	}
	return value
}

// SizeInBytes 获取字节大小的配置项
func (c *Config) SizeInBytes(name string) (value uint, err error) {
	vv, vn, err := c.parseName(name)

	if err != nil {
		return 0, err
	}

	return vv.GetSizeInBytes(vn), nil
}

// GetSizeInBytes 获取字节大小的配置项，出错时返回默认值或 0
func (c *Config) GetSizeInBytes(name string, defaultValue ...uint) uint {
	value, err := c.SizeInBytes(name)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return value
}
