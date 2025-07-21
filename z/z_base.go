package z

import (
	"os"
	"path"
)

// BasePath 返回项目目录绝对路径
func BasePath(paths ...string) string {
	if dir, err := os.Getwd(); err != nil {
		panic("Failed to get current working directory: " + err.Error())
	} else {
		return path.Join(append([]string{dir}, paths...)...)
	}
}

// StoragePath 返回存储目录绝对路径
func StoragePath(paths ...string) string {
	return path.Join(append([]string{BasePath(), "storage"}, paths...)...)
}

// TmpPath 返回临时目录绝对路径
func TmpPath(paths ...string) string {
	return path.Join(append([]string{BasePath(), "storage", "tmp"}, paths...)...)
}

// CachePath 返回缓存绝对路径
func CachePath(paths ...string) string {
	return path.Join(append([]string{BasePath(), "storage", "cache"}, paths...)...)
}

// LogPath 返回日志绝对路径
func LogPath(paths ...string) string {
	return path.Join(append([]string{BasePath(), "storage", "log"}, paths...)...)
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
