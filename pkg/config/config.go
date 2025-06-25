package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var GSOAConf *Configuration

var ModelToService map[string][]ModelDetails
var LoadBalancingStrategy string
var ServerPort string
var APIKey string
var Debug bool
var LogLevel string
var SupportModels map[string]string
var GlobalModelRedirect map[string]string
var SupportMultiContentModels = []string{"gpt-4o", "gpt-4-turbo", "glm-4v", "gemini-*", "yi-vision", "gpt-4o*"}

// var SupportReasoningModels = []string{"deepseek-reasoner", "gpt-4-turbo"}
var GProxyConf *ProxyConf
var GTranslation *Translation

var apiKeyMap map[string]APIKeyConfig

// 全局viper实例
var v *viper.Viper

// 配置变更回调函数
var configChangeCallbacks []func()

// Limit 结构体定义
type Limit struct {
	QPS         float64 `json:"qps" yaml:"qps"`
	QPM         float64 `json:"qpm" yaml:"qpm"`
	RPM         float64 `json:"rpm" yaml:"rpm"`
	Concurrency float64 `json:"concurrency" yaml:"concurrency"`
	Timeout     int     `json:"timeout" yaml:"timeout"`
}

// Range 结构体定义
type Range struct {
	Min float64 `json:"min" yaml:"min"`
	Max float64 `json:"max" yaml:"max"`
}

// ModelParams 结构体定义
type ModelParams struct {
	TemperatureRange Range `json:"temperatureRange" yaml:"temperatureRange"`
	TopPRange        Range `json:"topPRange" yaml:"topPRange"`
	MaxTokens        int   `json:"maxTokens" yaml:"maxTokens"`
}

// ServiceModel 定义相关结构体
type ServiceModel struct {
	Provider          string                   `json:"provider" yaml:"provider"`
	EmbeddingModels   []string                 `json:"embedding_models" yaml:"embedding_models"`
	EmbeddingLimit    Limit                    `json:"embedding_limit" yaml:"embedding_limit"`
	Models            []string                 `json:"models" yaml:"models"`
	ReasoningModels   map[string]string        `json:"reasoning_models" yaml:"reasoning_models"`
	Enabled           bool                     `json:"enabled" yaml:"enabled"`
	Credentials       map[string]interface{}   `json:"credentials" yaml:"credentials"`
	CredentialList    []map[string]interface{} `json:"credential_list" yaml:"credential_list"`
	ServerUrl         string                   `json:"server_url" yaml:"server_url" mapstructure:"server_url"`
	ModelMap          map[string]string        `json:"model_map" yaml:"model_map"`
	ModelRedirect     map[string]string        `json:"model_redirect" yaml:"model_redirect"`
	Limit             Limit                    `json:"limit" yaml:"limit"`
	UseProxy          *bool                    `json:"use_proxy,omitempty" yaml:"use_proxy,omitempty"`
	Timeout           int                      `json:"timeout" yaml:"timeout"`
	ProviderNamespace string                   `json:"provider_namespace" yaml:"provider_namespace" mapstructure:"provider_namespace"`
}

// ProxyConf 结构体定义
type ProxyConf struct {
	Strategy    string `json:"strategy" yaml:"strategy"`
	Type        string `json:"type" yaml:"type"`
	HTTPProxy   string `json:"http_proxy" yaml:"http_proxy"`
	HTTPSProxy  string `json:"https_proxy" yaml:"https_proxy"`
	Socks5Proxy string `json:"socks5_proxy" yaml:"socks5_proxy"`
	Timeout     int    `json:"timeout" yaml:"timeout"`
}

// Translation 结构体定义
type Translation struct {
	Enable         bool   `json:"enable" yaml:"enable"`
	PromptTemplate string `json:"promptTemplate" yaml:"prompt_template"`
	Retry          int    `json:"retry" yaml:"retry"`
	Concurrency    int    `json:"concurrency" yaml:"concurrency"`
}

// APIKeyConfig 结构体定义
type APIKeyConfig struct {
	APIKey          string              `json:"api_key" yaml:"api_key"`
	SupportedModels map[string][]string `json:"supported_models" yaml:"supported_models"`
}

// Configuration 结构体定义
type Configuration struct {
	ServerPort         string                    `json:"server_port" yaml:"server_port"`
	Debug              bool                      `json:"debug" yaml:"debug"`
	LogLevel           string                    `json:"log_level" yaml:"log_level"`
	Proxy              ProxyConf                 `json:"proxy" yaml:"proxy"`
	APIKey             string                    `json:"api_key" yaml:"api_key"`
	LoadBalancing      string                    `json:"load_balancing" yaml:"load_balancing"`
	MultiContentModels []string                  `json:"multi_content_models" yaml:"multi_content_models"`
	ModelRedirect      map[string]string         `json:"model_redirect" yaml:"model_redirect"`
	ParamsRange        map[string]ModelParams    `json:"params_range" yaml:"params_range"`
	Services           map[string][]ServiceModel `json:"services" yaml:"services"`
	Translation        Translation               `json:"translation" yaml:"translation"`
	EnableWeb          bool                      `json:"enable_web" yaml:"enable_web"`
	APIKeys            []APIKeyConfig            `json:"api_keys" yaml:"api_keys"`
}

