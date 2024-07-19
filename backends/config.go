// 导入工具包，用于处理配置文件中的节点数据
import (
	util "github.com/Risingtao/nacos-confd/util"
)

// Config 结构体定义了Nacos配置文件中的所有可配置项
type Config struct {
	// AuthToken 用于Nacos的认证令牌
	AuthToken string `toml:"auth_token"`
	// AuthType 指定认证类型，例如"nacos"
	AuthType string `toml:"auth_type"`
	// Backend 指定后端存储类型，例如"file"、"nacos"
	Backend string `toml:"backend"`
	// BasicAuth 是否启用基本认证
	BasicAuth bool `toml:"basic_auth"`
	// ClientCaKeys 客户端CA证书密钥路径
	ClientCaKeys string `toml:"client_cakeys"`
	// ClientCert 客户端证书路径
	ClientCert string `toml:"client_cert"`
	// ClientKey 客户端私钥路径
	ClientKey string `toml:"client_key"`
	// ClientInsecure 是否跳过客户端证书验证
	ClientInsecure bool `toml:"client_insecure"`
	// BackendNodes 后端节点列表，用于指定Nacos集群中的节点地址
	BackendNodes util.Nodes `toml:"nodes"`
	// Password 用于Nacos的认证密码
	Password string `toml:"password"`
	// Scheme 指定协议类型，例如"http"或"https"
	Scheme string `toml:"scheme"`
	// Separator 用于配置项路径的分隔符
	Separator string `toml:"separator"`
	// Username 用于Nacos的认证用户名
	Username string `toml:"username"`
	// Filter 配置项过滤规则
	Filter string `toml:"filter"`
	// Path 配置项路径
	Path string `toml:"path"`
	// Group 配置项分组
	Group string `toml:"group"`
	// Endpoint Nacos服务的端点地址
	Endpoint string `toml:"endpoint"`
	// Namespace Nacos的命名空间ID
	Namespace string `toml:"namespace"`
	// AccessKey 用于Nacos的访问密钥
	AccessKey string `toml:"accessKey"`
	// SecretKey 用于Nacos的访问密钥
	SecretKey string `toml:"secretKey"`
	// OpenKMS 是否启用KMS服务
	OpenKMS bool `toml:"openKMS"`
	// RegionId 用于指定地域ID
	RegionId string `toml:"regionId"`
	// Role 角色名称
	Role string
}