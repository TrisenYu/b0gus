//go:build linux || darwin

// Package terminal
package terminal

import (
	"fmt"
	"regexp"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	bash "github.com/tree-sitter/tree-sitter-bash/bindings/go"
)

var cmdNameRegex = regexp.MustCompile(`^(?:/[\w/-]+|^[a-zA-Z][a-zA-Z0-9_-]*)$`)

// getNodeText will extract text of node from source by checking node.
// return empty string if startByte or endByte violates common sense.
// otherwise return calculated result.
func getNodeText(node *sitter.Node, source []byte) string {
	st, ed := int(node.StartByte()), int(node.EndByte())
	if st < 0 || ed > len(source) || st > ed {
		return ""
	}
	// text := strings.Trim(string(source[st:ed]), `"`)
	// return strings.TrimSpace(text)
	return string(source[st:ed])
}

// TODO check if the code is acceptable

func parseCommandNode(cmdNode *sitter.Node, source []byte) *GenericCmd {
	cmd := &GenericCmd{}
	for child := cmdNode.NamedChild(0); child != nil; child = child.NextNamedSibling() {
		switch child.GrammarName() {
		case "command_name":
			currCmd := getNodeText(child, source)
			// One possible situation: -Dname="$Helo"
			// TODO: then we have to record the argument and the dependProgRes in order?
			if len(currCmd) > 1 && currCmd[0] == '-' {
				// assignment for argument
				argsStruct := strings.SplitN(currCmd, "=", -1)
				cmd.InitArgs = append(cmd.InitArgs, currCmd)
				for j := 1; j < len(argsStruct); j++ {
					cmd.DependProgRes = append(cmd.DependProgRes, GenericCmd{CmdName: argsStruct[j]})
				}
				continue
			} else if strings.Contains(currCmd, `\`) || !cmdNameRegex.MatchString(currCmd) {
				// high {probability}/{confidence level} like an invalid command
				cmd.IsInvalid = true
				continue
			}
			cmd.CmdName = currCmd
		case "concatenation":
			currObj := getNodeText(child, source)
			if strings.Contains(currObj, "=") && len(currObj) > 1 && currObj[0] == '-' {
				// assignment for argument
				argsStruct := strings.SplitN(currObj, "=", -1)
				cmd.InitArgs = append(cmd.InitArgs, currObj)
				for j := 1; j < len(argsStruct); j++ {
					cmd.DependProgRes = append(cmd.DependProgRes, GenericCmd{CmdName: argsStruct[j]})
				}
				continue
			}
		case "argument":
			fallthrough
		case "word":
			fallthrough
		case "string":
			fallthrough
		case "test_operator":
			cmd.InitArgs = append(cmd.InitArgs, getNodeText(child, source))
		}
	}
	return cmd
}

func collectPipeCommands(pipeNode *sitter.Node, source []byte) []GenericCmd {
	var pipeCmds []GenericCmd
	for cmdNode := pipeNode.NamedChild(0); cmdNode != nil; cmdNode = cmdNode.NextNamedSibling() {
		if cmdNode.GrammarName() != "command" {
			// ? what is this?
			continue
		}
		cmd := parseCommandNode(cmdNode, source)
		if cmd != nil {
			if len(cmd.CmdName) > 0 {
				cmd.AsSrcCmd = true
				pipeCmds = append(pipeCmds, *cmd)
				continue
			}
			tail := len(pipeCmds) - 1
			for _, arg := range cmd.InitArgs {
				pipeCmds[tail].InitArgs = append(pipeCmds[tail].InitArgs, arg)
			}
			for _, dep := range cmd.DependProgRes {
				pipeCmds[tail].DependProgRes = append(pipeCmds[tail].DependProgRes, dep)
			}
		}
	}
	return pipeCmds
}

// travelShellAST will iterate Shell script string input(at most 1024 bytes) by outer untrusted user
// FIXME: too complicated, even can not pass the test.
func travelShellAST(node *sitter.Node, source []byte) []GenericCmd {
	var cmdArr []GenericCmd = nil
	if len(source) > 1024 {
		source = source[:1024]
	}
	fmt.Print(node.ToSexp())
	for child := node.NamedChild(0); child != nil; child = child.NextNamedSibling() {
		switch child.GrammarName() {
		case "for_statement":
		case "if_statement":
		case "list":
			fallthrough
		case "pipeline":
			fallthrough
		case "pipe_command":
			cmdArr = append(cmdArr, collectPipeCommands(child, source)...)
		case "redirected_statement":
			fallthrough
		case "command":
			cmd := parseCommandNode(child, source)
			if cmd == nil {
				continue
			}
			if len(cmd.CmdName) == 0 && len(cmdArr) >= 1 {
				tail := len(cmdArr) - 1
				for _, arg := range cmd.InitArgs {
					cmdArr[tail].InitArgs = append(cmdArr[tail].InitArgs, arg)
				}
				for _, dep := range cmd.DependProgRes {
					cmdArr[tail].DependProgRes = append(cmdArr[tail].DependProgRes, dep)
				}
				cmdArr[tail].IsInvalid = cmd.IsInvalid
				continue
			}
			cmdArr = append(cmdArr, *cmd)
		case "test_command":
			// TODO
		case "subshell":
			// WARN: recursion
			depProg := travelShellAST(child, source)
			if len(cmdArr) > 0 {
				cmdArr[len(cmdArr)-1].DependProgRes = append(cmdArr[len(cmdArr)-1].DependProgRes, depProg...)
			} else {
				cmdArr = append(cmdArr, depProg...)
			}

		case "ERROR":
			// no sense if we still keep it running.
			break
		default:
		}
	}
	return cmdArr
}

func (at *AbsTerminal) AnalyzeInput() {
	// When typing down the enter key, check if the last char is merely backslash.
	// Otherwise, split any space between input and mark the whole input as a (broken/correct-syntax) command.
	at.lock.Lock()
	defer at.lock.Unlock()

	switch at.ShellType {
	case "bash":
		at.ResolveShellCmd(bash.Language())
	case "powershell":
	case "Redis":
		fallthrough
	case "SQLite":
		fallthrough
	case "PostgreSQL":
		fallthrough
	case "MySQL":
		at.ResolveSQLCli()
	default:
		// ?
	}
}
