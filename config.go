package enjin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// A Config is a struct that has all the information needed to
type Config struct {
	Host     string `json:"host"` // the host of the database
	Port     uint16 `json:"port"` // the port of the database
    // The User to use for the database authentication
	User     string `json:"user"`
    // The Password to use for the database authentication
	Password string `json:"passwd"`
	Database string `json:"db"` // the database name
}

func (c Config) GetDbUri() string {
	return fmt.Sprintf("neo4j://%s:%d", c.Host, c.Port)
}

func (c Config) GetDriver() (*neo4j.DriverWithContext, error) {
	auth := neo4j.BasicAuth(c.User, c.Password, "")

	driver, err := neo4j.NewDriverWithContext(c.GetDbUri(), auth)

	return &driver, err
}

func (c Config) CreateProxy() *DriverProxy {
	driver, err := c.GetDriver()

	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	return &DriverProxy{&ctx, driver, &c.Database}
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
