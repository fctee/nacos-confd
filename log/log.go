// 定义一个全局变量tag，用于存储日志标签
var tag string

// init函数在包初始化时调用，设置日志格式
func init() {
	// 获取程序运行时的名称作为日志标签
	tag = os.Args[0]
	// 设置日志格式为ConfdFormatter自定义格式
	log.SetFormatter(&ConfdFormatter{})
}

// SetTag函数用于设置日志标签
func SetTag(t string) {
	tag = t
}

// SetLevel函数用于设置日志级别
func SetLevel(level string) {
	// 解析传入的级别字符串为log.Level类型
	lvl, err := log.ParseLevel(level)
	// 如果解析出错，则输出错误并退出程序
	if err != nil {
		Fatal(fmt.Sprintf(`not a valid level: "%s"`, level))
	}
	// 设置日志级别
	log.SetLevel(lvl)
}

// Debug函数用于输出调试信息
func Debug(format string, v ...interface{}) {
	// 使用fmt.Sprintf处理格式化字符串和参数，然后调用log.Debug输出日志
	log.Debug(fmt.Sprintf(format, v...))
}

// Error函数用于输出错误信息
func Error(format string, v ...interface{}) {
	// 使用fmt.Sprintf处理格式化字符串和参数，然后调用log.Error输出日志
	log.Error(fmt.Sprintf(format, v...))
}

// Fatal函数用于输出致命错误信息并退出程序
func Fatal(format string, v ...interface{}) {
	// 使用fmt.Sprintf处理格式化字符串和参数，然后调用log.Fatal输出日志并退出程序
	log.Fatal(fmt.Sprintf(format, v...))
}

// Info函数用于输出普通信息
func Info(format string, v ...interface{}) {
	// 使用fmt.Sprintf处理格式化字符串和参数，然后调用log.Info输出日志
	log.Info(fmt.Sprintf(format, v...))
}

// Warning函数用于输出警告信息
func Warning(format string, v ...interface{}) {
	// 使用fmt.Sprintf处理格式化字符串和参数，然后调用log.Warning输出日志
	log.Warning(fmt.Sprintf(format, v...))
}
