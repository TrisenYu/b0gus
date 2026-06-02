package assets

// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License
import _ "embed"

// 1. shell command validation (only for syntax, not requires for the judgment of meaning)
//    or the stimulation of response
// 2. translation demand on natural language text

//go:embed prompts/cmd-validate.txt
var RequestToValidateCmd string

//go:embed prompts/cmd-response.txt
var RequestToResponseCmd string

//go:embed prompts/translation-demand.txt
var RequestToTranslate string
