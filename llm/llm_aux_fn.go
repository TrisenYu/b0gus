package llm

/// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License
/// Last modified at 2026/05/12 星期二 12:29:13

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"

	"b0gus/assets"
	"b0gus/configs"
	"b0gus/internal/misc_utils"
)

// configuration of LLM should be accessed from config.toml
// including API and which content to interact with
// https://github.com/0x4D31/galah/blob/main/pkg/llm/llm.go

type (
	LLMtaskType     int
	llmProviderType int
)

const (
	ValidateCmd LLMtaskType = iota
	ResponseCmd
	Translation
)

const (
	providerUnk llmProviderType = iota
	providerOllama
	providerOpenAI
	providerGoogle
	providerAmazon
	providerAzure
	providerAnthropic
	providerDeepSeek
	providerMoonshot
	providerByteDance
	providerAlibaba
)

var (
	llmTaskDispatcher = map[LLMtaskType]string{
		ValidateCmd: assets.RequestToValidateCmd,
		ResponseCmd: assets.RequestToResponseCmd,
		Translation: assets.RequestToTranslate,
	}
)

// Wrap will convert the current input string `i` into
// `[ProtocolSt] i [ProtocolEd]`, whereas both [ProtocolSt] and [ProtocolEd]
// are defined by specific protocol.
func (i InputForLLM) Wrap(ProtocolSt, ProtocolEd string) string {
	var sb strings.Builder
	sb.WriteString(ProtocolSt)
	sb.WriteString(string(i))
	sb.WriteString(ProtocolEd)
	return sb.String()
}

type HttpHeaderSetAuthFn func(*http.Request, string)

func (l *LLMcli) AlterPrompt(taskType LLMtaskType) {
	content, ok := llmTaskDispatcher[taskType]
	if !ok {
		payload := configs.GetLocalizedMsg(
			"llm.UnableToSetPromptDueToUnkCurrTaskType",
			map[string]any{"TaskType": strconv.Itoa(int(taskType))},
		)
		configs.Logger().Warn(payload)
		return
	}
	l.RolePrompt = content
}

func (l *LLMcli) initCheck() error {
	switch {
	case len(l.BaseURL) == 0:
		return errors.New(configs.GetLocalizedMsg("llm.EmptyBaseURLErr", nil))
	case len(l.APIKey) == 0:
		return errors.New(configs.GetLocalizedMsg("llm.EmptyAPIKeyErr", nil))
	case len(l.ModelName) == 0:
		return errors.New(configs.GetLocalizedMsg("llm.EmptyModelNameErr", nil))
	case len(l.RolePrompt) == 0:
		return errors.New(configs.GetLocalizedMsg("llm.EmptyRolePromptErr", nil))
	}
	return nil
}
func (l *LLMcli) initOpenAI(providerType llmProviderType) error {
	l.providerType = providerType
	if err := l.initCheck(); err != nil {
		return err
	}
	opts := []openai.Option{
		openai.WithBaseURL(l.BaseURL),
		openai.WithModel(l.ModelName),
		openai.WithToken(l.APIKey),
	}
	m, err := openai.New(opts...)
	if err != nil {
		return err
	}
	l.mu.Lock()
	l.model = m
	l.mu.Unlock()
	return nil
}
func (l *LLMcli) initOllama() error {
	if err := l.initCheck(); err != nil {
		return err
	}
	opts := []ollama.Option{
		ollama.WithServerURL(l.BaseURL),
		ollama.WithModel(l.ModelName),
	}
	m, err := ollama.New(opts...)
	if err != nil {
		return err
	}
	l.mu.Lock()
	l.model = m
	l.mu.Unlock()
	return nil
}

func (l *LLMcli) initAnthropic() error {
	if err := l.initCheck(); err != nil {
		return err
	}
	opts := []anthropic.Option{
		anthropic.WithModel(l.ModelName),
		anthropic.WithBaseURL(l.BaseURL),
		anthropic.WithToken(l.APIKey),
	}
	m, err := anthropic.New(opts...)
	if err != nil {
		return err
	}
	l.mu.Lock()
	l.model = m
	l.mu.Unlock()
	return nil
}

