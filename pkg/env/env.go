package env

import (
	"os"

	"github.com/joho/godotenv"
)

var defaultEnv = ".env"

func ReadLocalEnv() (map[string]string, error) {
	file, err := os.OpenFile(defaultEnv, os.O_RDONLY|os.O_CREATE, 0666)

	if err != nil {
		return nil, err
	}

	return godotenv.Parse(file)
}

func LoadLocalEnv() error {
	return godotenv.Load(defaultEnv)
}
