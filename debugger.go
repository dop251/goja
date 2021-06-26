package goja

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/dop251/goja/parser"
	"github.com/dop251/goja/unistring"
)

const (
	SetBreakpoint   = "sb"
	ClearBreakpoint = "cb"
	Breakpoints     = "breakpoints"
	Next            = "n"
	Continue        = "c"
	StepIn          = "s"
	StepOut         = "o"
	Exec            = "e"
	Print           = "p"
	List            = "l"
	Help            = "h"
	Quit            = "q"
	Empty           = ""
	NewLine         = "\n"
)

const (
	GreenColor = "\u001b[32m"
	GrayColor  = "\u001b[38;5;245m"
	ResetColor = "\u001b[0m"
)

type Debugger struct {
	vm *vm

	lastDebuggerCmdAndArgs []string
	debuggerExec           bool
	currentLine            int
	lastLines              []int
	breakpoints            []Breakpoint
}

func NewDebugger(vm *vm) *Debugger {
	dbg := &Debugger{
		vm: vm,
	}
	dbg.lastLines = append(dbg.lastLines, 0)
	return dbg
}

type Breakpoint struct {
	Filename string
	Line     int
}

func (dbg *Debugger) setBreakpoint(fileName string, line int) {
	b := Breakpoint{Filename: fileName, Line: line}
	for _, elem := range dbg.breakpoints {
		if elem == b {
			return
		}
	}
	dbg.breakpoints = append(dbg.breakpoints, b)
}

func (dbg *Debugger) clearBreakpoint(fileName string, line int) {
	if len(dbg.breakpoints) == 0 {
		return
	}

	b := Breakpoint{Filename: fileName, Line: line}
	for idx, elem := range dbg.breakpoints {
		if elem == b {
			dbg.breakpoints = append(dbg.breakpoints[:idx], dbg.breakpoints[idx+1:]...)
			return
		}
	}
}

type Command interface {
	Execute()
}

type NextCommand struct{}

func (*NextCommand) Execute(dbg *Debugger) {
	lastLine := dbg.getCurrentLine()
	dbg.updateCurrentLine()
	if dbg.getLastLine() != dbg.getCurrentLine() {
		dbg.REPL(false)
	}
	nextLine := dbg.getNextLine()
	for dbg.isSafeToRun() && dbg.getCurrentLine() != nextLine {
		dbg.updateCurrentLine()
		if dbg.isDebuggerStatement() {
			break
		}
		dbg.vm.prg.code[dbg.vm.pc].exec(dbg.vm)
	}
	dbg.updateLastLine(lastLine)
}

type ContinueCommand struct{}

func (*ContinueCommand) Execute(dbg *Debugger) {
	lastLine := dbg.getCurrentLine()
	dbg.updateCurrentLine()
	for dbg.isSafeToRun() && !dbg.isDebuggerStatement() {
		if dbg.isBreakpoint() {
			dbg.REPL(false)
			dbg.updateCurrentLine()
			dbg.updateLastLine(lastLine)
			return
		}
		dbg.vm.prg.code[dbg.vm.pc].exec(dbg.vm)
		dbg.updateCurrentLine()
	}
	dbg.updateLastLine(lastLine)
}

type StepInCommand struct{}

func (*StepInCommand) Execute(dbg *Debugger) {
	fmt.Println("Not Implemented Yet")
}

type StepOutCommand struct{}

func (*StepOutCommand) Execute(dbg *Debugger) {
	fmt.Println("Not Implemented Yet")
}

type ExecCommand struct {
	expression string
}

func (e *ExecCommand) Execute(dbg *Debugger) {
	dbg.debuggerExec = true
	value := dbg.evalCode(e.expression)
	fmt.Printf("< Return: %s\n", value.ToString())
	dbg.debuggerExec = false

	lastLine := dbg.getCurrentLine()
	dbg.REPL(false)
	dbg.updateLastLine(lastLine)
}

type PrintCommand struct {
	varName string
}

