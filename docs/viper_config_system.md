# Viper 配置系统使用指南

## 概述

simple-one-api 现在使用 [Viper](https://github.com/spf13/viper) 作为配置管理库，提供了以下功能：

- 支持 JSON 和 YAML 配置文件格式
- 自动文件变更监听和热重载
- 启动时等待文件可读（防止文件被其他程序写入时的问题）
- 配置变更回调机制
- 更好的错误处理和调试信息

## 主要特性

### 1. 文件格式支持

支持以下配置文件格式：
- JSON (`.json`)
- YAML (`.yml`, `.yaml`)

### 2. 自动文件监听

系统会自动监听配置文件的变更，当文件发生变化时会：
- 自动重新加载配置
- 更新所有相关的全局变量
- 执行注册的配置变更回调函数
- 记录详细的日志信息

### 3. 启动时文件等待

在启动时，如果配置文件正在被其他程序写入，系统会：
- 等待文件变为可读状态（最多30秒）
- 避免因文件锁定导致的启动失败
- 提供详细的等待状态日志

### 4. 配置变更回调

可以注册回调函数来处理配置变更事件：

```go
import "simple-one-api/pkg/config"

// 注册配置变更回调
config.RegisterConfigChangeCallback(func() {
    // 在这里处理配置变更后的逻辑
    log.Println("Configuration changed, updating application state...")
})
```

## 使用方法

### 基本使用

```go
package main

import (
    "log"
    "simple-one-api/pkg/config"
)

func main() {
    // 初始化配置（支持相对路径和绝对路径）
    err := config.InitConfig("config.json")
    if err != nil {
        log.Fatalf("Failed to initialize config: %v", err)
    }
    
    // 配置会自动监听文件变更
    // 无需手动重新加载
}
```

### 手动重新加载配置

```go
// 手动重新加载配置
err := config.ReloadConfig()
if err != nil {
    log.Printf("Failed to reload config: %v", err)
}
```

### 获取 Viper 实例

```go
// 获取全局 viper 实例（用于高级操作）
v := config.GetViper()
if v != nil {
    // 使用 viper 实例进行自定义操作
    value := v.GetString("some.key")
}
```

## 配置文件示例

### JSON 格式

```json
{
  "server_port": ":9090",
  "debug": true,
  "log_level": "info",
  "load_balancing": "random",
  "api_key": "your-api-key-here",
  "services": {
    "openai": [
      {
        "provider": "openai",
        "models": ["gpt-4", "gpt-3.5-turbo"],
        "enabled": true,
        "credentials": {
          "api_key": "your-openai-api-key"
        }
      }
    ]
  }
}
```

### YAML 格式

```yaml
server_port: ":9090"
debug: true
log_level: "info"
load_balancing: "random"
api_key: "your-api-key-here"

services:
  openai:
    - provider: "openai"
      models:
        - "gpt-4"
        - "gpt-3.5-turbo"
      enabled: true
      credentials:
        api_key: "your-openai-api-key"
```

## 错误处理

### JSON 语法错误

当 JSON 配置文件存在语法错误时，系统会提供详细的错误信息：

```
JSON 语法错误在第 15 行，第 8 个字符附近: invalid character '}' looking for beginning of object key string
上下文: "models": ["gpt-4", "gpt-3.5-turbo"}}
```

### 文件读取错误

如果配置文件无法读取，系统会提供清晰的错误信息：

```
Error reading config file: open config.json: no such file or directory
```

### 文件等待超时

如果文件在指定时间内无法变为可读状态：

```
timeout waiting for file to be readable: /path/to/config.json
```

## 日志输出

系统会输出详细的配置加载和变更日志：

```
config name: /path/to/config.json
Configuration applied successfully
LoadBalancingStrategy: random
ServerPort: :9090
LogLevel: info
GlobalModelRedirect: map[gpt-4:gpt-4-turbo]
SupportMultiContentModels: [gpt-4o gpt-4-turbo glm-4v gemini-* yi-vision gpt-4o* custom-multi-model-1 custom-multi-model-2]
Configuration initialized successfully with file watching enabled
```

当配置文件发生变更时：

```
Config file changed: /path/to/config.json
Configuration file changed, reloading...
Configuration applied successfully
Configuration reloaded successfully
```

## 迁移指南

### 从旧版本迁移

1. **配置文件格式**：现有配置文件无需修改，系统会自动识别格式
2. **API 兼容性**：所有现有的配置相关 API 保持不变
3. **新增功能**：可以逐步使用新的功能，如配置变更回调

### 最佳实践

1. **使用相对路径**：推荐使用相对路径指定配置文件
2. **监控日志**：关注配置加载和变更的日志输出
3. **错误处理**：正确处理配置初始化错误
4. **回调函数**：合理使用配置变更回调函数

## 故障排除

### 常见问题

1. **配置文件无法读取**
   - 检查文件路径是否正确
   - 确认文件权限
   - 检查文件是否被其他程序锁定

2. **配置变更不生效**
   - 确认文件监听已启用
   - 检查文件变更事件是否被触发
   - 查看日志中的错误信息

3. **启动时等待时间过长**
   - 检查配置文件是否被其他程序占用
   - 考虑增加等待时间或优化文件访问

### 调试技巧

1. **启用调试模式**：设置 `debug: true` 获取更多日志信息
2. **检查文件权限**：确保程序有读取配置文件的权限
3. **监控文件系统事件**：使用系统工具监控文件变更事件
4. **查看详细日志**：关注配置加载和变更的详细日志输出 