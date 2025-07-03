package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"log"
	"os"
	"path/filepath"
	"simple-one-api/pkg/utils"
	"strings"
	"time"
)

// 全局viper实例
var v *viper.Viper

// 配置变更回调函数
var configChangeCallbacks []func()

// InitViperConfig 初始化配置
func InitViperConfig(configName string) error {
	// 解析 JSON 数据到结构体
	var conf *Configuration

	configAbsolutePath, err := utils.ResolveRelativePathToAbsolute(configName)
	if err != nil {
		log.Println("Error getting absolute path:", err)
		return err
	}

	if !utils.FileExists(configAbsolutePath) {
		log.Println("config name:", configAbsolutePath, "not exist")
		configName = "config/" + configName
		configAbsolutePath, err = utils.ResolveRelativePathToAbsolute(configName)
		if err != nil {
			log.Println("Error getting absolute path:", err)
			return err
		}
	}

	log.Println("config name:", configAbsolutePath)

	// 等待文件可读（最多等待30秒）
	if err := waitForFileReadable(configAbsolutePath, 30*time.Second); err != nil {
		log.Printf("Error waiting for file to be readable: %v", err)
		return err
	}

	// 获取文件扩展名
	ext := filepath.Ext(configAbsolutePath)
	if ext == "" {
		log.Println("unsupport config type: no extension")
		return errors.New("unsupport config type: no extension")
	}

	// 初始化viper
	v = viper.New()
	v.SetConfigFile(configAbsolutePath)

	// 根据文件扩展名设置配置类型
	switch strings.ToLower(ext) {
	case ".json":
		v.SetConfigType("json")
	case ".yml", ".yaml":
		v.SetConfigType("yaml")
	default:
		log.Printf("unsupport config type: %s", ext)
		return errors.New("unsupport config type: " + ext)
	}

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		log.Printf("Error reading config file: %v", err)
		return err
	}

	// 加载配置到结构体
	conf, err = loadConfiguration()
	if err != nil {
		log.Printf("Error loading configuration: %v", err)
		return err
	}

	// 应用配置
	applyConfiguration(conf)

	// 设置文件监听
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		log.Printf("Config file changed: %s", e.Name)
		onConfigChange()
	})

	log.Println("Configuration initialized successfully with file watching enabled")
	return nil
}

// waitForFileReadable 等待文件可读
func waitForFileReadable(filePath string, maxWaitTime time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), maxWaitTime)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for file to be readable: %s", filePath)
		case <-ticker.C:
			// 检查文件是否存在且可读
			if file, err := os.OpenFile(filePath, os.O_RDONLY, 0); err == nil {
				file.Close()
				return nil
			}
		}
	}
}

// onConfigChange 配置文件变更处理函数
func onConfigChange() {
	log.Println("Configuration file changed, reloading...")

	// 重新加载配置
	conf, err := loadConfiguration()
	if err != nil {
		log.Printf("Failed to reload configuration: %v", err)
		return
	}

	// 应用新配置
	applyConfiguration(conf)

	// 执行注册的回调函数
	for _, callback := range configChangeCallbacks {
		callback()
	}

	log.Println("Configuration reloaded successfully")
}

// applyConfiguration 应用配置到全局变量
func applyConfiguration(conf *Configuration) {
	// 设置负载均衡策略，默认为 "random"
	if conf.LoadBalancing == "" {
		LoadBalancingStrategy = "random"
	} else {
		LoadBalancingStrategy = conf.LoadBalancing
	}

	GSOAConf = conf
	GProxyConf = &(conf.Proxy)

	if conf.APIKey != "" {
		APIKey = conf.APIKey
	}

	initAPIKeyMap()

	// 设置服务器端口，默认为 ":9090"
	if conf.ServerPort == "" {
		ServerPort = ":9090"
	} else {
		ServerPort = conf.ServerPort
	}

	Debug = conf.Debug
	LogLevel = conf.LogLevel

	// 创建映射
	ModelToService = createModelToServiceMap(*conf)
	GlobalModelRedirect = conf.ModelRedirect
	GTranslation = &conf.Translation

	// 更新多内容模型支持列表
	// 重置为基础列表，然后添加配置中的模型
	SupportMultiContentModels = []string{"gpt-4o", "gpt-4-turbo", "glm-4v", "gemini-*", "yi-vision", "gpt-4o*"}
	if len(conf.MultiContentModels) > 0 {
		SupportMultiContentModels = append(SupportMultiContentModels, conf.MultiContentModels...)
	}

	log.Println("Configuration applied successfully")
	log.Println("LoadBalancingStrategy:", LoadBalancingStrategy)
	log.Println("ServerPort:", ServerPort)
	log.Println("LogLevel:", LogLevel)
	log.Println("GlobalModelRedirect:", GlobalModelRedirect)
	log.Println("SupportMultiContentModels:", SupportMultiContentModels)

	ShowSupportModels()
}

// loadConfiguration 从viper加载配置到结构体
func loadConfiguration() (*Configuration, error) {
	var conf Configuration

	// 使用viper的Unmarshal方法
	if err := v.Unmarshal(&conf); err != nil {
		// 如果是JSON语法错误，提供详细的错误信息
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			// 读取原始文件内容用于错误定位
			data, readErr := os.ReadFile(v.ConfigFileUsed())
			if readErr == nil {
				line, character := FindLineAndCharacter(data, int(syntaxErr.Offset))
				log.Printf("JSON 语法错误在第 %d 行，第 %d 个字符附近: %v\n", line, character, err)
				log.Printf("上下文: %s\n", GetErrorContext(data, int(syntaxErr.Offset)))
			}
		}
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &conf, nil
}

// RegisterConfigChangeCallback 注册配置变更回调函数
func RegisterConfigChangeCallback(callback func()) {
	configChangeCallbacks = append(configChangeCallbacks, callback)
}

// GetViper 获取全局viper实例
func GetViper() *viper.Viper {
	return v
}

// ReloadConfig 手动重新加载配置
func ReloadConfig() error {
	if v == nil {
		return errors.New("viper not initialized")
	}

	// 重新读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// 重新加载配置
	conf, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// 应用新配置
	applyConfiguration(conf)

	return nil
}