func (p *PrintCommand) Execute(dbg *Debugger) {
	val := dbg.getValue(p.varName)
	if val == Undefined() {
		fmt.Println("Cannot get variable from local scope. However, the current values on the stack are:")
		fmt.Printf("< %s\n", dbg.vm.prg.values)
	} else {
		fmt.Printf("< %s\n", val)
	}
}

type ListCommand struct{}

func (*ListCommand) Execute(dbg *Debugger) {
	fmt.Println(dbg.printSource())
}

type HelpCommand struct{}

func (*HelpCommand) Execute(dbg *Debugger) {
	help := []string{
		"setBreakpoint, sb\tSet a breakpoint on a given file and line",
		"clearBreakpoint, cb\tClear a breakpoint on a given file and line",
		"breakpoints\t\tList all known breakpoints",
		"next, n\t\t\tContinue to next line in current file",
		"cont, c\t\t\tResume execution until next debugger line",
		"step, s\t\t\tStep into, potentially entering a function (Not Implemented Yet)",
		"out, o\t\t\tStep out, leaving the current function (Not Implemented Yet)",
		"exec, e\t\t\tEvaluate the expression and print the value",
		"list, l\t\t\tPrint the source around the current line where execution is currently paused",
		"print, p\t\tPrint the provided variable's value",
		"help, h\t\t\tPrint this very help message",
		"quit, q\t\t\tExit debugger and quit (Ctrl+C)",
	}

	for _, value := range help {
		fmt.Println(value)
	}
}

type QuitCommand struct {
	exitCode int
}

func (q *QuitCommand) Execute(dbg *Debugger) {
	os.Exit(q.exitCode)
}

type EmptyCommand struct{}
type NewLineCommand struct{}

func StringToLines(s string) (lines []string, err error) {
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	err = scanner.Err()
	return
}

func CountDigits(number int) int {
	if number < 10 {
		return 1
	} else {
		return 1 + CountDigits(number/10)
	}
}

func InBetween(i, min, max int) bool {
	if (i >= min) && (i <= max) {
		return true
	} else {
		return false
	}
}

func (dbg *Debugger) isDebuggerStatement() bool {
	return dbg.vm.prg.code[dbg.vm.pc] == debugger
}

func (dbg *Debugger) isNextDebuggerStatement() bool {
	return dbg.vm.pc+1 < len(dbg.vm.prg.code) && dbg.vm.prg.code[dbg.vm.pc+1] == debugger
}

func (dbg *Debugger) isBreakpoint() bool {
	currentLine := dbg.getCurrentLine()
	currentFilename := dbg.getCurrentFilename()

	b := Breakpoint{Filename: currentFilename, Line: currentLine}
	for _, elem := range dbg.breakpoints {
		if elem == b {
			return true
		}
	}
	return false
}

func (dbg *Debugger) lastDebuggerCommand() string {
	if len(dbg.lastDebuggerCmdAndArgs) > 0 {
		return dbg.lastDebuggerCmdAndArgs[0]
	}

	return Empty
}

func (dbg *Debugger) getLastLine() int {
	if len(dbg.lastLines) > 0 {
		return dbg.lastLines[len(dbg.lastLines)-1]
	}
	// First executed line (current line) is considered the last line
	return dbg.getCurrentLine()
}

func (dbg *Debugger) updateLastLine(lineNumber int) {
	if len(dbg.lastLines) > 0 && dbg.lastLines[len(dbg.lastLines)-1] != lineNumber {
		dbg.lastLines = append(dbg.lastLines, lineNumber)
	}
}

func (dbg *Debugger) getCurrentLine() int {
	// FIXME: Some lines are skipped, which causes this function to report incorrect lines
	currentLine := dbg.vm.prg.src.Position(dbg.vm.prg.sourceOffset(dbg.vm.pc)).Line
	return currentLine
}

func (dbg *Debugger) getCurrentFilename() string {
	currentFilename := dbg.vm.prg.src.Position(dbg.vm.prg.sourceOffset(dbg.vm.pc)).Filename
	return currentFilename
}

func (dbg *Debugger) updateCurrentLine() {
	dbg.currentLine = dbg.getCurrentLine()
}

