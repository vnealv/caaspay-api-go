package openapi

import (
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
	// Add other HTTP methods as needed
}

type Operation struct {
	Summary     string              `json:"summary"`
	Description string              `json:"description"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	Responses   map[string]Response `json:"responses"`
}

type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	Schema      Schema `json:"schema"`
}

type Schema struct {
	Type string `json:"type"`
}

type Response struct {
	Description string      `json:"description"`
	Content     interface{} `json:"content,omitempty"`
}

type Components struct {
	Schemas map[string]interface{} `json:"schemas,omitempty"`
}

// GenerateOpenAPISpec generates an OpenAPI spec from the route configuration.
func GenerateOpenAPISpec(routeConfigs []routes.RouteConfig) (*OpenAPISpec, error) {
	openAPISpec := &OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:       "CaasPay API",
			Description: "API documentation for CaasPay",
			Version:     "1.0.0",
		},
		Paths:      make(map[string]PathItem),
		Components: Components{Schemas: make(map[string]interface{})},
	}

	for _, route := range routeConfigs {
		pathItem := PathItem{}
		operation := Operation{
			Summary:     fmt.Sprintf("Operation for %s", route.Path),
			Description: route.Description, // Ensure Description field is added to RouteConfig if used
			Parameters:  []Parameter{},
			Responses: map[string]Response{
				"200": {
					Description: "Successful response",
				},
			},
		}

		// Add parameters to operation
		for _, param := range route.Params {
			operation.Parameters = append(operation.Parameters, Parameter{
				Name:        param.Name,
				In:          "query", // Adjust this to fit query, path, or body based on your needs
				Description: param.Description,
				Required:    param.Required,
				Schema:      Schema{Type: param.Type},
			})
		}

		// Assign the operation to the corresponding HTTP method
		switch route.Type {
		case "GET":
			pathItem.Get = &operation
		case "POST":
			pathItem.Post = &operation
		}

		// Add the path item to the spec
		openAPISpec.Paths[route.Path] = pathItem
	}

	return openAPISpec, nil
}
