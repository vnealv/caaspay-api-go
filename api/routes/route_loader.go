package routes

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// LoadRouteConfigs loads the route configurations from a YAML file
func LoadRouteConfigs(filename string) ([]RouteConfig, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config struct {
		Routes []RouteConfig `yaml:"routes"`
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return config.Routes, nil
}
