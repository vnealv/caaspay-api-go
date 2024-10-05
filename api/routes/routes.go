package routes

import (
    "io/ioutil"
    "log"
    "gopkg.in/yaml.v2"
)

// Define a wrapper to hold the routes key
type RoutesWrapper struct {
    Routes []RouteConfig `yaml:"routes"`
}

// RouteConfig defines the structure of a route
type RouteConfig struct {
    Path          string `yaml:"path"`
    Type          string `yaml:"type"`
    Authorization bool   `yaml:"authorization"`
    AuthType      string `yaml:"auth_type"` // jwt, oauth, cloudflare_jwt
    Role          string `yaml:"role"`      // For RBAC
}

// LoadRoutes reads and parses the YAML file to return a list of routes
func LoadRoutes() ([]RouteConfig, error) {
    // Read YAML file
    data, err := ioutil.ReadFile("config/routes.yaml")
    if err != nil {
 	   log.Printf("Error reading routes file: %v", err)
 	   return nil, err
    }

    // Parse YAML data
    var wrapper RoutesWrapper
    if err := yaml.Unmarshal(data, &wrapper); err != nil {
 	   log.Printf("Error parsing YAML: %v", err)
 	   return nil, err
    }

    return wrapper.Routes, nil
}

