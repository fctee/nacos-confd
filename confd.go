// 导入必要的包
package main

import (
	"flag" // 用于解析命令行参数
	"fmt" // 用于格式化输出
	"os"   // 提供操作系统相关功能，如文件操作、退出程序等
	"os/signal" // 用于接收操作系统信号
	"runtime" // 提供运行时信息，如Go版本
	"syscall" // 提供对底层系统调用的封装

	// 导入自定义包
	"github.com/Risingtao/nacos-confd/backends" // 后端存储客户端
	"github.com/Risingtao/nacos-confd/log"     // 日志处理
	"github.com/Risingtao/nacos-confd/resource/template" // 模板处理
)

// main函数是程序的入口点
func main() {
	// 解析命令行参数
	flag.Parse()
	// 如果配置中要求打印版本信息，则打印并退出
	if config.PrintVersion {
		fmt.Printf("confd %s (Author: %s, Git SHA: %s, Go Version: %s)\n", Build_time, Author, GitSHA, runtime.Version())
		os.Exit(0)
	}

	// 初始化配置，如果出错则记录错误并退出程序
	if err := initConfig(); err != nil {
		log.Fatal("初始化配置时出错: %v", err)
	}

	// 启动confd，记录日志信息
	log.Info("Starting confd")

	// 初始化后端存储客户端，如果出错则记录错误并退出程序
	storeClient, err := backends.New(config.BackendsConfig)
	if err != nil {
		log.Fatal("创建后端存储客户端时出错: %v", err)
	}

	// 处理模板配置，将后端存储客户端传递给模板配置
	config.TemplateConfig.StoreClient = storeClient
	// 如果配置中要求只处理一次，则处理模板配置并退出程序
	if config.OneTime {
		if err := template.Process(config.TemplateConfig); err != nil {
			log.Fatal("处理模板配置时出错: %v", err)
		}
		os.Exit(0)
	}

	// 创建控制通道，用于停止处理器和接收完成信号
	stopChan := make(chan bool)
	doneChan := make(chan bool)
	errChan := make(chan error, 10) // 用于接收处理器错误

	// 根据配置选择并创建处理器
	var processor template.Processor
	switch {
	case config.Watch: // 如果配置中要求监听文件变化
		processor = template.WatchProcessor(config.TemplateConfig, stopChan, doneChan, errChan)
	default: // 否则使用定时处理
		processor = template.IntervalProcessor(config.TemplateConfig, stopChan, doneChan, errChan, config.Interval)
	}

	// 启动处理器，使用goroutine异步执行
	go processor.Process()

	// 设置信号处理循环，接收操作系统信号
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM) // 监听SIGINT和SIGTERM信号
	for {
		select {
		case err := <-errChan: // 如果接收到处理器错误，则记录错误
			log.Error("处理器出错: %v", err)
		case s := <-signalChan: // 如果接收到操作系统信号，则准备退出程序
			log.Info(fmt.Sprintf("捕获到信号 %v,准备退出...", s))
			close(doneChan) // 关闭完成通道，通知处理器退出
		case <-doneChan: // 如果接收到完成信号，则退出程序
			os.Exit(0)
		}
	}
}
