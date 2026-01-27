package callback

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"text/template"

	"google.golang.org/adk/agent"
	"google.golang.org/genai"

	"github.com/easeaico/project-her/internal/config"
	"github.com/easeaico/project-her/internal/models"
	"github.com/easeaico/project-her/internal/types"
	"github.com/easeaico/project-her/internal/utils"
)

const (
	tplUsage   = "usage"
	tplError   = "error"
	tplSuccess = "success"
)

// defaultTemplatesText 定义基于角色数据的默认响应模板。
var defaultTemplatesText = `
{{define "usage"}}{{.Name}} 眨眨眼说："想看什么图呀？比如：/image 一只在发呆的猫"{{end}}
{{define "error"}}{{.Name}} 挠挠头："哎呀，画笔断了（生成失败），稍后再试试吧。"{{end}}
{{define "success"}}
{{.Name}} 把画递给你："画好啦！"

![生成的图片]({{.URL}})
{{end}}
`
var defaultTemplates = template.Must(template.New("command").Parse(defaultTemplatesText))

// NewCommandCallback 创建 /image 命令的前置处理回调。
func NewCommandCallback(ctx context.Context, cfg *config.Config, character *types.Character) agent.BeforeAgentCallback {
	imageService, err := models.NewGeminiImageGenerator(ctx, cfg.GoogleAPIKey, cfg.ImageModel, cfg.AspectRatio)
	if err != nil {
		slog.Error("failed to create image generator", "error", err.Error())
		return nil
	}

	return func(cbCtx agent.CallbackContext) (*genai.Content, error) {
		userText := utils.ExtractContentText(cbCtx.UserContent())
		trimmed := strings.TrimSpace(userText)

		if trimmed == "/image" || strings.HasPrefix(trimmed, "/image ") {
			return processImageCommand(cbCtx, trimmed, character, imageService)
		}

		return nil, nil

	}
}

func processImageCommand(ctx context.Context, input string, character *types.Character, imageService *models.ImageGenerator) (*genai.Content, error) {
	prompt := strings.TrimSpace(strings.TrimPrefix(input, "/image"))
	if prompt == "" {
		return renderResponse(tplUsage, nil)
	}

	promptTemplate, err := template.New("prompt").Parse(prompt)
	if err != nil {
		slog.Error("failed to parse image prompt template", "error", err.Error())
		return renderResponse(tplError, nil)
	}

	data := struct {
		CharacterName        string
		CharacterAppearance  string
		CharacterDescription string
		CharacterScenario    string
	}{
		CharacterName:        character.Name,
		CharacterAppearance:  character.Appearance,
		CharacterDescription: character.Description,
		CharacterScenario:    character.Scenario,
	}

	var buf bytes.Buffer
	if err := promptTemplate.Execute(&buf, data); err != nil {
		slog.Error("failed to build image prompt", "error", err.Error())
		return renderErrorResponse(character)
	}

	imageURL, err := imageService.Generate(ctx, buf.String())
	if err != nil {
		slog.Error("failed to generate image", "error", err.Error())
		return renderErrorResponse(character)
	}

	return renderSuccessResponse(character, imageURL)
}

func renderSuccessResponse(character *types.Character, imageURL string) (*genai.Content, error) {
	return renderResponse(tplSuccess, map[string]any{
		"URL":  imageURL,
		"Name": character.Name,
	})
}

func renderErrorResponse(character *types.Character) (*genai.Content, error) {
	return renderResponse(tplError, map[string]any{
		"Name": character.Name,
	})
}

func renderResponse(tplName string, data map[string]any) (*genai.Content, error) {
	var buf bytes.Buffer
	if err := defaultTemplates.ExecuteTemplate(&buf, tplName, data); err != nil {
		slog.Error("failed to execute template", "template", tplName, "error", err.Error())
		// 模板渲染失败时返回兜底文本，避免中断对话。
		return genai.NewContentFromText("处理您的请求时出现错误。", "model"), nil
	}

	return genai.NewContentFromText(buf.String(), "model"), nil
}
