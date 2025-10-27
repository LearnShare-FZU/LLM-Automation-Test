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
	config       = &Config{}
	runtimeViper *viper.Viper
)

// Init ��ʼ������ģ��
func Init() {
	runtimeViper = viper.New()
	configPath := "./config.yaml"
	runtimeViper.SetConfigFile(configPath)
	runtimeViper.SetConfigType("yaml")

	// ��������ļ��Ƿ���ڣ�����������򴴽�Ĭ�������ļ�
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := createDefaultConfig(configPath); err != nil {
			fmt.Printf("config.Init: failed to create default config: %v", err)
			return
		}
		fmt.Printf("config.Init: default config file created")
		os.Exit(0)
	}

	// ��ȡ�����ļ�
	if err := runtimeViper.ReadInConfig(); err != nil {
		fmt.Printf("config.Init: config: read error: %v\n", err)
		return
	}
	configMapping()

	// ���������ļ��ı仯
	runtimeViper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Printf("config: notice config changed: %v\n", e.String())
		configMapping()
	})
	runtimeViper.WatchConfig()
}

// createDefaultConfig ����Ĭ�ϵ������ļ�
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
		TestRuns:    3,
		Timeout:     90 * time.Second,
	}

	v := viper.New()
	v.Set("DeepSeek", defaultConfig.DeepSeek)
	v.Set("Qwen", defaultConfig.Qwen)
	v.Set("Gemini", defaultConfig.Gemini)
	v.Set("GeminiProxy", defaultConfig.GeminiProxy)
	v.Set("MaxRounds", defaultConfig.MaxRounds)
	v.Set("TestRuns", defaultConfig.TestRuns)
	v.Set("Timeout", defaultConfig.Timeout)

	return v.WriteConfigAs(configPath)
}

// configMapping �������ļ�������ӳ�䵽ȫ�ֱ���
func configMapping() {
	c := &Config{}
	if err := runtimeViper.Unmarshal(c); err != nil {
		fmt.Printf("config.configMapping: config: unmarshal error: %v", err)
		return
	}

	*config = *c
}
