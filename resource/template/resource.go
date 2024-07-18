package template

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Risingtao/nacos-confd/backends"
	"github.com/Risingtao/nacos-confd/crypt/encoding/secconf"
	"github.com/Risingtao/nacos-confd/log"
	"github.com/Risingtao/nacos-confd/memkv"
	"github.com/Risingtao/nacos-confd/toml"
	"github.com/Risingtao/nacos-confd/util"
)

type Config struct {
	ConfDir       string `toml:"confdir"`
	ConfigDir     string
	KeepStageFile bool
	Noop          bool   `toml:"noop"`
	Prefix        string `toml:"prefix"`
	StoreClient   backends.StoreClient
	SyncOnly      bool `toml:"sync-only"`
	TemplateDir   string
	PGPPrivateKey []byte
}

type TemplateResourceConfig struct {
	TemplateResource TemplateResource `toml:"template"`
}

type TemplateResource struct {
	CheckCmd      string `toml:"check_cmd"`
	Dest          string
	FileMode      os.FileMode
	Gid           int
	Keys          []string
	Mode          string
	Prefix        string
	ReloadCmd     string `toml:"reload_cmd"`
	Src           string
	StageFile     *os.File
	Uid           int
	funcMap       map[string]interface{}
	lastIndex     uint64
	keepStageFile bool
	noop          bool
	store         memkv.Store
	storeClient   backends.StoreClient
	syncOnly      bool
	PGPPrivateKey []byte
}

// 错误类型
var ErrEmptySrc = errors.New("空的 src 模板")

// LogEntry 定义
type LogEntry [2]string

type Stream struct {
	Stream map[string]string `json:"stream"`
	Values []LogEntry        `json:"values"`
}

type Payload struct {
	Streams []Stream `json:"streams"`
}

// 获取内网 IP 地址
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", errors.New("没有找到有效的IP地址")
}

