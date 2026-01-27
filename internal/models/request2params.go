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

// buildOpenAIParams 将 ADK 请求映射为 OpenAI 请求参数。
func buildOpenAIParams(req *model.LLMRequest, model string) *openai.ChatCompletionNewParams {
	params := openai.ChatCompletionNewParams{
		Model: req.Model,
	}

	if req.Model == "" {
		params.Model = model
	}

	if req.Config != nil {
		if req.Config.SystemInstruction != nil {
			if len(req.Config.SystemInstruction.Parts) > 0 && req.Config.SystemInstruction.Parts[0].Text != "" {
				systemMessage := openai.SystemMessage(req.Config.SystemInstruction.Parts[0].Text)
				params.Messages = append(params.Messages, systemMessage)
			}
		}
		if req.Config.Temperature != nil {
			params.Temperature = openai.Float(float64(*req.Config.Temperature))
		}
		if req.Config.MaxOutputTokens > 0 {
			params.MaxTokens = openai.Int(int64(req.Config.MaxOutputTokens))
		}
		if req.Config.TopP != nil {
			params.TopP = openai.Float(float64(*req.Config.TopP))
		}

		if len(req.Config.Tools) > 0 {
			tools := convertToolsToOpenAI(req.Config.Tools)
			if len(tools) > 0 {
				params.Tools = tools
			}
		}
	}

	messages := convertContentsToMessages(req.Contents)
	if len(messages) > 0 {
		params.Messages = append(params.Messages, messages...)
	}

	return &params
}

// convertToolsToOpenAI 将 genai.Tool 映射为 OpenAI 的工具定义。
func convertToolsToOpenAI(toolsMap []*genai.Tool) []openai.ChatCompletionToolUnionParam {
	var tools []openai.ChatCompletionToolUnionParam

	for _, t := range toolsMap {
		for _, fn := range t.FunctionDeclarations {
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

// convertFunctionParameters 将函数参数映射为 OpenAI 的 JSON Schema。
func convertFunctionParameters(fn *genai.FunctionDeclaration) openai.FunctionParameters {
	if fn.ParametersJsonSchema != nil {
		if schema, ok := fn.ParametersJsonSchema.(*jsonschema.Schema); ok {
			return convertSchemaToJSONSchema(schema)
		}
		if schemaMap, ok := fn.ParametersJsonSchema.(map[string]any); ok {
			return openai.FunctionParameters(schemaMap)
		}
	}

	return nil
}

// convertSchemaToJSONSchema 将 jsonschema.Schema 转换为 JSON Schema 映射。
func convertSchemaToJSONSchema(schema *jsonschema.Schema) openai.FunctionParameters {
	result := make(map[string]any)

	if schema.Type != "" {
		result["type"] = string(schema.Type)
	} else {
		result["type"] = "object"
	}

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

	// 保证 required 字段存在，便于下游一致处理。
	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	} else {
		result["required"] = []string{}
	}

	return openai.FunctionParameters(result)
}

// convertSchemaProperty 将单个 schema 属性转换为 JSON Schema 映射。
func convertSchemaProperty(schema *jsonschema.Schema) map[string]any {
	if schema == nil {
		return nil
	}

	prop := make(map[string]any)

	if len(schema.Types) > 0 {
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

	if len(schema.Enum) > 0 {
		prop["enum"] = schema.Enum
	}

	if schema.Const != nil {
		prop["const"] = *schema.Const
	}

	if len(schema.Default) > 0 {
		var defaultVal any
		if err := json.Unmarshal(schema.Default, &defaultVal); err == nil {
			prop["default"] = defaultVal
		}
	}

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

	if schema.MinLength != nil {
		prop["minLength"] = *schema.MinLength
	}
	if schema.MaxLength != nil {
		prop["maxLength"] = *schema.MaxLength
	}
	if schema.Pattern != "" {
		prop["pattern"] = schema.Pattern
	}

	if schema.Items != nil {
		prop["items"] = convertSchemaProperty(schema.Items)
	}

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

	if len(schema.Required) > 0 {
		prop["required"] = schema.Required
	}

	return prop
}

// convertContentsToMessages 将 genai.Content 转换为 OpenAI 消息序列。
func convertContentsToMessages(contents []*genai.Content) []openai.ChatCompletionMessageParamUnion {
	var messages []openai.ChatCompletionMessageParamUnion

	for _, content := range contents {
		var hasFunctionResponse bool
		for _, part := range content.Parts {
			if part.FunctionResponse != nil && part.FunctionResponse.ID != "" {
				hasFunctionResponse = true
			}
		}

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
