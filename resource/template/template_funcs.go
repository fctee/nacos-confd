// 导入所需的包
import (
	"encoding/base64" // 用于base64编码和解码
	"encoding/json"   // 用于JSON的编码和解码
	"errors"          // 错误处理
	"fmt"             // 格式化输出
	"net"             // 网络相关的功能
	"os"              // 操作系统相关的功能，如文件操作
	"path"            // 路径操作
	"sort"            // 排序功能
	"strconv"         // 字符串和其他基本数据类型之间的转换
	"strings"         // 字符串操作
	"time"            // 时间处理

	// 导入自定义的包
	"github.com/Risingtao/nacos-confd/depends/memkv"
	util "github.com/Risingtao/nacos-confd/util"
)

// newFuncMap 创建一个新的函数映射，用于模板渲染时调用
func newFuncMap() map[string]interface{} {
	m := make(map[string]interface{}) // 创建一个空的函数映射
	// 添加各种实用函数到映射中
	m["base"] = path.Base              // 获取路径的最后一个元素
	m["split"] = strings.Split        // 根据分隔符分割字符串
	m["json"] = UnmarshalJsonObject    // 解析JSON对象
	m["jsonArray"] = UnmarshalJsonArray // 解析JSON数组
	m["dir"] = path.Dir               // 获取路径的目录部分
	m["map"] = CreateMap               // 创建一个新的映射
	m["getenv"] = Getenv               // 获取环境变量
	m["join"] = strings.Join          // 将字符串数组连接成一个字符串
	m["datetime"] = time.Now           // 获取当前时间
	m["toUpper"] = strings.ToUpper     // 将字符串转换为大写
	m["toLower"] = strings.ToLower     // 将字符串转换为小写
	m["contains"] = strings.Contains   // 检查字符串是否包含另一个字符串
	m["replace"] = strings.Replace     // 替换字符串中的指定子串
	m["trimSuffix"] = strings.TrimSuffix // 去除字符串末尾的指定后缀
	m["lookupIP"] = LookupIP           // 查找IP地址
	m["lookupIPV4"] = LookupIPV4       // 查找IPv4地址
	m["lookupIPV6"] = LookupIPV6       // 查找IPv6地址
	m["lookupSRV"] = LookupSRV         // 查找SRV记录
	m["fileExists"] = util.IsFileExist // 检查文件是否存在
	m["base64Encode"] = Base64Encode   // 对数据进行base64编码
	m["base64Decode"] = Base64Decode   // 对base64编码的数据进行解码
	m["parseBool"] = strconv.ParseBool // 将字符串转换为布尔值
	m["reverse"] = Reverse             // 反转数据（可能是字符串或其他可迭代类型）
	m["sortByLength"] = SortByLength   // 根据长度排序（可能是字符串数组）
	m["sortKVByLength"] = SortKVByLength // 根据键值对的长度排序
	// 添加基本的数学运算函数
	m["add"] = func(a, b int) int { return a + b }
	m["sub"] = func(a, b int) int { return a - b }
	m["div"] = func(a, b int) int { return a / b }
	m["mod"] = func(a, b int) int { return a % b }
	m["mul"] = func(a, b int) int { return a * b }
	m["seq"] = Seq                     // 生成序列（可能是数字序列）
	m["atoi"] = strconv.Atoi           // 将字符串转换为整数
	return m                           // 返回函数映射
}

// addFuncs函数用于将输入的map中的函数添加到输出的map中
func addFuncs(out, in map[string]interface{}) {
	for name, fn := range in {
		out[name] = fn
	}
}

// Seq函数用于生成一个从first到last的整数切片
func Seq(first, last int) []int {
	var arr []int
	for i := first; i <= last; i++ {
		arr = append(arr, i)
	}
	return arr
}

// byLengthKV类型用于定义一个根据键长度排序的KVPair切片
type byLengthKV []memkv.KVPair

// Len方法返回切片的长度
func (s byLengthKV) Len() int {
	return len(s)
}

// Swap方法交换切片中的两个元素
func (s byLengthKV) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less方法根据键的长度比较两个元素的顺序
func (s byLengthKV) Less(i, j int) bool {
	return len(s[i].Key) < len(s[j].Key)
}

// SortKVByLength函数用于根据键的长度对KVPair切片进行排序
func SortKVByLength(values []memkv.KVPair) []memkv.KVPair {
	sort.Sort(byLengthKV(values))
	return values
}

// byLength类型用于定义一个根据长度排序的字符串切片
type byLength []string

// Len方法返回切片的长度
func (s byLength) Len() int {
	return len(s)
}

