package zLib

import (
	"os"
	"path"
)

// BasePath 返回项目目录绝对路径
func BasePath() (string, error) {
	return os.Getwd()
}

// StorePath 返回存储目录绝对路径
func StorePath(paths ...string) string {
	base, _ := BasePath()
	return path.Join(append([]string{base, "storage"}, paths...)...)
}

// TmpPath 返回临时目录绝对路径
func TmpPath(paths ...string) string {
	base, _ := BasePath()
	return path.Join(append([]string{base, "storage", "tmp"}, paths...)...)
}

// CachePath 返回缓存绝对路径
func CachePath(paths ...string) string {
	base, _ := BasePath()
	return path.Join(append([]string{base, "storage", "cache"}, paths...)...)
}

// LogPath 返回日志绝对路径
func LogPath(paths ...string) string {
	base, _ := BasePath()
	return path.Join(append([]string{base, "storage", "log"}, paths...)...)
}

// IsExists 文件或目录是否存在
func IsExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