func (dbg *Debugger) getNextLine() int {
	for idx := range dbg.vm.prg.code[dbg.vm.pc:] {
		nextLine := dbg.vm.prg.src.Position(dbg.vm.prg.sourceOffset(dbg.vm.pc + idx + 1)).Line
		if nextLine > dbg.getCurrentLine() {
			return nextLine
		}
	}
	return 0
}

func (dbg *Debugger) isSafeToRun() bool {
	return dbg.vm.pc < len(dbg.vm.prg.code)
}

func (dbg *Debugger) printSource() string {
	lines, err := StringToLines(dbg.vm.prg.src.Source())
	if err != nil {
		log.Fatal(err)
	}
	currentLine := dbg.getCurrentLine()
	lineIndex := currentLine - 1
	var builder strings.Builder
	for idx, lineContents := range lines {
		if InBetween(lineIndex, idx-4, idx+4) {
			lineNumber := idx + 1
			totalPadding := 6
			digitCount := CountDigits(lineNumber)
			if digitCount >= totalPadding {
				totalPadding = digitCount + 1
			}
			if currentLine == lineNumber {
				padding := strings.Repeat(" ", totalPadding-digitCount)
				builder.Write([]byte(fmt.Sprintf("%s>%s %d%s%s\n", GreenColor, ResetColor, currentLine, padding, lines[lineIndex])))
			} else {
				padding := strings.Repeat(" ", totalPadding-digitCount)
				builder.Write([]byte(fmt.Sprintf("%s  %d%s%s%s\n", GrayColor, lineNumber, padding, lineContents, ResetColor)))
			}
		}
	}

	return builder.String()
}

func (dbg *Debugger) evalCode(src string) Value {
	prg, err := parser.ParseFile(nil, "<eval>", src, 0)
	if err != nil {
		log.Fatal(&CompilerSyntaxError{
			CompilerError: CompilerError{
				Message: err.Error(),
			},
		})
	}

	c := newCompiler()

	defer func() {
		if x := recover(); x != nil {
			c.p = nil
			switch x1 := x.(type) {
			case *CompilerSyntaxError:
				err = x1
			default:
				panic(x)
			}
		}
	}()

	var this Value
	if dbg.vm.sb >= 0 {
		this = dbg.vm.stack[dbg.vm.sb]
	} else {
		this = dbg.vm.r.globalObject
	}

	c.compile(prg, false, true, this == dbg.vm.r.globalObject)

	defer func() {
		if x := recover(); x != nil {
			if ex, ok := x.(*uncatchableException); ok {
				err = ex.err
			} else {
				panic(x)
			}
		}
	}()

	dbg.vm.pushCtx()
	dbg.vm.prg = c.p
	dbg.vm.pc = 0
	dbg.vm.args = 0
	dbg.vm.result = _undefined
	dbg.vm.sb = dbg.vm.sp
	dbg.vm.push(this)
	dbg.vm.run()
	retval := dbg.vm.result
	dbg.vm.popCtx()
	dbg.vm.halt = false
	dbg.vm.sp -= 1
	return retval
}

func (dbg *Debugger) isBreakOnStart() bool {
	return dbg.vm.pc < 3 && dbg.vm.prg.code[2] == debugger
}

func (dbg *Debugger) getValue(varName string) Value {
	name := unistring.String(varName)
	var val Value
	for stash := dbg.vm.stash; stash != nil; stash = stash.outer {
		if v, exists := stash.getByName(name); exists {
			val = v
			break
		}
	}
	if val == nil {
		if dbg.vm.sb >= 0 {
			val = dbg.vm.stack[dbg.vm.sb]
		}
		if val != Undefined() || val != nil {
			return val
		}

		val = dbg.vm.r.globalObject.self.getStr(name, nil)
		if val == nil {
			val = valueUnresolved{r: dbg.vm.r, ref: name}
		}
	}
	return val
}