// Swap方法交换切片中的两个元素
func (s byLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less方法根据字符串的长度比较两个元素的顺序
func (s byLength) Less(i, j int) bool {
	return len(s[i]) < len(s[j])
}

// SortByLength函数用于根据字符串的长度对字符串切片进行排序
func SortByLength(values []string) []string {
	sort.Sort(byLength(values))
	return values
}

// Reverse函数用于反转字符串切片或KVPair切片的顺序
func Reverse(values interface{}) interface{} {
	switch values.(type) {
	case []string:
		v := values.([]string)
		for left, right := 0, len(v)-1; left< right; left, right = left+1, right-1 {
			v[left], v[right] = v[right], v[left]
		}
	case []memkv.KVPair:
		v := values.([]memkv.KVPair)
		for left, right := 0, len(v)-1; left< right; left, right = left+1, right-1 {
			v[left], v[right] = v[right], v[left]
		}
	}
	return values
}

// Getenv函数用于获取环境变量的值，如果不存在则返回默认值
func Getenv(key string, v ...string) string {
	defaultValue := ""
	if len(v) > 0 {
		defaultValue = v[0]
	}

	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// CreateMap函数用于根据传入的键值对创建一个map
func CreateMap(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid map call")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("map keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

// UnmarshalJsonObject函数用于将JSON字符串反序列化为map[string]interface{}
func UnmarshalJsonObject(data string) (map[string]interface{}, error) {
	var ret map[string]interface{}
	err := json.Unmarshal([]byte(data), &ret)
	return ret, err
}

// UnmarshalJsonArray函数用于将JSON字符串反序列化为[]interface{}
func UnmarshalJsonArray(data string) ([]interface{}, error) {
	var ret []interface{}
	err := json.Unmarshal([]byte(data), &ret)
	return ret, err
}

// LookupIP函数用于根据域名查找IP地址
func LookupIP(data string) []string {
	ips, err := net.LookupIP(data)
	if err != nil {
		return nil
	}
	// "Cast" IPs into strings and sort the array
	ipStrings := make([]string, len(ips))

	for i, ip := range ips {
		ipStrings[i] = ip.String()
	}
	sort.Strings(ipStrings)
	return ipStrings
}

// LookupIPV6函数用于根据域名查找IPv6地址
func LookupIPV6(data string) []string {
	var addresses []string
	for _, ip := range LookupIP(data) {
		if strings.Contains(ip, ":") {
			addresses = append(addresses, ip)
		}
	}
	return addresses
}

// LookupIPV4 函数接收一个字符串参数 data，返回 data 中所有 IPv4 地址的切片。
func LookupIPV4(data string) []string {
	var addresses []string // 定义一个字符串切片用于存储找到的 IPv4 地址
	// 遍历 LookupIP 函数返回的所有 IP 地址
	for _, ip := range LookupIP(data) {
		if strings.Contains(ip, ".") { // 检查 IP 地址是否包含 "."，以确定是否为 IPv4 地址
			addresses = append(addresses, ip) // 如果是 IPv4 地址，则添加到 addresses 切片中
		}
	}
	return addresses // 返回包含所有 IPv4 地址的切片
}

// sortSRV 类型定义为一个 net.SRV 指针的切片，用于对 SRV 记录进行排序。
type sortSRV []*net.SRV

// Len 方法返回 sortSRV 类型的长度，这是 sort.Interface 接口的一部分。
func (s sortSRV) Len() int {
	return len(s)
}

// Swap 方法交换 sortSRV 类型中的两个元素，这也是 sort.Interface 接口的一部分。
func (s sortSRV) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less 方法比较 sortSRV 类型中的两个元素，根据 SRV 记录的 Target、Port、Priority 和 Weight 进行排序。
func (s sortSRV) Less(i, j int) bool {
	str1 := fmt.Sprintf("%s%d%d%d", s[i].Target, s[i].Port, s[i].Priority, s[i].Weight)
	str2 := fmt.Sprintf("%s%d%d%d", s[j].Target, s[j].Port, s[j].Priority, s[j].Weight)
	return str1 < str2 // 返回比较结果
}

// LookupSRV 函数接收 service、proto 和 name 三个字符串参数，返回与这些参数匹配的 SRV 记录的切片。
func LookupSRV(service, proto, name string) []*net.SRV {
	_, addrs, err := net.LookupSRV(service, proto, name) // 使用 net.LookupSRV 函数查找 SRV 记录
	if err != nil { // 如果查找过程中出现错误
		return []*net.SRV{} // 返回一个空的 SRV 记录切片
	}
	sort.Sort(sortSRV(addrs)) // 对找到的 SRV 记录进行排序
	return addrs // 返回排序后的 SRV 记录切片
}

// Base64Encode 函数接收一个字符串参数 data，返回 data 的 Base64 编码字符串。
func Base64Encode(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data)) // 使用 base64 标准编码进行编码
}

// Base64Decode 函数接收一个字符串参数 data，返回 data 的 Base64 解码字符串和可能出现的错误。
func Base64Decode(data string) (string, error) {
	s, err := base64.StdEncoding.DecodeString(data) // 使用 base64 标准编码进行解码
	return string(s), err // 返回解码后的字符串和错误（如果有）
}