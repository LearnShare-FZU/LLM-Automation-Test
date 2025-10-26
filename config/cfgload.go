package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	config       = &Config{} // 初始化全局配置变量
	runtimeViper *viper.Viper
)

// Init 初始化配置模块
func Init() {
	runtimeViper = viper.New()
	configPath := "./config.yaml"
	runtimeViper.SetConfigFile(configPath)
	runtimeViper.SetConfigType("yaml")

	// 检查配置文件是否存在，如果不存在则创建默认配置文件
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := createDefaultConfig(configPath); err != nil {
			fmt.Printf("config.Init: failed to create default config: %v", err)
			return
		}
		fmt.Printf("config.Init: default config file created")
		os.Exit(0)
	}

	// 读取配置文件
	if err := runtimeViper.ReadInConfig(); err != nil {
		fmt.Printf("config.Init: config: read error: %v\n", err)
		return
	}
	configMapping()

	// 监听配置文件的变化
	runtimeViper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Printf("config: notice config changed: %v\n", e.String())
		configMapping()
	})
	runtimeViper.WatchConfig()
}

// createDefaultConfig 创建默认的配置文件
func createDefaultConfig(configPath string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	defaultConfig := Config{
		DeepSeek: AIConfig{
			Model:   "",
			APIKey:  "",
			BaseURL: "",
		},
		Qwen: AIConfig{
			Model:   "",
			APIKey:  "",
			BaseURL: "",
		},
		Gemini: AIConfig{
			Model:   "",
			APIKey:  "",
			BaseURL: "",
		},
		GeminiProxy: "http://127.0.0.1:7890",
		MaxRounds:   20,
		Timeout:     90 * time.Second,
	}

	v := viper.New()
	v.Set("DeepSeek", defaultConfig.DeepSeek)
	v.Set("Qwen", defaultConfig.Qwen)
	v.Set("Gemini", defaultConfig.Gemini)
	v.Set("GeminiProxy", defaultConfig.GeminiProxy)
	v.Set("MaxRounds", defaultConfig.MaxRounds)
	v.Set("Timeout", defaultConfig.Timeout)

	return v.WriteConfigAs(configPath)
}

// configMapping 将配置文件的内容映射到全局变量
func configMapping() {
	c := &Config{}
	if err := runtimeViper.Unmarshal(c); err != nil {
		fmt.Printf("config.configMapping: config: unmarshal error: %v", err)
		return
	}

	*config = *c
}