// ModelDetails 结构用于返回模型相关的服务信息
type ModelDetails struct {
	ServiceName  string `json:"service_name" yaml:"service_name"`
	ServiceModel `json:",inline" yaml:",inline"`
	ServiceID    string `json:"service_id" yaml:"service_id"`
	Namespace    string `json:"-" yaml:"-"`
}

// 创建模型到服务的映射
func createModelToServiceMap(config Configuration) map[string][]ModelDetails {
	modelToService := make(map[string][]ModelDetails)
	SupportModels = make(map[string]string)
	for serviceName, serviceModels := range config.Services {
		for _, model := range serviceModels {
			if model.Enabled {
				log.Printf("Models: %v, service Timeout:%v,Limit Timeout: %v, QPS: %v, QPM: %v, RPM: %v,Concurrency: %v\n",
					model.Models, model.Timeout, model.Limit.Timeout, model.Limit.QPS, model.Limit.QPM, model.Limit.RPM, model.Limit.Concurrency)

				log.Printf("Models: %v\n", model.EmbeddingModels)

				if len(model.Models) == 0 {
					dmv, exists := DefaultSupportModelMap[serviceName]
					if exists {
						model.Models = dmv
						log.Println("use default support models:", dmv)
					}
				}

				if model.Timeout <= 0 {
					model.Timeout = ServiceTimeOut
				}

				for _, modelName := range model.Models {
					detail := ModelDetails{
						ServiceName:  serviceName,
						ServiceModel: model,
						ServiceID:    uuid.New().String(),
						Namespace:    model.ProviderNamespace,
					}

					//modelNameLower := strings.ToLower(modelName)
					modelToService[modelName] = append(modelToService[modelName], detail)

					//存储支持的模型名称列表
					SupportModels[modelName] = modelName
					for k, v := range detail.ModelRedirect {
						//support models
						SupportModels[k] = v

						_, exists := SupportModels[v]
						if exists {
							delete(SupportModels, v)
						}

						//
						modelToService[k] = append(modelToService[k], detail)
						//delete(modelToService, modelName)
					}
				}

				for _, modelName := range model.EmbeddingModels {
					detail := ModelDetails{
						ServiceName:  serviceName,
						ServiceModel: model,
						ServiceID:    uuid.New().String(),
					}

					//modelNameLower := strings.ToLower(modelName)
					modelToService[modelName] = append(modelToService[modelName], detail)
					for k, _ := range detail.ModelRedirect {
						modelToService[k] = append(modelToService[k], detail)
					}
				}
			}
		}
	}
	return modelToService
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

// RegisterConfigChangeCallback 注册配置变更回调函数
func RegisterConfigChangeCallback(callback func()) {
	configChangeCallbacks = append(configChangeCallbacks, callback)
}

// InitConfig 初始化配置
func InitConfig(configName string) error {
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

/*
// GetAllModelService 根据模型名称获取服务和凭证信息
func GetAllModelService(modelName string) ([]ModelDetails, error) {
	if serviceDetails, found := ModelToService[modelName]; found {
		return serviceDetails, nil
	}
	return nil, fmt.Errorf("model %s not found in the configuration", modelName)
}

*/

// GetModelService 根据模型名称获取启用的服务和凭证信息
func GetModelService(modelName string, namespace string) (*ModelDetails, error) {
	if serviceDetails, found := ModelToService[modelName]; found {
		var enabledServices []ModelDetails
		for _, sd := range serviceDetails {
			if sd.Enabled && sd.ProviderNamespace == namespace {
				enabledServices = append(enabledServices, sd)
			}
		}

		if len(enabledServices) == 0 {
			return nil, fmt.Errorf("no enabled model %s found in the configuration", modelName)
		}

		index := GetLBIndex(LoadBalancingStrategy, modelName, len(enabledServices))

		return &enabledServices[index], nil
	}
	return nil, fmt.Errorf("model %s not found in the configuration", modelName)
}

func GetRandomEnabledModelDetails() (*ModelDetails, error) {

	index := GetLBIndex(LoadBalancingStrategy, KEYNAME_RANDOM, len(ModelToService))

	keys := make([]string, 0, len(ModelToService))

	// 遍历 ModelToService 映射，收集所有 Enabled 为 true 的 ModelDetails
	for modelName := range ModelToService {
		keys = append(keys, modelName)
	}

	sort.Strings(keys)

	model := keys[index]

	modelDetails := ModelToService[model]

	index2 := GetLBIndex(LoadBalancingStrategy, model, len(modelDetails))

	randomModel := modelDetails[index2]

	return &randomModel, nil
}

func GetRandomEnabledModelDetailsV1() (*ModelDetails, string, error) {
	md, err := GetRandomEnabledModelDetails()
	if err != nil {
		return nil, "", err
	}

	randomString := md.Models[getRandomIndex(len(md.Models))]

	//	log.Println(randomString)

	return md, randomString, nil

}

// GetModelMapping 函数，根据model在ModelMap中查找对应的映射，如果找不到则返回原始model
func GetModelMapping(s *ModelDetails, model string) string {
	if mappedModel, exists := s.ModelMap[model]; exists {
		mylog.Logger.Info("model map found", zap.String("model", model), zap.String("mappedModel", mappedModel))
		return mappedModel
	}
	mylog.Logger.Debug("no model map found", zap.String("model", model))
	return model
}

// GetModelRedirect 函数，根据model在ModelMap中查找对应的映射，如果找不到则返回原始model
func GetModelRedirect(s *ModelDetails, model string) string {
	if redirectModel, exists := s.ModelRedirect[model]; exists {
		mylog.Logger.Info("ModelRedirect model found", zap.String("model", model), zap.String("redirectModel", redirectModel))
		return redirectModel
	}
	mylog.Logger.Debug(" ModelRedirect no model found", zap.String("model", model))
	return model
}

// GetGlobalModelRedirect 函数，根据model在ModelMap中查找对应的映射，如果找不到则返回原始model
func GetGlobalModelRedirect(model string) string {
	if redirectModel, exists := GlobalModelRedirect[KEYNAME_ALL]; exists {
		if redirectModel == KEYNAME_ALL {
			redirectModel = KEYNAME_RANDOM
		}
		mylog.Logger.Info("GlobalModelRedirect model all found", zap.String("model", model), zap.String("redirectModel", redirectModel))
		return redirectModel
	}

	if redirectModel, exists := GlobalModelRedirect[model]; exists {
		mylog.Logger.Info("GlobalModelRedirect model found", zap.String("model", model), zap.String("redirectModel", redirectModel))
		return redirectModel
	}

	mylog.Logger.Debug(" GlobalModelRedirect no model found", zap.String("model", model))
	return model
}

func ShowSupportModels() {
	keys := make([]string, 0, len(ModelToService))

	for k := range SupportModels {
		keys = append(keys, k)
	}
	sort.Strings(keys) // 对keys进行排序

	log.Println("other support models:", keys)
}

func IsSupportMultiContent(model string) bool {
	for _, item := range SupportMultiContentModels {
		if strings.HasSuffix(item, "*") {
			prefix := strings.TrimSuffix(item, "*")
			if strings.HasPrefix(model, prefix) {
				return true
			}
		} else if item == model {
			return true
		}
	}
	return false
}

func IsProxyEnabled(s *ModelDetails) bool {
	switch GProxyConf.Strategy {
	case PROXY_STRATEGY_FORCEALL:
		// 配置全部启用代理，即使服务内配置了false，也忽略
		return true
	case PROXY_STRATEGY_ALL:
		// 配置全部启用代理，如果服务内配置了false，则不启动，其他情况全部启用
		if s.UseProxy == nil || (s.UseProxy != nil && *s.UseProxy) {
			return true
		}
	case PROXY_STRATEGY_DEFAULT:
		// 配置根据配置启用代理，默认是关闭
		if s.UseProxy != nil && *s.UseProxy {
			return true
		}
	case PROXY_STRATEGY_DISABLED:
		// 配置全部禁用代理
		return false
	default:
		return false
	}

	return false
}

func initAPIKeyMap() {
	apiKeyMap = make(map[string]APIKeyConfig)
	for _, keyConfig := range GSOAConf.APIKeys {
		apiKeyMap[keyConfig.APIKey] = keyConfig
	}
}

func ValidateAPIKeyAndModel(apikey string, model string) (bool, string) {
	if len(apiKeyMap) == 0 {
		return true, ""
	}
	keyConfig, exists := apiKeyMap[apikey]
	if !exists {
		mylog.Logger.Error("ValidateAPIKeyAndModel|Forbidden: invalid API key", zap.String("apikey", apikey))
		return false, "Forbidden: invalid API key"
	}

	mylog.Logger.Debug("ValidateAPIKeyAndModel", zap.String("model", model))

	// 检查所有服务和通配符的配置
	for service, models := range keyConfig.SupportedModels {
		mylog.Logger.Info(service, zap.Any("SupportedModels", models))
		for _, m := range models {
			if m == "*" || m == model {
				mylog.Logger.Debug("ValidateAPIKeyAndModel", zap.String("model", model), zap.String("m", m))
				return true, ""
			}
		}
	}
	return false, "Forbidden: model not supported"
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
