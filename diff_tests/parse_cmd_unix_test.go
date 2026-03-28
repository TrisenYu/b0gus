package diff_tests

import (
	"bytes"
	"testing"

	"b0gus/.vscode"
	"github.com/stretchr/testify/assert"
	bash "github.com/tree-sitter/tree-sitter-bash/bindings/go"
)

func TestCmdParser(t *testing.T) {
	// recommend that the size will not exceed 1024,
	// therefore ensure the robustness and shield of stackoverflow
	buf := make([]byte, 1024)
	readerFD := bytes.NewReader(buf)
	termInHand := _vscode.AbsTerminal{
		ReadSrc:   readerFD,
		WriteSrc:  nil,
		ShellType: "bash",
	}
	bashLang := bash.Language()
	copy(buf, "\000")
	got := termInHand.ResolveShellCmd(bashLang)
	assert.Nil(t, got, "should get nil but not empty structure")

	// case 1 \000 must set otherwise the last input will pollute current testcase.
	copy(buf, "ls -liha"+"\000")
	_, err := readerFD.Seek(0, 0)
	assert.NoError(t, err)
	got = termInHand.ResolveShellCmd(bashLang)
	assert.NotNil(t, got, "got unexpected nil")
	if assert.Equal(t, 1, len(got), "len(got) should be equal to 1 but output shows a weird result") {
		assert.Equal(t, "ls", got[0].CmdName, "can not properly set command")
		assert.Equal(t, 1, len(got[0].InitArgs), "invalid length of init args")
		assert.Equal(t, "-liha", got[0].InitArgs[0], "can not properly set init args")
	}

	// case 2
	copy(buf, "ls \\\n-liha"+"\000")
	_, err = readerFD.Seek(0, 0)
	assert.NoError(t, err)
	got = termInHand.ResolveShellCmd(bashLang)
	assert.NotEqual(t, nil, got, "got unexpected nil")
	if assert.Equal(t, 1, len(got), "len(got) should be equal to 1 but output shows a weird result") {
		assert.Equal(t, "ls", got[0].CmdName, "can not properly set command")
		assert.Equal(t, "-liha", got[0].InitArgs[0], "can not properly set init args")
	}
	// case 3 broken argument
	copy(buf, `ls \\\\
-`+"\000")
	_, err = readerFD.Seek(0, 0)
	assert.NoError(t, err)
	got = termInHand.ResolveShellCmd(bashLang)
	assert.NotNil(t, got, "got unexpected nil")
	if assert.Equal(t, 1, len(got), "invalid escape cause memory leak") {
		assert.Equal(t, true, got[0].IsInvalid, "false judge")
	}

	// case 4 ridiculous one
	copy(buf, `gcc -D__NONSE_FLAG__="$a"="$b" a.c -O2 -o a.exe`+"\000")
	_, err = readerFD.Seek(0, 0)
	assert.NoError(t, err)
	got = termInHand.ResolveShellCmd(bashLang)
	assert.NotNil(t, got, "got unexpected nil")
	assert.Equal(t, "gcc", got[0].CmdName, "can not properly set command")
	if assert.Equal(t, 5, len(got[0].InitArgs), "invalid length of init args") {
		assert.Equal(t, `-D__NONSE_FLAG__="$a"="$b"`, got[0].InitArgs[0], "wrong init args")
		assert.Equal(t, `a.c`, got[0].InitArgs[1], "wrong init args")
		assert.Equal(t, `-O2`, got[0].InitArgs[2], "wrong init args")
		assert.Equal(t, `-o`, got[0].InitArgs[3], "wrong init args")
		assert.Equal(t, `a.exe`, got[0].InitArgs[4], "wrong init args")
		assert.Equal(t, false, got[0].IsInvalid, "false judge")
	}

	// case 5 complex one
	copy(
		buf,
		`find *.g4 -print0 | xargs -0 antlr4 -Werror \
# this is a comment line. And I set $Lang to Go
-Dlanguage="$Lang" \
# I choose listener mode but not visitor mode.
-no-visitor -listener \
-o "$(pwd)" -Xexact-output-dir`+"\000",
	)

	_, err = readerFD.Seek(0, 0)
	assert.NoError(t, err)
	got = termInHand.ResolveShellCmd(bashLang)
	assert.NotNil(t, got, "got unexpected nil")
	if assert.Equal(t, 2, len(got), "invalid length of GenericCmd array") {
		assert.Equal(t, false, got[0].IsInvalid, "false judge")
		assert.Equal(t, false, got[1].IsInvalid, "false judge")
	}

	copy(buf, `grep -rin "\"test\"" ./*`+"\000")
	_, err = readerFD.Seek(0, 0)
	assert.NoError(t, err)
	got = termInHand.ResolveShellCmd(bashLang)
	assert.NotNil(t, got, "got unexpected nil")
	assert.Equal(t, 1, len(got), "invalid escape cause memory leak")
}

// Have a try
func FuzzTerminal(f *testing.F) {
	buf := make([]byte, 1024)
	readerFD := bytes.NewReader(buf)
	termInHand := _vscode.AbsTerminal{
		ReadSrc:   readerFD,
		WriteSrc:  nil,
		ShellType: "bash",
	}
	tests := []string{
		"",
		"ls -liha" + "\000",
		"ls \\\\\\\\\n-liha" + "\000",
		"gcc -D__NONSE_FLAG__=\"$a\"=\"$b\" a.c -O2 -o a.exe" + "\000",
		"gcc -D__NONSE_FLAG__=\"$a\"==\"$b\" a.c -O2 -o a.exe" + "\000",
		`find *.g4 -print0 | xargs -0 antlr4 -Werror \
# this is a comment line. And I set $Lang to Go
-Dlanguage="$Lang" \
# I choose listener mode but not visitor mode.
-no-visitor -listener \
-o "$(pwd)" -Xexact-output-dir` + "\000",
		`[ -w "$PWD" -a -x "$PWD" ] && echo "可创建文件" || echo "无权创建"` + "\000", // utf8 test
		"readelf -a ./build/lib/halo_world.o | tail -n 32" + "\000",
		"uv pip freeze > requirements.txt" + "\000",
		"echo 3 > /proc/sys/vm/drop_caches" + "\000",
		`history | grep -in "<<" |` + "\000",  // illness
		`history | grep -in "<<" &>` + "\000", // illness
		`xargs grep -in "<<" &>` + "\000",     // illness
		`\
for cur_mirr in ${famous_mirrors[@]}; do 
	local mid_rtt_val=$(ping -c $ping_times $cur_mirr)
	[[ "$?" != 0 ]] && continue
	local rtt_val=$(echo "$mid_rtt_val" | $ggrep " min/avg/max/$ping_dev = [0-9\\./]\\+ ms\$" | \
	awk -F'/' '{ print $6 }')
	if [[ "$rtt_val" == "" ]]; then
	fi
done` + "\000", // !problematic
		"1+2+3+4+5+6+8+9+10" + "\000", // illness
		"abc" + "\000",                // illness
		`if [ "$rtt_val" -eq "" ]; then 
	echo yes, you are right
else
	echo no, you are wrong
fi` + "\000",
	}
	for _, ts := range tests {
		f.Add(ts)
	}
	bashLang := bash.Language()
	f.Fuzz(
		func(t *testing.T, a string) {
			copy(buf, a)
			_, err := readerFD.Seek(0, 0)
			assert.NoError(t, err)
			get := termInHand.ResolveShellCmd(bashLang)
			if len(a) == 0 {
				assert.Nil(t, get)
				return
			}
			assert.NotNil(t, get)
		},
	)
}
