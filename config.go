package enjin

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Host     string `json:"host"`
	Port     uint16 `json:"port"`
	User     string `json:"user"`
	Password string `json:"passwd"`
	Database string `json:"db"`
}

func (c Config) GetDbUri() string {
	return fmt.Sprintf("neo4j://%s:%d", c.Host, c.Port)
}

func CreateConfig(filePath string) (conf Config, err error) {
	var file *os.File

	file, err = os.Open(filePath)
	defer file.Close()

	if err == nil {
		err = json.NewDecoder(file).Decode(&conf)
	}

	return conf, err
}
