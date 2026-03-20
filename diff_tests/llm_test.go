package diff_tests

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"b0gus/llm"

	ark "github.com/sashabaranov/go-openai"
)

// TODO:
// 	1. when tokens run up, we have to detect this and switch the interation mode.
// 	2. evaluate different Models.
//  3. gain configuration from strict configuration file.
func TestLLM(t *testing.T) {
	config := ark.DefaultConfig(os.Getenv("LLM_API_KEY"))
	config.BaseURL = os.Getenv("LLM_URL")
	client := ark.NewClientWithConfig(config)
	stream, err := client.CreateChatCompletionStream(
		context.Background(),
		ark.ChatCompletionRequest{
			Model: os.Getenv("LLM_MODEL_TYPE"),
			Messages: []ark.ChatCompletionMessage{{
				Role: ark.ChatMessageRoleUser,
				MultiContent: []ark.ChatMessagePart{{
					Type: ark.ChatMessagePartTypeText,
					Text: "麻烦你为我翻译一下这段文本：“" + llm.ServPrompt + "”",
				}},
			}},
			ReasoningEffort: "medium",
		},
	)
	if err != nil {
		t.Fatalf("Stream chat error: %v\n", err)
	}
	defer func() { _ = stream.Close() }()
	for {
		recv, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Stream chat error: %v\n", err)
		}
		if len(recv.Choices) > 0 {
			if recv.Choices[0].Delta.ReasoningContent != "" {
				fmt.Print(recv.Choices[0].Delta.ReasoningContent)
			}
			fmt.Print(recv.Choices[0].Delta.Content)
		}
	}
}
