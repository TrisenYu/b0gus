package llm

/// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License
/// Last modified at 2026/05/12 星期二 12:29:13
import (
	"context"
	"encoding/json"
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
)

// configuration of LLM should be accessed from config.toml
// including API and which content to interact with
// https://github.com/0x4D31/galah/blob/main/pkg/llm/llm.go

type llmTaskType int

const (
	cmdValidation llmTaskType = iota
	cmdResponse
	translation
)

var (
	llmTaskDispatcher = map[llmTaskType]string{
		cmdValidation: assets.RequestForCmdValidation,
		cmdResponse:   assets.RequestForCmdResponse,
		translation:   assets.RequestForTranslation,
	}
)

// Wrap will convert the current input string `i` into
// `[ProtocolSt] i [ProtocolEd]`, whereas both [ProtocolSt] and [ProtocolEd]
// are defined by specific protocol.
func (i InputForLLM) Wrap(
	ProtocolSt, ProtocolEd string,
) string {
	var sb strings.Builder
	sb.WriteString(ProtocolSt)
	sb.WriteString(string(i))
	sb.WriteString(ProtocolEd)
	return sb.String()
}

type HttpHeaderSetAuthFn func(*http.Request, string)

// WithBearerAuth will set http header with `Authorization: Bearer [APIKey]`
// Typically will be utilized by Deepseek or Kimi.
func WithBearerAuth(req *http.Request, key string) {
	var sb strings.Builder
	sb.WriteString("Bearer ")
	sb.WriteString(key)
	req.Header.Set("Authorization", sb.String())
}

// WithXapiAuth will set http header with `x-api-key: [APIKey]`
// Typically will be utilized by Anthropic.
func WithXapiAuth(req *http.Request, key string) {
	req.Header.Set("x-api-key", key)
}

func (l *LLMconfig) CheckBalance(PtrOfRes any, headerOpt HttpHeaderSetAuthFn) error {
	// [TODO]: define the structure of <PtrOfRes>
	if len(l.BalanceAPI) == 0 || len(l.APIKey) == 0 {
		return errors.New("balance api or api_key is empty")
	}
	var sb strings.Builder
	sb.WriteString(l.BaseURL)
	sb.WriteString(l.BalanceAPI)
	req, err := http.NewRequest(http.MethodGet, sb.String(), nil)
	if err != nil {
		return err
	}
	headerOpt(req, l.APIKey)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return json.NewDecoder(resp.Body).Decode(PtrOfRes)
}

func (l *LLMconfig) AlterPrompt(taskType llmTaskType) {
	content, ok := llmTaskDispatcher[taskType]
	if !ok {
		var sb strings.Builder
		sb.WriteString("unable to set prompt by current taskType<")
		sb.WriteString(strconv.Itoa(int(taskType)))
		sb.WriteRune('>')
		configs.Logger().Warn(sb.String())
		return
	}
	l.RolePrompt = content
}

func (l *LLMconfig) initCheck() error {
	switch {
	case len(l.BaseURL) == 0:
		return errors.New("base url is empty")
	case len(l.APIKey) == 0:
		return errors.New("api key is empty")
	case len(l.ModelName) == 0:
		return errors.New("model name is empty")
	case len(l.RolePrompt) == 0:
		return errors.New("role prompt is empty")
	}
	return nil
}
func (l *LLMconfig) initOpenAI() error {
	if err := l.initCheck(); err != nil {
		return err
	}
	opts := []openai.Option{
		openai.WithBaseURL(l.BaseURL),
		openai.WithModel(l.ModelName),
	}
	m, err := openai.New(opts...)
	if err != nil {
		return err
	}
	l.model = m
	return nil
}
func (l *LLMconfig) initOllama() error {
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
	l.model = m
	return nil
}

func (l *LLMconfig) initAnthropic() error {
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
	l.model = m
	return nil
}

func (l *LLMconfig) SelectBackend(providerName string) error {
	providerName = strings.ToLower(providerName)
	switch providerName {
	// case "googleai":
	//	return l.initGoogleAI()
	case "ollama":
		return l.initOllama()
	case "anthropic":
		return l.initAnthropic()
	case "deepseek", "kimi", "qwen":
		fallthrough
	case "openai":
		return l.initOpenAI()
	}
	return errors.New("unknown provider")
}

func (l *LLMconfig) SetMsg(msg string) []llms.MessageContent {
	return []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, l.RolePrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, msg),
	}
}

func (l *LLMconfig) GenerateResponse( // GenerateResponse(SetMsg(...), nil)
	msg []llms.MessageContent,
	streamFn func(context.Context, []byte) error,
) (string, error) {
	// [TODO]: concurrent rate-limit and balance limit.
	// it seems that Most LLM providers lack official public APIs for directly querying account balance and
	// remaining credits.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	payload := []llms.CallOption{
		llms.WithTemperature(l.Temperature),
	}
	if streamFn != nil {
		payload = append(payload, llms.WithStreamingFunc(streamFn))
	}
	if l.model == nil {
		return "", errors.New("model is empty")
	}
	resp, err := l.model.GenerateContent(
		ctx, msg,
		payload...,
	)
	if err != nil {
		return "", err
	} else if resp == nil {
		return "", errors.New("nil response")
	}
	if len(resp.Choices) < 1 {
		return "", errors.New("no choices found")
	}
	content := resp.Choices[0].Content
	if len(content) == 0 {
		return "", errors.New("no content found")
	}
	return content, nil
}

func (l *LLMconfig) ResetModel() { l.model = nil }

func (l *LLMconfig) LoadFromFile(path string) error {
	viper.SetConfigFile(path)
	viper.SetConfigType("toml")
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	err = viper.Unmarshal(l)
	return err
}

func (l *LLMconfig) ValidateShellCmd(x string) (string, error) {
	err := l.LoadFromFile("[TODO]")
	if err != nil {
		// handle by other shell command checking functions.
		return "", err
	} // unavailable
	l.AlterPrompt(cmdValidation)
	return l.GenerateResponse(l.SetMsg(x), nil)
}
