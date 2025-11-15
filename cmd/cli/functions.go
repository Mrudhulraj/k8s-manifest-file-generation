package cli

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

var findSchemaNames openai.FunctionDefinition = openai.FunctionDefinition{
	Name: "findschemaNames",
	Description: "Get the Namespace Names for a Kubernetes resource",
	Parameters: jsonschema.Definition{
		Type: jsonSchema.Object,
		Properties: map[string]jsonschema.Definition{
			"resourceName": {
				Type: jsonschema.String,
				Description: "The name of the Kubernetes resource or field",
			},
		},
		Required: []string{"resourceName"},
	},

}
