package zLib

import (
	cron "github.com/robfig/cron/v3"
)

// _cron 结构体，用于封装cron.Cron
type _cron struct {
	cronIns *cron.Cron
}

// Cron 全局变量，用于调用_cron结构体
var Cron _cron

// Init 初始化
func (p *_cron) Init(opts ...cron.Option) {
	p.cronIns = cron.New(opts...)
}

// Add 增加任务
func (p *_cron) Add(spec string, cmd func()) (cron.EntryID, error) {
	// 添加定时任务
	return p.cronIns.AddFunc(spec, cmd)
}

// Remove 删除任务
func (p *_cron) Remove(id cron.EntryID) {
	p.cronIns.Remove(id)
}

// Start 后台运行
func (p *_cron) Start() {
	p.cronIns.Start()
}

// Run 前台运行
func (p *_cron) Run() {
	p.cronIns.Run()
}