// 发送日志到loki
func SendLogToLoki(lokiURL string, labels map[string]string, logLine string) {
	// 使用 goroutine 异步发送日志
	go func() {
		now := time.Now().UnixNano()
		entry := LogEntry{fmt.Sprintf("%d", now), logLine}

		stream := Stream{
			Stream: labels,
			Values: []LogEntry{entry},
		}

		payload := Payload{
			Streams: []Stream{stream},
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			log.Error("Loki日志序列化失败: %v", err)
			return
		}

		req, err := http.NewRequest("POST", lokiURL, bytes.NewBuffer(jsonData))
		if err != nil {
			log.Error("创建Loki请求失败: %v", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{
			Timeout: 5 * time.Second, // 设置超时时间
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Error("发送Loki日志失败: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			log.Error("Loki返回非预期状态码: %d", resp.StatusCode)
		}
	}()
}

func NewTemplateResource(path string, config Config) (*TemplateResource, error) {
	if config.StoreClient == nil {
		return nil, errors.New("需要一个有效的 StoreClient。")
	}

	tc := &TemplateResourceConfig{TemplateResource{Uid: -1, Gid: -1}}

	log.Debug("从 " + path + " 加载模板资源")
	_, err := toml.DecodeFile(path, &tc)
	if err != nil {
		return nil, fmt.Errorf("无法处理模板资源 %s - %s", path, err.Error())
	}

	tr := tc.TemplateResource
	tr.keepStageFile = config.KeepStageFile
	tr.noop = config.Noop
	tr.storeClient = config.StoreClient
	tr.funcMap = newFuncMap()
	tr.store = memkv.New()
	tr.syncOnly = config.SyncOnly
	addFuncs(tr.funcMap, tr.store.FuncMap)

	if config.Prefix != "" {
		tr.Prefix = config.Prefix
	}

	if !strings.HasPrefix(tr.Prefix, "/") {
		tr.Prefix = "/" + tr.Prefix
	}

	if len(config.PGPPrivateKey) > 0 {
		tr.PGPPrivateKey = config.PGPPrivateKey
		addCryptFuncs(&tr)
	}

	if tr.Src == "" {
		return nil, ErrEmptySrc
	}

	if tr.Uid == -1 {
		tr.Uid = os.Geteuid()
	}

	if tr.Gid == -1 {
		tr.Gid = os.Getegid()
	}

	tr.Src = filepath.Join(config.TemplateDir, tr.Src)
	return &tr, nil
}

func addCryptFuncs(tr *TemplateResource) {
	addFuncs(tr.funcMap, map[string]interface{}{
		"cget": func(key string) (memkv.KVPair, error) {
			kv, err := tr.funcMap["get"].(func(string) (memkv.KVPair, error))(key)
			if err == nil {
				var b []byte
				b, err = secconf.Decode([]byte(kv.Value), bytes.NewBuffer(tr.PGPPrivateKey))
				if err == nil {
					kv.Value = string(b)
				}
			}
			return kv, err
		},
		"cgets": func(pattern string) (memkv.KVPairs, error) {
			kvs, err := tr.funcMap["gets"].(func(string) (memkv.KVPairs, error))(pattern)
			if err == nil {
				for i := range kvs {
					b, err := secconf.Decode([]byte(kvs[i].Value), bytes.NewBuffer(tr.PGPPrivateKey))
					if err != nil {
						return memkv.KVPairs(nil), err
					}
					kvs[i].Value = string(b)
				}
			}
			return kvs, err
		},
		"cgetv": func(key string) (string, error) {
			v, err := tr.funcMap["getv"].(func(string, ...string) (string, error))(key)
			if err == nil {
				var b []byte
				b, err = secconf.Decode([]byte(v), bytes.NewBuffer(tr.PGPPrivateKey))
				if err == nil {
					return string(b), nil
				}
			}
			return v, err
		},
		"cgetvs": func(pattern string) ([]string, error) {
			vs, err := tr.funcMap["getvs"].(func(string) ([]string, error))(pattern)
			if err == nil {
				for i := range vs {
					b, err := secconf.Decode([]byte(vs[i]), bytes.NewBuffer(tr.PGPPrivateKey))
					if err != nil {
						return []string(nil), err
					}
					vs[i] = string(b)
				}
			}
			return vs, err
		},
	})
}

func (t *TemplateResource) setVars() error {
	result, err := t.storeClient.GetValues(util.AppendPrefix(t.Prefix, t.Keys))
	if err != nil {
		return err
	}

	// 创建一个新的 map，仅包含键名
	keysOnly := make([]string, 0, len(result))
	for k := range result {
		keysOnly = append(keysOnly, k)
	}
	log.Debug("从存储中获取到以下键: %v", keysOnly)

	t.store.Purge()

	for k, v := range result {
		t.store.Set(path.Join("/", strings.TrimPrefix(k, t.Prefix)), v)
	}
	return nil
}

func (t *TemplateResource) createStageFile() error {
	log.Debug("使用源模板 " + t.Src)

	if !util.IsFileExist(t.Src) {
		return errors.New("缺少模板: " + t.Src)
	}

	// 读取模板内容以确保它正确包含所有行
	templateContent, err := ioutil.ReadFile(t.Src)
	if err != nil {
		return fmt.Errorf("无法读取模板 %s, %s", t.Src, err)
	}

	tmpl, err := template.New(filepath.Base(t.Src)).Funcs(t.funcMap).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("无法处理模板 %s, %s", t.Src, err)
	}

	// 在目标目录中创建临时文件以避免跨文件系统问题
	temp, err := ioutil.TempFile(filepath.Dir(t.Dest), "."+filepath.Base(t.Dest))
	if err != nil {
		return err
	}

	// 使用缓冲区执行模板并捕获所有行，包括空行
	var buffer bytes.Buffer
	if err = tmpl.Execute(&buffer, nil); err != nil {
		temp.Close()
		os.Remove(temp.Name())
		return err
	}

	// 将缓冲区内容写入临时文件
	if _, err = temp.Write(buffer.Bytes()); err != nil {
		temp.Close()
		os.Remove(temp.Name())
		return err
	}

	defer temp.Close()

	// 现在在阶段文件上设置所有者、组和模式，以便稍后更容易地与目标配置文件进行比较。
	os.Chmod(temp.Name(), t.FileMode)
	os.Chown(temp.Name(), t.Uid, t.Gid)
	t.StageFile = temp
	return nil
}

func (t *TemplateResource) sync() error {
    staged := t.StageFile.Name()
    defer func() {
        if !t.keepStageFile {
            os.Remove(staged)
        } else {
            log.Info("保留暂存文件: %s", staged)
        }
    }()

    log.Debug("正在比较候选配置与 %s", t.Dest)
    changed, err := util.IsConfigChanged(staged, t.Dest)
    if err != nil {
        return fmt.Errorf("比较配置时出错: %v", err)
    }

    if t.noop {
        log.Warning("Noop 模式已启用。%s 不会被修改", t.Dest)
        return nil
    }

    if !changed {
        log.Debug("目标配置 %s 已同步", t.Dest)
        return nil
    }

    log.Info("目标配置 %s 不同步", t.Dest)

    if err := t.performSync(staged); err != nil {
        return err
    }

    if err := t.sendSyncNotification(); err != nil {
        log.Warning("发送同步通知失败: %v", err)
    }

    return nil
}

func (t *TemplateResource) performSync(staged string) error {
    if !t.syncOnly && t.CheckCmd != "" {
        if err := t.check(); err != nil {
            return fmt.Errorf("配置检查失败: %v", err)
        }
    }

    log.Debug("正在覆盖目标配置 %s", t.Dest)
    if err := t.replaceConfig(staged); err != nil {
        return err
    }

    if !t.syncOnly && t.ReloadCmd != "" {
        log.Info("执行reload脚本: %s", t.ReloadCmd)
        if err := t.reload(); err != nil {
            return fmt.Errorf("重新加载配置失败: %v", err)
        }
    }

    log.Info("目标配置 %s 已更新", t.Dest)
    return nil
}

func (t *TemplateResource) replaceConfig(staged string) error {
    err := os.Rename(staged, t.Dest)
    if err == nil {
        return nil
    }

    if !strings.Contains(err.Error(), "device or resource busy") {
        return fmt.Errorf("重命名文件失败: %v", err)
    }

    log.Debug("重命名失败 - 目标可能是一个挂载点。尝试写入")
    contents, err := ioutil.ReadFile(staged)
    if err != nil {
        return fmt.Errorf("读取暂存文件失败: %v", err)
    }

    if err := ioutil.WriteFile(t.Dest, contents, t.FileMode); err != nil {
        return fmt.Errorf("写入目标文件失败: %v", err)
    }

    if err := os.Chown(t.Dest, t.Uid, t.Gid); err != nil {
        return fmt.Errorf("更改文件所有者失败: %v", err)
    }

    return nil
}

func (t *TemplateResource) sendSyncNotification() error {
    ip, err := getLocalIP()
    if err != nil {
        log.Warning("获取本地 IP 地址失败: %v", err)
        ip = "unknown"
    }

    host, err := os.Hostname()
    if err != nil {
        log.Error("获取主机名失败: %v", err)
        host = "unknown"
    }

    labels := map[string]string{
        "ip":       ip,
        "hostname": host,
        "template": t.Src,
        "config":   t.Dest,
        "reload":   t.ReloadCmd,
    }
    logLine := fmt.Sprintf("IP: %s - 配置同步通知", ip)

    // 异步发送日志到Loki，不等待结果
    go SendLogToLoki("http://loki.xjsj.com:3100/loki/api/v1/push", labels, logLine)

    return nil
}

func (t *TemplateResource) check() error {
	var cmdBuffer bytes.Buffer
	data := make(map[string]string)
	data["src"] = t.StageFile.Name()
	tmpl, err := template.New("checkcmd").Parse(t.CheckCmd)
	if err != nil {
		return err
	}
	if err := tmpl.Execute(&cmdBuffer, data); err != nil {
		return err
	}
	return runCommand(cmdBuffer.String())
}

func (t *TemplateResource) reload() error {
	log.Info("开始执行reload脚本: %s", t.ReloadCmd)
	err := runCommand(t.ReloadCmd)
	if err != nil {
        log.Error("Reload脚本执行失败: %v", err)
        return err
    }
    log.Info("Reload脚本执行成功")
    return nil

}

func runCommand(cmd string) error {
	log.Debug("运行脚本: " + cmd)
	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		c = exec.Command("cmd", "/C", cmd)
	} else {
		c = exec.Command("/bin/sh", "-c", cmd)
	}

	output, err := c.CombinedOutput()
	if err != nil {
		log.Error("脚本执行失败: %s, 错误: %v", string(output), err)
		return fmt.Errorf("脚本执行失败: %s, 错误: %v", string(output), err)
	}
	log.Debug("脚本输出信息 >>>>\n%s", strings.TrimSpace(string(output)))
	return nil
}

func (t *TemplateResource) process() error {
	if err := t.setFileMode(); err != nil {
		return err
	}
	if err := t.setVars(); err != nil {
		return err
	}
	if err := t.createStageFile(); err != nil {
		return err
	}
	if err := t.sync(); err != nil {
		return err
	}
	return nil
}

func (t *TemplateResource) setFileMode() error {
	if t.Mode == "" {
		if !util.IsFileExist(t.Dest) {
			t.FileMode = 0644
		} else {
			fi, err := os.Stat(t.Dest)
			if err != nil {
				return err
			}
			t.FileMode = fi.Mode()
		}
	} else {
		mode, err := strconv.ParseUint(t.Mode, 0, 32)
		if err != nil {
			return err
		}
		t.FileMode = os.FileMode(mode)
	}
	return nil
}