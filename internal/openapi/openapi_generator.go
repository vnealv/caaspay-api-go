package openapi

import (
	"caaspay-api-go/api/config"
	"caaspay-api-go/api/routes"
	"fmt"
)

type OpenAPISpec struct {
	OpenAPI    string              `json:"openapi"`
	Info       Info                `json:"info"`
	Paths      map[string]PathItem `json:"paths"`
	Components Components          `json:"components"`
}

type Info struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type PathItem struct {
	Get  *Operation `json:"get,omitempty"`
	Post *Operation `json:"post,omitempty"`
}

type Operation struct {
	Summary     string                `json:"summary"`
	Description string                `json:"description"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses"`
	Security    []map[string][]string `json:"security,omitempty"`
}

type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	Schema      Schema `json:"schema"`
}

type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Required    bool                 `json:"required"`
	Content     map[string]MediaType `json:"content"`
}

type MediaType struct {
	Schema Schema `json:"schema"`
}

type Schema struct {
	Type       string            `json:"type"`
	Properties map[string]Schema `json:"properties,omitempty"`
}

type Response struct {
	Description string      `json:"description"`
	Content     interface{} `json:"content,omitempty"`
}

type Components struct {
	Schemas         map[string]interface{}    `json:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
}

type SecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme"`
	BearerFormat string `json:"bearerFormat,omitempty"`
}

// GenerateOpenAPISpec generates an OpenAPI spec from the route configuration.
func GenerateOpenAPISpec(routeConfigs []routes.RouteConfig, cfg *config.Config) (*OpenAPISpec, error) {
	openAPISpec := &OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:       "CaasPay API",
			Description: "API documentation for CaasPay",
			Version:     "1.0.0",
		},
		Paths: make(map[string]PathItem),
		Components: Components{
			Schemas: make(map[string]interface{}),
			SecuritySchemes: map[string]SecurityScheme{
				"BearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
				},
			},
		},
	}

	// Process each route configuration
	for _, route := range routeConfigs {
		pathItem := PathItem{}
		operation := Operation{
			Summary:     fmt.Sprintf("Operation for %s", route.Path),
			Description: route.Description,
			Responses: map[string]Response{
				"200": {
					Description: "Successful response",
				},
			},
		}

		// Add requestBody for POST with parameters
		if route.Type == "POST" && len(route.Params) > 0 {
			properties := make(map[string]Schema)
			for _, param := range route.Params {
				properties[param.Name] = Schema{Type: param.Type}
			}

			requestBody := RequestBody{
				Description: "Request body parameters",
				Required:    true,
				Content: map[string]MediaType{
					"application/json": {
						Schema: Schema{
							Type:       "object",
							Properties: properties,
						},
					},
				},
			}
			operation.RequestBody = &requestBody
		} else {
			// For GET requests, use URL parameters
			for _, param := range route.Params {
				operation.Parameters = append(operation.Parameters, Parameter{
					Name:        param.Name,
					In:          "query",
					Description: param.Description,
					Required:    param.Required,
					Schema:      Schema{Type: param.Type},
				})
			}
		}

		// Assign to correct HTTP method
		switch route.Type {
		case "GET":
			pathItem.Get = &operation
		case "POST":
			pathItem.Post = &operation
		}

		// Apply security for routes requiring authorization
		if route.Authorization {
			operation.Security = []map[string][]string{
				{"BearerAuth": {}},
			}
		}

		openAPISpec.Paths[route.Path] = pathItem
	}

	addStaticRouteDocs(openAPISpec, cfg)
	return openAPISpec, nil
}

// Add documentation for static routes (health, status, JWT) if enabled in config.
func addStaticRouteDocs(openAPISpec *OpenAPISpec, cfg *config.Config) {
	if cfg.HealthRouteEnabled {
		openAPISpec.Paths["/health"] = PathItem{
			Get: &Operation{
				Summary:     "Health Check",
				Description: "API health check",
				Responses: map[string]Response{
					"200": {Description: "Service is healthy"},
				},
			},
		}
	}

	if cfg.StatusRouteEnabled {
		openAPISpec.Paths["/status"] = PathItem{
			Get: &Operation{
				Summary:     "Status Check",
				Description: "API status check",
				Responses: map[string]Response{
					"200": {Description: "Service is operational"},
				},
			},
		}
	}

	if cfg.SelfJWTEnabled {
		openAPISpec.Paths["/jwt/login"] = PathItem{
			Post: &Operation{
				Summary:     "JWT Login",
				Description: "Authenticate and obtain a JWT",
				RequestBody: &RequestBody{
					Description: "Login credentials",
					Required:    true,
					Content: map[string]MediaType{
						"application/json": {
							Schema: Schema{
								Type: "object",
								Properties: map[string]Schema{
									"username": {Type: "string"},
									"password": {Type: "string"},
								},
							},
						},
					},
				},
				Responses: map[string]Response{
					"200": {Description: "JWT token generated"},
				},
			},
		}

		openAPISpec.Paths["/jwt/renew"] = PathItem{
			Post: &Operation{
				Summary:     "JWT Renewal",
				Description: "Renew an existing JWT",
				RequestBody: &RequestBody{
					Description: "JWT renewal token",
					Required:    true,
					Content: map[string]MediaType{
						"application/json": {
							Schema: Schema{
								Type: "object",
								Properties: map[string]Schema{
									"token": {Type: "string"},
								},
							},
						},
					},
				},
				Responses: map[string]Response{
					"200": {Description: "JWT token renewed"},
				},
			},
		}
	}
}
