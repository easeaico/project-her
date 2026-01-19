package models

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/openai/openai-go/v3"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// buildOpenAIParams converts ADK request to OpenAI parameters
func buildOpenAIParams(req *model.LLMRequest, model string) *openai.ChatCompletionNewParams {
	params := openai.ChatCompletionNewParams{
		Model: req.Model,
	}
	if req.Model == "" {
		params.Model = model
	}

	messages := convertContentsToMessages(req.Contents)
	if len(messages) > 0 {
		params.Messages = messages
	}

	if req.Config != nil {
		if req.Config.Temperature != nil {
			params.Temperature = openai.Float(float64(*req.Config.Temperature))
		}
		if req.Config.MaxOutputTokens > 0 {
			params.MaxTokens = openai.Int(int64(req.Config.MaxOutputTokens))
		}
		if req.Config.TopP != nil {
			params.TopP = openai.Float(float64(*req.Config.TopP))
		}

		// 将LLMRequest中的Tools转换为OpenAI的function_call参数
		if len(req.Config.Tools) > 0 {
			tools := convertToolsToOpenAI(req.Config.Tools)
			if len(tools) > 0 {
				params.Tools = tools
			}
		}
	}

	return &params
}

// convertToolsToOpenAI converts LLMRequest.Config.Tools to OpenAI tools format
func convertToolsToOpenAI(toolsMap []*genai.Tool) []openai.ChatCompletionToolUnionParam {
	var tools []openai.ChatCompletionToolUnionParam

	for _, t := range toolsMap {
		// 转换 FunctionDeclarations
		for _, fn := range t.FunctionDeclarations {
			// Convert function parameters
			parameters := convertFunctionParameters(fn)
			tool := openai.ChatCompletionToolUnionParam{
				OfFunction: &openai.ChatCompletionFunctionToolParam{
					Function: openai.FunctionDefinitionParam{
						Name:        fn.Name,
						Description: openai.String(fn.Description),
						Parameters:  parameters,
					},
				},
			}

			tools = append(tools, tool)
		}
	}

	return tools
}

// convertFunctionParameters converts genai.FunctionDeclaration parameters to OpenAI FunctionParameters format
func convertFunctionParameters(fn *genai.FunctionDeclaration) openai.FunctionParameters {
	// Priority 1: Use ParametersJsonSchema if available (already in JSON Schema format)
	if fn.ParametersJsonSchema != nil {
		if schema, ok := fn.ParametersJsonSchema.(*jsonschema.Schema); ok {
			return convertSchemaToJSONSchema(schema)
		}
		// Try to handle other types that might be used
		if schemaMap, ok := fn.ParametersJsonSchema.(map[string]any); ok {
			return openai.FunctionParameters(schemaMap)
		}
	}

	// Priority 2: Fallback to Parameters field if available
	// Note: This would require converting genai.Schema to jsonschema.Schema
	// For now, return nil if ParametersJsonSchema is not available
	return nil
}

// convertSchemaToJSONSchema converts jsonschema.Schema to JSON Schema format
func convertSchemaToJSONSchema(schema *jsonschema.Schema) openai.FunctionParameters {
	result := make(map[string]any)

	// Set type
	if schema.Type != "" {
		result["type"] = string(schema.Type)
	} else {
		result["type"] = "object" // Default to object for function parameters
	}

	// Convert properties
	if len(schema.Properties) > 0 {
		properties := make(map[string]any)
		for name, propSchema := range schema.Properties {
			if propSchema != nil {
				properties[name] = convertSchemaProperty(propSchema)
			}
		}
		if len(properties) > 0 {
			result["properties"] = properties
		}
	}

	// Set required fields
	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	} else {
		result["required"] = []string{}
	}

	return openai.FunctionParameters(result)
}

