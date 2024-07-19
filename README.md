# nacos-confd(To be improved)
nacos-confd 是一个基于 Nacos 的配置管理工具，用于动态更新和管理应用程序配置。

## 功能特性

- 支持从 Nacos 后端获取配置
- 提供模板处理功能，动态生成配置文件
- 支持一次性处理和持续监听模式
- 可配置的处理间隔
- 优雅的错误处理和信号处理

## 安装

```bash
go get github.com/Risingtao/nacos-confd
```

### 使用说明

1. 配置 nacos-confd：
   创建一个配置文件（例如 config.toml），设置必要的参数如 Nacos 服务器地址、命名空间等。

2. 运行 nacos-confd：

   ```bash
   nacos-confd -config config.toml
   ```

   常用命令行参数：

   -config: 指定配置文件路径
   -onetime: 一次性处理模式
   -interval: 设置轮询间隔（秒）
   -watch: 启用文件变化监听模式
   -version: 打印版本信息

3. 模板配置：
   在 /etc/confd/templates 目录下创建模板文件，使用 Go 模板语法。

4. 后端配置：
   在 /etc/confd/conf.d 目录下创建后端配置文件，指定模板源和目标路径。

### 配置示例

```toml
Toml
[nacos]
server_addr = "localhost:8848"
namespace = "public"

[template]
src = "app.conf.tmpl"
dest = "/etc/app/app.conf"
keys = [
    "/config/database",
    "/config/cache"
]
```

### 开发

1. 克隆仓库：

   ```bash
   git clone https://github.com/Risingtao/nacos-confd.git
   ```

2. 安装依赖：

   ```bash
   go mod tidy
   ```

3. 构建项目：

   ```bash
   go build
   ```