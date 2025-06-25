package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestWaitForFileReadable(t *testing.T) {
	// 创建一个临时文件
	tmpFile, err := os.CreateTemp("", "test_config_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// 测试文件可读
	err = waitForFileReadable(tmpFile.Name(), 5*time.Second)
	if err != nil {
		t.Errorf("Expected file to be readable, got error: %v", err)
	}
}

func TestWaitForFileReadableTimeout(t *testing.T) {
	// 测试不存在的文件
	err := waitForFileReadable("nonexistent_file.json", 1*time.Second)
	if err == nil {
		t.Error("Expected timeout error for nonexistent file")
	}
}

func TestLoadConfiguration(t *testing.T) {
	// 创建测试配置文件
	testConfig := `{
		"server_port": ":8080",
		"debug": true,
		"log_level": "debug",
		"load_balancing": "random",
		"api_key": "test-key",
		"services": {
			"test_service": [
				{
					"provider": "test",
					"models": ["test-model"],
					"enabled": true,
					"credentials": {
						"api_key": "test-credential"
					}
				}
			]
		}
	}`

	tmpFile, err := os.CreateTemp("", "test_config_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// 写入测试配置
	_, err = tmpFile.WriteString(testConfig)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	tmpFile.Close()

	// 初始化viper
	v = viper.New()
	v.SetConfigFile(tmpFile.Name())
	v.SetConfigType("json")

	// 读取配置
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	// 加载配置
	conf, err := loadConfiguration()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// 验证配置
	if conf.ServerPort != ":8080" {
		t.Errorf("Expected server_port to be :8080, got %s", conf.ServerPort)
	}
	if !conf.Debug {
		t.Error("Expected debug to be true")
	}
	if conf.LogLevel != "debug" {
		t.Errorf("Expected log_level to be debug, got %s", conf.LogLevel)
	}
	if conf.LoadBalancing != "random" {
		t.Errorf("Expected load_balancing to be random, got %s", conf.LoadBalancing)
	}
	if conf.APIKey != "test-key" {
		t.Errorf("Expected api_key to be test-key, got %s", conf.APIKey)
	}

	// 验证服务配置
	services, exists := conf.Services["test_service"]
	if !exists {
		t.Error("Expected test_service to exist in services")
	}
	if len(services) == 0 {
		t.Error("Expected test_service to have at least one service model")
	}
	if services[0].Provider != "test" {
		t.Errorf("Expected provider to be test, got %s", services[0].Provider)
	}
	if len(services[0].Models) == 0 || services[0].Models[0] != "test-model" {
		t.Errorf("Expected models to contain test-model, got %v", services[0].Models)
	}
	if !services[0].Enabled {
		t.Error("Expected service to be enabled")
	}
}

func TestApplyConfiguration(t *testing.T) {
	// 创建测试配置
	conf := &Configuration{
		ServerPort:    ":9090",
		Debug:         true,
		LogLevel:      "info",
		LoadBalancing: "round_robin",
		APIKey:        "test-api-key",
		Services: map[string][]ServiceModel{
			"test_service": {
				{
					Provider: "test",
					Models:   []string{"test-model"},
					Enabled:  true,
				},
			},
		},
	}

	// 应用配置
	applyConfiguration(conf)

	// 验证全局变量
	if LoadBalancingStrategy != "round_robin" {
		t.Errorf("Expected LoadBalancingStrategy to be round_robin, got %s", LoadBalancingStrategy)
	}
	if ServerPort != ":9090" {
		t.Errorf("Expected ServerPort to be :9090, got %s", ServerPort)
	}
	if !Debug {
		t.Error("Expected Debug to be true")
	}
	if LogLevel != "info" {
		t.Errorf("Expected LogLevel to be info, got %s", LogLevel)
	}
	if APIKey != "test-api-key" {
		t.Errorf("Expected APIKey to be test-api-key, got %s", APIKey)
	}

	// 验证模型映射
	if len(ModelToService) == 0 {
		t.Error("Expected ModelToService to be populated")
	}
	modelDetails, exists := ModelToService["test-model"]
	if !exists {
		t.Error("Expected test-model to exist in ModelToService")
	}
	if len(modelDetails) == 0 {
		t.Error("Expected test-model to have at least one service detail")
	}
	if modelDetails[0].ServiceName != "test_service" {
		t.Errorf("Expected service name to be test_service, got %s", modelDetails[0].ServiceName)
	}
}
