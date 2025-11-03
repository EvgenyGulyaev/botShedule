package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/EvgenyGulyaev/botShedule/pkg/singleton"
	"github.com/joho/godotenv"
)

type Config struct {
	IsLoaded bool
	Env      map[string]string
}

func LoadConfig() *Config {
	return singleton.GetInstance("config", func() interface{} {
		return load()
	}).(*Config)
}

func load() *Config {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	rootPath := filepath.Dir(exePath)
	rootPath = filepath.Join(rootPath, "..", ".env")

	err = godotenv.Load(rootPath)
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	env, err := godotenv.Read(rootPath)
	if err != nil {
		log.Fatal("Error cannot read .env file")
	}
	return &Config{IsLoaded: true, Env: env}
}
