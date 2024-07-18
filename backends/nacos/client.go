package nacos

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Risingtao/nacos-confd/log"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// Replacer 用于处理键格式
var replacer = strings.NewReplacer("/", ".")

// Client 结构体，包含配置客户端和命名客户端
type Client struct {
	configClient config_client.IConfigClient
	namingClient naming_client.INamingClient
	group        string
	namespace    string
	accessKey    string
	secretKey    string
	channel      chan int
	count        int
}

// NewNacosClient 初始化 Nacos 客户端
func NewNacosClient(nodes []string, group string, config constant.ClientConfig) (*Client, error) {
	var servers []constant.ServerConfig

	// 解析节点信息并生成服务器配置
	for _, key := range nodes {
		nacosUrl, err := url.Parse(key)
		if err != nil {
			log.Error(fmt.Sprintf("解析 URL 失败: %s, 错误: %v", key, err))
			return nil, err
		}
		port, err := strconv.Atoi(nacosUrl.Port())
		if err != nil {
			log.Error(fmt.Sprintf("转换端口失败: %s, 错误: %v", nacosUrl.Port(), err))
			return nil, err
		}
		servers = append(servers, constant.ServerConfig{
			IpAddr: nacosUrl.Hostname(),
			Port:   uint64(port),
		})
	}

	// 如果组名为空，设置为默认组
	if strings.TrimSpace(group) == "" {
		group = "DEFAULT_GROUP"
	}

	log.Info("endpoint=" + config.Endpoint + ", namespace=" + config.NamespaceId + ", group=" + group +
         ", accessKey=" + config.AccessKey + ", secretKey=" + config.SecretKey +
         ", openKMS=" + fmt.Sprint(config.OpenKMS) + ", regionId=" + config.RegionId)

	// 使用配置参数创建 ClientConfig
	clientConfig := *constant.NewClientConfig(
		constant.WithNamespaceId(config.NamespaceId),
		constant.WithTimeoutMs(30000),
		constant.WithBeatInterval(10000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithUpdateCacheWhenEmpty(false),
		constant.WithLogDir("/etc/confd/log"),
		constant.WithCacheDir("/etc/confd/cache"),
		constant.WithLogLevel("info"),
	)

	// 创建配置客户端
	configClient, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: servers,
		},
	)

	if err != nil {
		log.Error(fmt.Sprintf("创建配置客户端失败: %v", err))
		return nil, err
	}

	// 创建命名客户端
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: servers,
		},
	)

	if err != nil {
		log.Error(fmt.Sprintf("创建命名客户端失败: %v", err))
		return nil, err
	}

	client := &Client{
		configClient: configClient,
		namingClient: namingClient,
		group:        group,
		namespace:    config.NamespaceId,
		accessKey:    config.AccessKey,
		secretKey:    config.SecretKey,
		channel:      make(chan int, 10),
		count:        0,
	}

	return client, nil
}

// GetValues 获取指定键的值
func (client *Client) GetValues(keys []string) (map[string]string, error) {
	vars := make(map[string]string)
	for _, key := range keys {
		k := strings.TrimPrefix(key, "/")
		k = replacer.Replace(k)

		// 如果键以 "naming." 开头，则获取服务实例
		if strings.HasPrefix(k, "naming.") {
			instances, err := client.namingClient.SelectAllInstances(vo.SelectAllInstancesParam{
				ServiceName: k,
				GroupName:   client.group,
			})
			if err != nil {
				log.Error(fmt.Sprintf("获取实例失败,key: %s, 错误: %v", key, err))
				return nil, err
			}
			vars[key] = util.ToJsonString(instances)
		} else {
			// 否则获取配置
			resp, err := client.configClient.GetConfig(vo.ConfigParam{
				DataId: k,
				Group:  client.group,
			})
			if err != nil {
				log.Error(fmt.Sprintf("获取配置失败,key: %s, 错误: %v", key, err))
				return nil, err
			}
			vars[key] = resp
		}
	}
	return vars, nil
}

// WatchPrefix 订阅服务和监听配置
func (client *Client) WatchPrefix(prefix string, keys []string, waitIndex uint64, stopChan chan bool) (uint64, error) {
	if waitIndex == 0 {
		client.count++
		for _, key := range keys {
			k := strings.TrimPrefix(key, "/")
			k = replacer.Replace(k)

			// 如果键以 "naming." 开头，则订阅服务
			if strings.HasPrefix(k, "naming.") {
				err := client.namingClient.Subscribe(&vo.SubscribeParam{
					ServiceName: k,
					GroupName:   client.group,
					SubscribeCallback: func(services []model.Instance, err error) {
						if err != nil {
							log.Error(fmt.Sprintf("订阅服务失败: %v\n", err))
							return
						}

						log.Info(fmt.Sprintf("订阅回调 - 服务实例: %s", util.ToJsonString(services)))

						for i := 0; i < client.count; i++ {
							client.channel <- 1
						}
					},
				})
				if err != nil {
					log.Error(fmt.Sprintf("订阅服务失败: %s, 错误: %v", k, err))
					return 0, err
				}
			} else {
				// 否则监听配置
				err := client.configClient.ListenConfig(vo.ConfigParam{
					DataId: k,
					Group:  client.group,
					OnChange: func(namespace, group, dataId, data string) {

						log.Info(fmt.Sprintf("配置变更: namespace: %s, dataId: %s, group: %s", namespace, dataId, group))

						for i := 0; i < client.count; i++ {
							client.channel <- 1
						}
					},
				})
				if err != nil {
					log.Error(fmt.Sprintf("监听配置失败: %s, 错误: %v", k, err))
					return 0, err
				}
			}
		}
		return 1, nil
	}

	select {
	case <-client.channel:
		// 通道有消息时的日志
		return waitIndex, nil
	case <-stopChan:
		log.Info("收到停止信号，停止监听。")
		return waitIndex, nil
	}
}
