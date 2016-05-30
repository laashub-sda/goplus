package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"qlang.io/qlang.v2/qlang"
	"qlang.io/qlang/terminal"

	qspec "qlang.io/qlang.spec.v1"
	qipt "qlang.io/qlang.v2/interpreter"
	qall "qlang.io/qlang/qlang.all"
)

var (
	historyFile = os.Getenv("HOME") + "/.qlang.history"
)

func main() {
	qall.InitSafe(false)
	qlang.Import("", qipt.Exports)
	qlang.SetDumpCode(os.Getenv("QLANG_DUMPCODE"))

	libs := os.Getenv("QLANG_PATH")
	if libs == "" {
		libs = os.Getenv("HOME") + "/qlang"
	}

	lang, err := qlang.New(qlang.InsertSemis)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	lang.SetLibs(libs)

	// exec source
	if len(os.Args) > 1 {
		fname := os.Args[1]
		b, err := ioutil.ReadFile(fname)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		err = lang.SafeExec(b, fname)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(3)
		}
		return
	}

	// interpreter
	qall.Copyright()

	var ret interface{}
	qlang.SetOnPop(func(v interface{}) {
		ret = v
	})

	var tokener tokener
	term := terminal.New(">>> ", "... ", tokener.ReadMore)
	term.SetWordCompleter(func(line string, pos int) (head string, completions []string, tail string) {
		return line[:pos], []string{"  "}, line[pos:]
	})

	term.LoadHistroy(historyFile) // load/save histroy
	defer term.SaveHistroy(historyFile)

	for {
		expr, err := term.Scan()
		if err != nil {
			if err == terminal.ErrPromptAborted {
				break
			} else if err == io.EOF {
				fmt.Println("^D")
				break
			}
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		expr = strings.TrimSpace(expr)
		if expr == "" {
			continue
		}
		ret = qspec.Undefined
		err = lang.SafeEval(expr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		if ret != qspec.Undefined {
			fmt.Println(ret)
		}
	}
}

// -----------------------------------------------------------------------------

type tokener struct {
	level int
	instr bool
}

var dontReadMoreChars = "+-})];"
var puncts = "([=,*/%|&<>^.:"

func readMore(line string) bool {

	n := len(line)
	if n == 0 {
		return false
	}

	pos := strings.IndexByte(dontReadMoreChars, line[n-1])
	if pos == 0 || pos == 1 {
		return n >= 2 && line[n-2] != dontReadMoreChars[pos]
	}
	return pos < 0 && strings.IndexByte(puncts, line[n-1]) >= 0
}

func (p *tokener) ReadMore(expr string, line string) (string, bool) { // read more line check

	ret := expr + line + "\n"
	for {
		if p.instr {
			pos := strings.IndexByte(line, '`')
			if pos < 0 {
				return ret, true
			}
			line = line[pos+1:]
			p.instr = false
		}

		pos := strings.IndexAny(line, "{}`'\"")
		if pos < 0 {
			if p.level != 0 {
				return ret, true
			}
			line = strings.TrimRight(line, " \t")
			return ret, readMore(line)
		}
		switch c := line[pos]; c {
		case '{':
			p.level++
		case '}':
			p.level--
		case '`':
			p.instr = true
		default:
			line = line[pos+1:]
			pos = strings.IndexByte(line, c)
			if pos < 0 {
				return ret, p.level != 0
			}
		}
		line = line[pos+1:]
	}
}

// -----------------------------------------------------------------------------