// convertSchemaProperty converts a single jsonschema.Schema property to JSON Schema format
func convertSchemaProperty(schema *jsonschema.Schema) map[string]any {
	if schema == nil {
		return nil
	}

	prop := make(map[string]any)

	// Handle type - support both single and multiple types
	if len(schema.Types) > 0 {
		// Multiple types - use first type (OpenAI typically supports single type)
		prop["type"] = schema.Types[0]
	} else if schema.Type != "" {
		prop["type"] = schema.Type
	}

	if schema.Description != "" {
		prop["description"] = schema.Description
	}

	if schema.Format != "" {
		prop["format"] = schema.Format
	}

	// Handle enum
	if len(schema.Enum) > 0 {
		prop["enum"] = schema.Enum
	}

	// Handle const
	if schema.Const != nil {
		prop["const"] = *schema.Const
	}

	// Handle default value - parse json.RawMessage
	if len(schema.Default) > 0 {
		var defaultVal any
		if err := json.Unmarshal(schema.Default, &defaultVal); err == nil {
			prop["default"] = defaultVal
		}
	}

	// Handle numeric constraints
	if schema.Minimum != nil {
		prop["minimum"] = *schema.Minimum
	}
	if schema.Maximum != nil {
		prop["maximum"] = *schema.Maximum
	}
	if schema.ExclusiveMinimum != nil {
		prop["exclusiveMinimum"] = *schema.ExclusiveMinimum
	}
	if schema.ExclusiveMaximum != nil {
		prop["exclusiveMaximum"] = *schema.ExclusiveMaximum
	}

	// Handle string constraints
	if schema.MinLength != nil {
		prop["minLength"] = *schema.MinLength
	}
	if schema.MaxLength != nil {
		prop["maxLength"] = *schema.MaxLength
	}
	if schema.Pattern != "" {
		prop["pattern"] = schema.Pattern
	}

	// Handle items for array types
	if schema.Items != nil {
		prop["items"] = convertSchemaProperty(schema.Items)
	}

	// Handle properties for object types
	if len(schema.Properties) > 0 {
		properties := make(map[string]any)
		for name, propSchema := range schema.Properties {
			if propSchema != nil {
				properties[name] = convertSchemaProperty(propSchema)
			}
		}
		if len(properties) > 0 {
			prop["properties"] = properties
		}
	}

	// Handle required for nested objects - only include if non-empty
	if len(schema.Required) > 0 {
		prop["required"] = schema.Required
	}

	return prop
}

// convertContentsToMessages converts genai.Content to OpenAI messages
func convertContentsToMessages(contents []*genai.Content) []openai.ChatCompletionMessageParamUnion {
	var messages []openai.ChatCompletionMessageParamUnion

	for _, content := range contents {
		// check if has function response
		var hasFunctionResponse bool
		for _, part := range content.Parts {
			if part.FunctionResponse != nil && part.FunctionResponse.ID != "" {
				hasFunctionResponse = true
			}
		}

		// convert function response to tool message
		if hasFunctionResponse {
			for _, part := range content.Parts {
				if part.FunctionResponse != nil && part.FunctionResponse.ID != "" {
					message, err := json.Marshal(part.FunctionResponse.Response)
					if err != nil {
						slog.Error("failed to marshal function response", "error", err.Error())
						continue
					}

					toolMessage := openai.ToolMessage(string(message), part.FunctionResponse.ID)
					messages = append(messages, toolMessage)
				}
			}

			continue
		}

		// Extract text from parts using strings.Builder for better performance
		var sb strings.Builder
		for _, part := range content.Parts {
			if part.Text != "" {
				sb.WriteString(part.Text)
			}
		}
		textContent := sb.String()

		switch content.Role {
		case "user":
			messages = append(messages, openai.UserMessage(textContent))
		case "model":
			messages = append(messages, openai.AssistantMessage(textContent))
		case "system":
			messages = append(messages, openai.SystemMessage(textContent))
		default:
			messages = append(messages, openai.UserMessage(textContent))
		}
	}

	return messages
}
