package config

import (
	"encoding/json"
	"io/ioutil"
)

// Config structure for server
type Config struct {
	Port string `yaml:"port"`
}

// ParseJSONFile the config file
func ParseJSONFile(filename string, c *Config) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, c)
}