func (l *LLMcli) SelectBackend(modelName string) error {
	modelName = strings.ToLower(modelName)
	// [TODO] only test deepseek
	switch {
	case strings.Contains(modelName, "google"):
		l.providerType = providerGoogle
	case strings.Contains(modelName, "amazon"): // ?
		l.providerType = providerAmazon
	case strings.Contains(modelName, "azure"): // ?
		l.providerType = providerAzure
	case strings.Contains(modelName, "ollama"):
		l.providerType = providerOllama
		return l.initOllama()
	case strings.Contains(modelName, "anthropic"):
		l.providerType = providerAnthropic
		return l.initAnthropic()
	case strings.Contains(modelName, "deepseek"):
		return l.initOpenAI(providerDeepSeek)
	case strings.Contains(modelName, "kimi"):
		return l.initOpenAI(providerMoonshot)
	case strings.Contains(modelName, "doubao"):
		return l.initOpenAI(providerByteDance)
	case strings.Contains(modelName, "qwen"):
		return l.initOpenAI(providerAlibaba)
	case strings.Contains(modelName, "openai"):
		return l.initOpenAI(providerOpenAI)
	default:
		l.providerType = providerUnk
	}
	return errors.New(configs.GetLocalizedMsg("llm.UnknownLLMProviderErr", nil))
}

// SetMsg builds a message slice consisting of system prompt and user content.
// It uses l.RolePrompt as the system role definition and combines it with the input message.
// This method is designed to work with GenerateResponse. Using them together resets chat history,
// ensuring the LLM generates responses strictly based on the current prompt and message.
//
// Parameters:
//
//	msg - User input text content
//
// Returns:
//
//	[]llms.MessageContent - Standard message slice for LLM request
//
// Example:
//
//	resp, err := cli.GenerateResponse(cli.SetMsg("abc"), nil)
func (l *LLMcli) SetMsg(msg string) []llms.MessageContent {
	// only read for l.RolePrompt
	return []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, l.RolePrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, msg),
	}
}

// GenerateResponse must be used together with the SetMsg function.
// For example: GenerateResponse(SetMsg("abc"), nil).
// This approach clears the chat history, enabling the LLM to generate
// responses strictly based on the given prompt.
func (l *LLMcli) GenerateResponse(
	msg []llms.MessageContent,
	streamFn func(context.Context, []byte) error,
	contentOpts ...func(x string) string,
) (string, error) {
	// it seems that Most LLM providers lack official public APIs for
	// directly querying account balance and remaining credits.
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second*time.Duration(l.Timeout),
	)
	defer cancel()
	payload := []llms.CallOption{
		llms.WithTemperature(l.Temperature),
	}
	if streamFn != nil {
		payload = append(payload, llms.WithStreamingFunc(streamFn))
	}
	l.mu.Lock()
	if l.model == nil {
		l.mu.Unlock()
		return "", errors.New(configs.GetLocalizedMsg("llm.EmptyModelNameErr", nil))
	}
	resp, err := l.model.GenerateContent(ctx, msg, payload...)
	l.mu.Unlock()

	if err != nil {
		return "", err
	} else if resp == nil {
		return "", errors.New(configs.GetLocalizedMsg("llm.EmptyLLMResponseErr", nil))
	}
	if len(resp.Choices) < 1 {
		return "", errors.New(configs.GetLocalizedMsg("llm.EmptyLLMChoicesErr", nil))
	}
	// strip the sign of code block in Markdown if possible
	var content = resp.Choices[0].Content
	for _, opt := range contentOpts {
		content = opt(content)
	}
	if len(content) == 0 {
		return "", errors.New(configs.GetLocalizedMsg("llm.EmptyLLMRespContentErr", nil))
	}
	return content, nil
}

func (l *LLMcli) LoadFromFile(path string) error {
	viper.SetConfigFile(path)
	viper.SetConfigType("toml")
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	err = viper.Unmarshal(l)
	return err
}

func NewLLMcli(filePath string) (*LLMcli, error) {
	ret := &LLMcli{}
	err := ret.LoadFromFile(filePath)
	if err != nil {
		return nil, err
	}
	if !misc_utils.IsURL(ret.BaseURL) {
		return nil, errors.New(configs.GetLocalizedMsg("llm.InvalidBaseURLErr", nil))
	} else if len(ret.APIKey) == 0 {
		return nil, errors.New(configs.GetLocalizedMsg("llm.EmptyAPIKeyErr", nil))
	} else if len(ret.ModelName) == 0 {
		return nil, errors.New(configs.GetLocalizedMsg("llm.EmptyModelNameErr", nil))
	}
	if ret.Timeout == 0 {
		ret.Timeout = 120
	}
	return ret, ret.SelectBackend(ret.ModelName)
}
