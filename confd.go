package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/Risingtao/nacos-confd/backends"
	"github.com/Risingtao/nacos-confd/log"
	"github.com/Risingtao/nacos-confd/resource/template"
)

func main() {
	//解析命令行参数
	flag.Parse()
	if config.PrintVersion {
		fmt.Printf("confd %s (Author: %s, Git SHA: %s, Go Version: %s)\n", Build_time, Author, GitSHA, runtime.Version())
		os.Exit(0)
	}

	//初始化配置
	if err := initConfig(); err != nil {
		log.Fatal("初始化配置时出错: %v", err)
	}

	//启动 confd
	log.Info("Starting confd")

	//初始化后端存储客户端
	storeClient, err := backends.New(config.BackendsConfig)
	if err != nil {
		log.Fatal("创建后端存储客户端时出错: %v", err)
	}

	//处理模板配置
	config.TemplateConfig.StoreClient = storeClient
	if config.OneTime {
		if err := template.Process(config.TemplateConfig); err != nil {
			log.Fatal("处理模板配置时出错: %v", err)
		}
		os.Exit(0)
	}

	//创建控制通道
	stopChan := make(chan bool)
	doneChan := make(chan bool)
	errChan := make(chan error, 10)

	//选择并创建处理器
	var processor template.Processor
	switch {
	case config.Watch:
		processor = template.WatchProcessor(config.TemplateConfig, stopChan, doneChan, errChan)
	default:
		processor = template.IntervalProcessor(config.TemplateConfig, stopChan, doneChan, errChan, config.Interval)
	}

	//启动处理器
	go processor.Process()

	//信号处理循环
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case err := <-errChan:
			log.Error("处理器出错: %v", err)
		case s := <-signalChan:
			log.Info(fmt.Sprintf("捕获到信号 %v,准备退出...", s))
			close(doneChan)
		case <-doneChan:
			os.Exit(0)
		}
	}
}
