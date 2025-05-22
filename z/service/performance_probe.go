package service

import (
	"errors"
	"runtime"
	"sync"
	"time"
)

// PointExecutionInfo 记录节点的执行时间和内存占用
type PointExecutionInfo struct {
	PointID  string    `json:"point_id"`
	Time     time.Time `json:"time"`     // 执行开始时间（精确到微秒）
	Duration uint64    `json:"duration"` // 执行耗时（微秒）
	Memory   uint64    `json:"memory"`   // 内存占用（字节）
}

// PerformanceProbe 记录节点性能数据的对象
type PerformanceProbe struct {
	pointExecutionInfos     []PointExecutionInfo
	pointExecutionInfosLock sync.Mutex
}

// NewPerformanceProbe 创建一个新的 PerformanceProbe 对象
func NewPerformanceProbe() *PerformanceProbe {
	return &PerformanceProbe{
		pointExecutionInfos:     make([]PointExecutionInfo, 0),
		pointExecutionInfosLock: sync.Mutex{},
	}
}

// RecordPerformance 记录节点的执行时间和内存占用
func (pp *PerformanceProbe) RecordPerformance(pointId string, startTime time.Time, duration uint64, memoryAlloc uint64) {
	info := PointExecutionInfo{
		PointID:  pointId,
		Time:     startTime,
		Duration: duration,
		Memory:   memoryAlloc,
	}

	// 异步记录性能数据
	go func() {
		pp.pointExecutionInfosLock.Lock()
		defer pp.pointExecutionInfosLock.Unlock()
		pp.pointExecutionInfos = append(pp.pointExecutionInfos, info)
	}()
}

// GetPerformanceData 获取所有节点的性能数据
func (pp *PerformanceProbe) GetPerformanceData() []PointExecutionInfo {
	pp.pointExecutionInfosLock.Lock()
	defer pp.pointExecutionInfosLock.Unlock()
	return pp.pointExecutionInfos
}

// MeasureFunction 执行函数并记录其性能数据
func (pp *PerformanceProbe) MeasureFunction(pointId string, fn func()) {
	var memStatsBefore, memStatsAfter runtime.MemStats

	// 记录开始时间
	startTime := time.Now()

	// 记录开始时的内存使用情况
	runtime.ReadMemStats(&memStatsBefore)

	// 执行函数
	fn()

	// 记录结束时的内存使用情况
	runtime.ReadMemStats(&memStatsAfter)

	// 计算执行时间（微秒）
	duration := time.Since(startTime).Microseconds()

	// 计算内存占用（字节）
	memoryAlloc := memStatsAfter.Alloc - memStatsBefore.Alloc

	// 记录性能数据
	pp.RecordPerformance(pointId, startTime, uint64(duration), memoryAlloc)
}

// FindPerformanceData 根据 pointId 查找性能数据
func (pp *PerformanceProbe) FindPerformanceData(pointId string) (PointExecutionInfo, error) {
	pp.pointExecutionInfosLock.Lock()
	defer pp.pointExecutionInfosLock.Unlock()

	for _, info := range pp.pointExecutionInfos {
		if info.PointID == pointId {
			return info, nil
		}
	}

	return PointExecutionInfo{}, errors.New("performance data not found for point ID: " + pointId)
}