func (dbg *Debugger) REPL(intro bool) {
	// Refactor this piece of sh!t
	debuggerCommands := map[string]string{
		"setBreakpoint":   SetBreakpoint,
		SetBreakpoint:     SetBreakpoint,
		"clearBreakpoint": ClearBreakpoint,
		ClearBreakpoint:   ClearBreakpoint,
		"breakpoints":     Breakpoints,
		"next":            Next,
		Next:              Next,
		"cont":            Continue,
		Continue:          Continue,
		"step":            StepIn,
		StepIn:            StepIn,
		"out":             StepOut,
		StepOut:           StepOut,
		"exec":            Exec,
		Exec:              Exec,
		"print":           Print,
		Print:             Print,
		"list":            List,
		List:              List,
		"help":            Help,
		Help:              Help,
		"quit":            Quit,
		Quit:              Quit,
		NewLine:           "\n",
	}

	if intro {
		fmt.Println("Welcome to Goja debugger")
		fmt.Println("Type 'help' or 'h' for list of commands.")
	} else {
		if dbg.isBreakOnStart() {
			fmt.Printf("Break on start in %s\n", dbg.vm.prg.src.Position(dbg.vm.prg.sourceOffset(dbg.vm.pc)))
		} else {
			fmt.Printf("Break in %s\n", dbg.vm.prg.src.Position(dbg.vm.prg.sourceOffset(dbg.vm.pc)))
		}
		fmt.Println(dbg.printSource())
	}

	var commandAndArguments []string

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("debug[%d]> ", dbg.vm.pc)
		command, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				break
			}
			log.Fatal(err)
		}

		commandAndArguments = strings.Split(command[:len(command)-1], " ")
		if command == NewLine && len(dbg.lastDebuggerCmdAndArgs) > 0 {
			// If enter is pressed and there's a command already executed,
			// run the last debugger command
			commandAndArguments = make([]string, len(dbg.lastDebuggerCmdAndArgs))
			copy(commandAndArguments, dbg.lastDebuggerCmdAndArgs)
		}

		if v, ok := debuggerCommands[commandAndArguments[0]]; ok {
			if command != NewLine {
				// FIXME: Exec command acts as Next on the next run
				dbg.lastDebuggerCmdAndArgs = make([]string, len(commandAndArguments))
				copy(dbg.lastDebuggerCmdAndArgs, commandAndArguments)
			}

			switch v {
			case SetBreakpoint:
				if len(commandAndArguments) < 3 {
					fmt.Println("sb filename linenumber")
					continue
				}
				if line, err := strconv.Atoi(commandAndArguments[2]); err != nil {
					fmt.Printf("Cannot convert %s to line number\n", commandAndArguments[2])
				} else {
					dbg.setBreakpoint(commandAndArguments[1], line)
				}
			case ClearBreakpoint:
				if len(commandAndArguments) < 3 {
					fmt.Println("cb filename linenumber")
					continue
				}
				if line, err := strconv.Atoi(commandAndArguments[2]); err != nil {
					fmt.Printf("Cannot convert %s to line number\n", commandAndArguments[2])
				} else {
					dbg.clearBreakpoint(commandAndArguments[1], line)
				}
			case Breakpoints:
				for _, b := range dbg.breakpoints {
					fmt.Printf("Breakpoint on %s:%d\n", b.Filename, b.Line)
				}
			case Next:
				return
			case Continue:
				return
			case StepIn:
				cmd := StepInCommand{}
				cmd.Execute(dbg)
			case StepOut:
				cmd := StepOutCommand{}
				cmd.Execute(dbg)
			case Exec:
				cmd := ExecCommand{expression: strings.Join(commandAndArguments[1:], ";")}
				cmd.Execute(dbg)
				return
			case Print:
				cmd := PrintCommand{varName: strings.Join(commandAndArguments[1:], "")}
				cmd.Execute(dbg)
			case List:
				cmd := ListCommand{}
				cmd.Execute(dbg)
			case Help:
				cmd := HelpCommand{}
				cmd.Execute(dbg)
			case Quit:
				cmd := QuitCommand{exitCode: 0}
				cmd.Execute(dbg)
			default:
				cmd := QuitCommand{exitCode: 0}
				cmd.Execute(dbg)
			}
		} else {
			fmt.Println("unknown command")
		}
	}
}
