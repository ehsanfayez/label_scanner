package config

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	ServerConfig    ServerConfig
	AuthConfig      AuthConfig
	OCRConfig       OCRConfig
	EmbeddingConfig EmbeddingConfig
}

type ServerConfig struct {
	App     string
	Port    string
	Proxy   string
	BaseUrl string
}

type AuthConfig struct {
	Username       string
	Password       string
	PrivateKeySeed string
	ExpiresIn      time.Duration
}

type OCRConfig struct {
	APIURL    string
	APIKey    string
	APIHeader string
}

type EmbeddingConfig struct {
	VectorsFile  string
	OpenaiApiKey string
}

var (
	cfg  *Config
	once sync.Once
)

func InitConfig() *Config {
	once.Do(func() {
		viper.SetConfigFile(".env")
		viper.AutomaticEnv()

		if err := viper.ReadInConfig(); err != nil {
			log.Printf("Error reading config file: %v", err)
		}

		server := &ServerConfig{
			App:     viper.GetString("APP"),
			Port:    viper.GetString("PORT"),
			Proxy:   viper.GetString("PROXY_URL"),
			BaseUrl: viper.GetString("BASE_URL"),
		}

		auth := &AuthConfig{
			Username:       viper.GetString("USERNAME"),
			Password:       viper.GetString("PASSWORD"),
			PrivateKeySeed: viper.GetString("PRIVATE_KEY_SEED"),
			ExpiresIn:      viper.GetDuration("EXPIRES_IN"),
		}

		ocr := &OCRConfig{
			APIURL: viper.GetString("OCR_API_URL"),
		}

		embedding := &EmbeddingConfig{
			VectorsFile:  viper.GetString("VECTORS_FILE"),
			OpenaiApiKey: viper.GetString("OPENAI_API_KEY"),
		}

		cfg = &Config{
			ServerConfig:    *server,
			AuthConfig:      *auth,
			OCRConfig:       *ocr,
			EmbeddingConfig: *embedding,
		}

		fmt.Println("Config initialized successfully")
	})

	return cfg
}

func GetConfig() *Config {
	if cfg == nil {
		InitConfig()
	}

	return cfg
}
