package z

// RemoveAtIndex 删除切片指定位置的元素
func RemoveAtIndex(slice []interface{}, index int) []interface{} {
	if index < 0 || index >= len(slice) {
		return slice
	}
	return append(slice[:index], slice[index+1:]...)
}
