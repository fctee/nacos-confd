package template

import (
	"fmt"
	"sync"
	"time"

	"github.com/Risingtao/nacos-confd/log"
	"github.com/Risingtao/nacos-confd/util"
)

// Processor 定义一个处理接口，所有处理器都需要实现这个接口
type Processor interface {
	Process()
}

// Process 函数用于处理配置
func Process(config Config) error {
	// 获取所有模板资源
	ts, err := getTemplateResources(config)
	if err != nil {
		return fmt.Errorf("获取模板资源出错: %w", err)
	}
	// 开始处理模板资源
	return process(ts)
}

// process 函数依次处理所有模板资源
func process(ts []*TemplateResource) error {
	var lastErr error
	for _, t := range ts {
		if err := t.process(); err != nil {
			log.Error("处理模板 %s 出错: %v", t.Src, err)
			lastErr = fmt.Errorf("模板 %s - 处理出错: %w", t.Src, err)
		}
	}
	return lastErr
}

// intervalProcessor 定时处理器结构体
type intervalProcessor struct {
	config   Config
	stopChan chan bool
	doneChan chan bool
	errChan  chan error
	interval int
}

// IntervalProcessor 构造函数，返回一个新的定时处理器
// 参数:
//   - config: 配置信息
//   - stopChan: 停止信号
//   - doneChan: 完成信号
//   - errChan: 错误信号
//   - interval: 执行间隔时间（秒）
// 返回值:
//   - Processor: 返回一个 Processor 接口的实现
func IntervalProcessor(config Config, stopChan, doneChan chan bool, errChan chan error, interval int) Processor {
	return &intervalProcessor{config, stopChan, doneChan, errChan, interval}
}

// Process 定时处理模板资源的方法
func (p *intervalProcessor) Process() {
	defer close(p.doneChan)
	for {
		ts, err := getTemplateResources(p.config)
		if err != nil {
			log.Fatal(err.Error())
			break
		}
		process(ts)
		select {
		case <-p.stopChan:
			break
		case <-time.After(time.Duration(p.interval) * time.Second):
			continue
		}
	}
}

// watchProcessor 监控处理器结构体
type watchProcessor struct {
	config   Config
	stopChan chan bool
	doneChan chan bool
	errChan  chan error
	wg       sync.WaitGroup
}

// WatchProcessor 构造函数，返回一个新的监控处理器
// 参数:
//   - config: 配置信息
//   - stopChan: 停止信号
//   - doneChan: 完成信号
//   - errChan: 错误信号
// 返回值:
//   - Processor: 返回一个 Processor 接口的实现
func WatchProcessor(config Config, stopChan, doneChan chan bool, errChan chan error) Processor {
	var wg sync.WaitGroup
	return &watchProcessor{config, stopChan, doneChan, errChan, wg}
}

// Process 监控处理模板资源的方法
func (p *watchProcessor) Process() {
	defer close(p.doneChan)
	ts, err := getTemplateResources(p.config)
	if err != nil {
		log.Fatal("获取模板资源出错: %v", err)
		return
	}
	for _, t := range ts {
		t := t
		p.wg.Add(1)
		go p.monitorPrefix(t)
	}
	p.wg.Wait()
}

// monitorPrefix 监控某一模板资源的方法
// 参数:
//   - t: 模板资源
func (p *watchProcessor) monitorPrefix(t *TemplateResource) {
	defer p.wg.Done()
	keys := util.AppendPrefix(t.Prefix, t.Keys)
	for {
		index, err := t.storeClient.WatchPrefix(t.Prefix, keys, t.lastIndex, p.stopChan)
		if err != nil {
			p.errChan <- err
			time.Sleep(time.Second * 2)
			continue
		}
		t.lastIndex = index
		if err := t.process(); err != nil {
			p.errChan <- err
		}
	}
}

// getTemplateResources 获取模板资源
// 参数:
//   - config: 配置信息
// 返回值:
//   - []*TemplateResource: 模板资源列表
//   - error: 是否有错误发生
func getTemplateResources(config Config) ([]*TemplateResource, error) {
	var lastError error
	templates := make([]*TemplateResource, 0)
	log.Debug("从 %s 加载模板", config.ConfDir)

	if !util.IsFileExist(config.ConfDir) {
		log.Warning(fmt.Sprintf("无法加载模板资源：配置目录 '%s' 不存在", config.ConfDir))
		return nil, nil
	}

	paths, err := util.RecursiveFilesLookup(config.ConfigDir, "*toml")
	if err != nil {
		return nil, fmt.Errorf("查找文件出错: %w", err)
	}

	if len(paths) < 1 {
		log.Warning("未找到任何模板")
	}

	for _, p := range paths {
		log.Debug(fmt.Sprintf("找到模板: %s", p))

		t, err := NewTemplateResource(p, config)

		if err != nil {
			lastError = fmt.Errorf("为 %s 创建模板资源出错: %w", p, err)
			continue
		}
		templates = append(templates, t)
	}

	return templates, lastError
}