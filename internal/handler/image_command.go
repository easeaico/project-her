package handler

import (
	"bytes"
	"context"
	"fmt"
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

// ImageService defines the interface for image generation services.
type ImageService interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// ImageCommandHandler processes the /image command before it reaches the LLM.
type ImageCommandHandler struct {
	character    *types.Character
	imageService ImageService
	templates    *template.Template
}

const (
	tplUsage   = "usage"
	tplError   = "error"
	tplSuccess = "success"
)

// defaultTemplates defines the default responses using character data.
var defaultTemplates = `
{{define "usage"}}{{.Name}} 眨眨眼说："想看什么图呀？比如：/image 一只在发呆的猫"{{end}}
{{define "error"}}{{.Name}} 挠挠头："哎呀，画笔断了（生成失败），稍后再试试吧。"{{end}}
{{define "success"}}
{{.Name}} 把画递给你："画好啦！"

![生成的图片]({{.URL}})
{{end}}
`

// NewImageCommandHandler creates a new ImageCommandHandler instance.
func NewImageCommandHandler(ctx context.Context, cfg *config.Config, character *types.Character) (*ImageCommandHandler, error) {
	imageService, err := models.NewGeminiImageGenerator(ctx, cfg.GoogleAPIKey, cfg.ImageModel, cfg.AspectRatio)
	if err != nil {
		return nil, fmt.Errorf("failed to create image generator: %w", err)
	}

	tmpl, err := template.New("handlers").Parse(defaultTemplates)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &ImageCommandHandler{
		character:    character,
		imageService: imageService,
		templates:    tmpl,
	}, nil
}

// Handle is an agent.BeforeAgentCallback that processes user commands.
func (h *ImageCommandHandler) Handle(cbCtx agent.CallbackContext) (*genai.Content, error) {
	userText := utils.ExtractContentText(cbCtx.UserContent())
	trimmed := strings.TrimSpace(userText)

	if trimmed == "/image" || strings.HasPrefix(trimmed, "/image ") {
		return h.processImageCommand(cbCtx, trimmed)
	}

	return nil, nil
}

func (h *ImageCommandHandler) processImageCommand(ctx context.Context, input string) (*genai.Content, error) {
	prompt := strings.TrimSpace(strings.TrimPrefix(input, "/image"))
	if prompt == "" {
		return h.renderResponse(tplUsage, nil)
	}

	promptTemplate, err := template.New("prompt").Parse(prompt)
	if err != nil {
		slog.Error("failed to parse image prompt template", "error", err.Error())
		return h.renderResponse(tplError, nil)
	}

	data := struct {
		CharacterName        string
		CharacterAppearance  string
		CharacterDescription string
		CharacterScenario    string
	}{
		CharacterName:        h.character.Name,
		CharacterAppearance:  h.character.Appearance,
		CharacterDescription: h.character.Description,
		CharacterScenario:    h.character.Scenario,
	}

	var buf bytes.Buffer
	if err := promptTemplate.Execute(&buf, data); err != nil {
		slog.Error("failed to build image prompt", "error", err.Error())
		return h.renderResponse(tplError, nil)
	}

	imageURL, err := h.imageService.Generate(ctx, buf.String())
	if err != nil {
		slog.Error("failed to generate image", "error", err.Error())
		return h.renderResponse(tplError, nil)
	}

	return h.renderResponse(tplSuccess, map[string]any{
		"URL": imageURL,
	})
}

func (h *ImageCommandHandler) renderResponse(name string, data map[string]any) (*genai.Content, error) {
	if data == nil {
		data = map[string]any{}
	}
	if h.character != nil {
		data["Name"] = h.character.Name
	} else {
		data["Name"] = "她"
	}

	var buf bytes.Buffer
	if err := h.templates.ExecuteTemplate(&buf, name, data); err != nil {
		slog.Error("failed to execute template", "template", name, "error", err.Error())
		// Fallback for safety
		return genai.NewContentFromText("处理您的请求时出现错误。", "model"), nil
	}

	return genai.NewContentFromText(buf.String(), "model"), nil
}
