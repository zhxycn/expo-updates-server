package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Host string `envconfig:"HOST" default:"0.0.0.0"`
	Port string `envconfig:"PORT" default:"8080"`

	Hostname string `envconfig:"HOSTNAME" default:"http://localhost:8080"`

	StorageType string `envconfig:"STORAGE_TYPE" default:"local"`

	StorageDir   string `envconfig:"STORAGE_DIR" default:"./data/updates"`
	DatabasePath string `envconfig:"DATABASE_PATH" default:"./data/ota.db"`

	S3Endpoint  string `envconfig:"S3_ENDPOINT"`
	S3Bucket    string `envconfig:"S3_BUCKET"`
	S3Region    string `envconfig:"S3_REGION"`
	S3AccessKey string `envconfig:"S3_ACCESS_KEY"`
	S3SecretKey string `envconfig:"S3_SECRET_KEY"`

	PrivateKey string `envconfig:"PRIVATE_KEY"`
	JWTSecret  string `envconfig:"JWT_SECRET"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config
	err := envconfig.Process("", &cfg)
	return &cfg, err
}
