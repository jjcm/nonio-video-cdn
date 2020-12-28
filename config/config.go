package config

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// server fields
type serverStruct struct {
	Port string `yaml:"port"`
}

// Config structure for server
type Config struct {
	Server serverStruct `yaml:"server"`
}

// ParseYamlFile the config file
func ParseYamlFile(filename string, c *Config) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, c)
}
