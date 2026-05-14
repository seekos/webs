# Webs

一个带有自动刷新功能的 Go Web 服务器。

## 特性

- 静态文件服务
- 文件监控 - 任何文件修改自动触发浏览器刷新
- WebSocket 实时通信
- 启动时自动打开浏览器
- 支持自定义端口和目录
- 支持 webs.toml 配置文件

## 安装

```bash
cd webs
go mod tidy
go run main.go
```

## 使用方法

### 基本用法

```bash
# 启动服务器，默认为当前目录
go run main.go

# 指定端口
go run main.go -port 3000

# 指定目录
go run main.go -dir /path/to/your/project

# 同时指定端口和目录
go run main.go -port 3000 -dir ./public
```

### 配置文件

可在可执行文件同目录下创建 `webs.toml`，配置项如下：

```toml
# 服务器端口
port = 5999
# 要服务和监控的目录
dir = "."
# 监测的文件扩展名，为空则监测所有文件
watch_exts = ["html", "css", "js"]
```

CLI 参数优先级高于配置文件。

### 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| -port | 5999 | 服务器端口 |
| -dir | . | 要监控和服务的目录 |

## 工作原理

1. **文件监控**: 使用 `fsnotify` 库监控指定目录下的所有文件变化
2. **WebSocket**: 服务器内置 WebSocket 服务 (`/ws` 端点)
3. **自动刷新**: 检测到文件变化后，通过 WebSocket 通知所有连接的浏览器进行刷新

## 示例

```bash
# 进入示例目录
cd webs

# 下载依赖
go mod tidy

# 启动服务器
go run main.go -port 8080
```

然后在浏览器打开 `http://localhost:8080`，修改任意文件，浏览器会自动刷新。
