// 导入需要的包
import (
	"fmt" // 用于格式化输出
	"strings" // 用于处理字符串

	// 导入nacos相关包
	"github.com/Risingtao/nacos-confd/backends/nacos"
	"github.com/Risingtao/nacos-confd/log"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
)

// 定义StoreClient接口，包含获取值和监听前缀的方法
type StoreClient interface {
	GetValues(keys []string) (map[string]string, error) // 获取指定键的值
	WatchPrefix(prefix string, keys []string, waitIndex uint64, stopChan chan bool) (uint64, error) // 监听指定前缀的键值变化
}

// New函数用于创建一个新的StoreClient实例
func New(config Config) (StoreClient, error) {
	// 根据配置确定后端来源
	source := config.Endpoint
	if config.Backend != "nacos" || len(config.Endpoint) == 0 {
		source = strings.Join(config.BackendNodes, ", ")
	}
	log.Info(fmt.Sprintf("Backend source(s) set to %s", source)) // 记录后端来源信息

	// 根据配置的后端类型创建相应的客户端
	switch config.Backend {
	case "nacos": // 如果后端是nacos
		// 创建nacos客户端，传入配置参数
		return nacos.NewNacosClient(config.BackendNodes, config.Group, constant.ClientConfig{
			NamespaceId: config.Namespace, // 命名空间ID
			AccessKey:   config.AccessKey, // 访问密钥
			SecretKey:   config.SecretKey, // 密钥
			Endpoint:    config.Endpoint, // 端点
			OpenKMS:     config.OpenKMS, // 是否开启KMS
			RegionId:    config.RegionId, // 区域ID
		})
	default: // 如果不是nacos或其他未识别的后端
		return nil, fmt.Errorf("Invalid backend: %s", config.Backend) // 返回错误信息
	}
}
