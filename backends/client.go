package backends

import (
	"fmt"
	"strings"

	"github.com/Risingtao/nacos-confd/backends/nacos"
	"github.com/Risingtao/nacos-confd/log"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
)

type StoreClient interface {
	GetValues(keys []string) (map[string]string, error)
	WatchPrefix(prefix string, keys []string, waitIndex uint64, stopChan chan bool) (uint64, error)
}

func New(config Config) (StoreClient, error) {
	source := config.Endpoint
	if config.Backend != "nacos" || len(config.Endpoint) == 0 {
		source = strings.Join(config.BackendNodes, ", ")
	}
	log.Info(fmt.Sprintf("Backend source(s) set to %s", source))

	switch config.Backend {
	case "nacos":
		return nacos.NewNacosClient(config.BackendNodes, config.Group, constant.ClientConfig{
			NamespaceId: config.Namespace,
			AccessKey:   config.AccessKey,
			SecretKey:   config.SecretKey,
			Endpoint:    config.Endpoint,
			OpenKMS:     config.OpenKMS,
			RegionId:    config.RegionId,
		})
	default:
		return nil, fmt.Errorf("Invalid backend: %s", config.Backend)
	}
}
