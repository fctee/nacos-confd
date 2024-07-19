// 导入必要的包
import (
	"flag" // 用于解析命令行参数
	"io/ioutil" // 用于读取文件内容
	"os" // 操作系统相关功能，如文件操作
	"path/filepath" // 用于处理文件路径

	// 导入项目内部的包
	"github.com/Risingtao/nacos-confd/backends" // 后端配置相关
	"github.com/Risingtao/nacos-confd/log" // 日志处理
	"github.com/Risingtao/nacos-confd/resource/template" // 模板处理
	"github.com/Risingtao/nacos-confd/depends/toml" // TOML配置解析
)

// 定义模板配置和后端配置的类型别名
type TemplateConfig = template.Config
type BackendsConfig = backends.Config

// 定义全局配置结构体
type Config struct {
	TemplateConfig // 模板配置
	BackendsConfig // 后端配置
	Interval      int    `toml:"interval"` // 轮询间隔时间
	SecretKeyring string `toml:"secret_keyring"` // 密钥环路径
	SRVDomain     string `toml:"srv_domain"` // 服务域名
	SRVRecord     string `toml:"srv_record"` // 服务记录
	LogLevel      string `toml:"log-level"` // 日志级别
	Watch         bool   `toml:"watch"` // 是否启用监听
	PrintVersion  bool   // 是否打印版本信息
	ConfigFile    string // 配置文件路径
	OneTime       bool   // 是否只运行一次
}

// 全局变量config用于存储配置信息
var config Config

// init函数用于初始化命令行参数
func init() {
	// 使用flag包定义命令行参数
	flag.StringVar(&config.AuthToken, "auth-token", "", "Auth bearer token to use")
	flag.StringVar(&config.Backend, "backend", "etcd", "backend to use")
	flag.StringVar(&config.ClientCaKeys, "client-ca-keys", "", "client ca keys")
	flag.StringVar(&config.ClientCert, "client-cert", "", "the client cert")
	flag.StringVar(&config.ClientKey, "client-key", "", "the client key")
	flag.StringVar(&config.ConfDir, "confdir", "/etc/confd", "confd conf directory")
	flag.StringVar(&config.ConfigFile, "config-file", "/etc/confd/confd.toml", "the confd config file")
	flag.IntVar(&config.Interval, "interval", 600, "backend polling interval")
	flag.BoolVar(&config.KeepStageFile, "keep-stage-file", false, "keep staged files")
	flag.StringVar(&config.LogLevel, "log-level", "", "level which confd should log messages")
	flag.Var(&config.BackendNodes, "node", "list of backend nodes")
	flag.BoolVar(&config.Noop, "noop", false, "only show pending changes")
	flag.BoolVar(&config.OneTime, "onetime", false, "run once and exit")
	flag.StringVar(&config.Prefix, "prefix", "", "key path prefix")
	flag.BoolVar(&config.PrintVersion, "version", false, "print version and exit")
	flag.StringVar(&config.Scheme, "scheme", "http", "the backend URI scheme for nodes retrieved from DNS SRV records (http or https)")
	flag.StringVar(&config.SecretKeyring, "secret-keyring", "", "path to armored PGP secret keyring (for use with crypt functions)")
	flag.BoolVar(&config.SyncOnly, "sync-only", false, "sync without check_cmd and reload_cmd")
	flag.StringVar(&config.AuthType, "auth-type", "", "Vault auth backend type to use (only used with -backend=vault)")
	flag.StringVar(&config.Endpoint, "endpoint", "", "the endpoint in nacos (only used with nacos backends)")
	flag.StringVar(&config.Group, "group", "DEFAULT_GROUP", "the group in nacos (only used with nacos backends)")
	flag.StringVar(&config.Namespace, "namespace", "", "the namespace in nacos (only used with nacos backends)")
	flag.StringVar(&config.AccessKey, "accessKey", "", "the accessKey to authenticate in nacos (only used with nacos backends)")
	flag.StringVar(&config.SecretKey, "secretKey", "", "the secretKey to authenticate in nacos (only used with nacos backends)")
	flag.BoolVar(&config.OpenKMS, "openKMS", false, "the switch if open kms in nacos (only used with nacos backends)")
	flag.StringVar(&config.RegionId, "regionId", "", "the kms regionId in nacos (only used with nacos backends)")
	flag.BoolVar(&config.Watch, "watch", false, "enable watch support")
}

// initConfig函数用于初始化配置信息
func initConfig() error {
	// 检查配置文件是否存在
	_, err := os.Stat(config.ConfigFile)
	if os.IsNotExist(err) {
		log.Debug("Skipping confd config file.")
	} else {
		log.Debug("Loading " + config.ConfigFile)
		// 读取配置文件内容
		configBytes, err := ioutil.ReadFile(config.ConfigFile)
		if err != nil {
			return err
		}

		// 解析TOML格式的配置文件内容到config结构体
		_, err = toml.Decode(string(configBytes), &config)
		if err != nil {
			return err
		}
	}

	// 如果指定了密钥环路径，则读取密钥环内容
	if config.SecretKeyring != "" {
		kr, err := os.Open(config.SecretKeyring)
		if err != nil {
			log.Fatal(err.Error())
		}
		defer kr.Close()
		config.PGPPrivateKey, err = ioutil.ReadAll(kr)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	// 如果指定了日志级别，则设置日志级别
	if config.LogLevel != "" {
		log.SetLevel(config.LogLevel)
	}

	// 如果没有指定后端节点，则根据后端类型设置默认节点
	if len(config.BackendNodes) == 0 {
		switch config.Backend {
		case "nacos":
			config.BackendNodes = []string{"127.0.0.1:8848"}
		}
	}

	log.Info("Backend set to " + config.Backend)

	// 设置配置文件和模板文件的目录路径
	config.ConfigDir = filepath.Join(config.ConfDir, "conf.d")
	config.TemplateDir = filepath.Join(config.ConfDir, "templates")
	return nil
}
