package diff_tests

import (
	"b0gus/internal/misc_utils"
	"testing"

	"github.com/stretchr/testify/assert"

	"b0gus/llm"
)

func TestLLMConfig(t *testing.T) {
	// [TODO] load from file
	lc := llm.LLMcli{
		ModelName: "deepseek",
		RolePrompt: "your role is to judge whether the given content can be " +
			"only utilized in unix shell environment",
	}
	err := lc.SelectBackend("deepseek")
	assert.Error(t, err)
	resp, _ := lc.GenerateResponse(lc.SetMsg("uname"), nil)
	t.Log(resp)

	err = lc.SelectBackend("doubao")
	assert.Error(t, err)
	resp, _ = lc.GenerateResponse(lc.SetMsg("uname"), nil, misc_utils.StripMarkdownSignIfAny)
	t.Log(resp)
}
