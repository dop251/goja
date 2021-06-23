package goja

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/dop251/goja/parser"
)

const (
	Next     = "n"
	Continue = "c"
	StepIn   = "s"
	StepOut  = "o"
	Exec     = "e"
	Print    = "p"
	List     = "l"
	Help     = "h"
	Quit     = "q"
	Empty    = ""
	NewLine  = "\n"
)

const (
	GreenColor = "\u001b[32m"
	GrayColor  = "\u001b[38;5;245m"
	ResetColor = "\u001b[0m"
)

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

func (vm *vm) isDebuggerStatement() bool {
	return vm.prg.code[vm.pc] == debugger
}

func (vm *vm) isNextDebuggerStatement() bool {
	return vm.pc+1 < len(vm.prg.code) && vm.prg.code[vm.pc+1] == debugger
}

func (vm *vm) lastDebuggerStatement() string {
	if len(vm.lastDebuggerCmdAndArgs) > 0 {
		return vm.lastDebuggerCmdAndArgs[0]
	}

	return Empty
}

func (vm *vm) getLastLine() int {
	if len(vm.lastLines) > 0 {
		return vm.lastLines[len(vm.lastLines)-1]
	}
	// First executed line (current line) is considered the last line
	return vm.getCurrentLine()
}

func (vm *vm) updateLastLine(lineNumber int) {
	if len(vm.lastLines) > 0 && vm.lastLines[len(vm.lastLines)-1] != lineNumber {
		vm.lastLines = append(vm.lastLines, lineNumber)
	}
}

func (vm *vm) getCurrentLine() int {
	// FIXME: Some lines are skipped, which causes this function to report incorrect lines
	currentLine := vm.prg.src.Position(vm.prg.sourceOffset(vm.pc)).Line
	return currentLine
}

func (vm *vm) updateCurrentLine() {
	vm.currentLine = vm.getCurrentLine()
}

func (vm *vm) getNextLine() int {
	for idx := range vm.prg.code[vm.pc:] {
		nextLine := vm.prg.src.Position(vm.prg.sourceOffset(vm.pc + idx + 1)).Line
		if nextLine > vm.getCurrentLine() {
			return nextLine
		}
	}
	return 0
}

func (vm *vm) isSafeToRun() bool {
	return vm.pc < len(vm.prg.code)
}

func (vm *vm) printSource() string {
	lines, err := StringToLines(vm.prg.src.Source())
	if err != nil {
		log.Fatal(err)
	}
	currentLine := vm.getCurrentLine()
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

func (vm *vm) evalCode(src string) Value {
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
	if vm.sb >= 0 {
		this = vm.stack[vm.sb]
	} else {
		this = vm.r.globalObject
	}

	c.compile(prg, false, true, this == vm.r.globalObject, false)

	defer func() {
		if x := recover(); x != nil {
			if ex, ok := x.(*uncatchableException); ok {
				err = ex.err
			} else {
				panic(x)
			}
		}
	}()

	vm.pushCtx()
	vm.prg = c.p
	vm.pc = 0
	vm.args = 0
	vm.result = _undefined
	vm.sb = vm.sp
	vm.push(this)
	vm.run()
	retval := vm.result
	vm.popCtx()
	vm.halt = false
	vm.sp -= 1
	return retval
}

func (vm *vm) isBreakOnStart() bool {
	return vm.pc < 3 && vm.prg.code[2] == debugger
}

func (vm *vm) repl(intro bool) {
	// Refactor this piece of sh!t
	debuggerCommands := map[string]string{
		"next":   Next,
		Next:     Next,
		"cont":   Continue,
		Continue: Continue,
		"step":   StepIn,
		StepIn:   StepIn,
		"out":    StepOut,
		StepOut:  StepOut,
		"exec":   Exec,
		Exec:     Exec,
		"print":  Print,
		Print:    Print,
		"list":   List,
		List:     List,
		"help":   Help,
		Help:     Help,
		"quit":   Quit,
		Quit:     Quit,
		NewLine:  "\n",
	}
	debuggerHelp := []string{
		"next, n\t\tContinue to next line in current file",
		"cont, c\t\tResume execution until next debugger line",
		"step, s\t\tStep into, potentially entering a function (Not Implemented Yet)",
		"out, o\t\tStep out, leaving the current function (Not Implemented Yet)",
		"exec, e\t\tEvaluate the expression and print the value",
		"list, l\t\tPrint the source around the current line where execution is currently paused (Not Implemented Yet)",
		"print, p\tPrint the provided variable's value (Not Implemented Yet)",
		"help, h\t\tPrint this very help message",
		"quit, q\t\tExit debugger and quit (Ctrl+C)",
	}

	if intro {
		fmt.Println("Welcome to Goja debugger")
		fmt.Println("Type 'help' or 'h' for list of commands.")
	} else {
		if vm.isBreakOnStart() {
			fmt.Printf("Break on start in %s\n", vm.prg.src.Position(vm.prg.sourceOffset(vm.pc)))
		} else {
			fmt.Printf("Break in %s\n", vm.prg.src.Position(vm.prg.sourceOffset(vm.pc)))
		}
		fmt.Println(vm.printSource())
	}

	var commandAndArguments []string

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("debug[%d]> ", vm.pc)
		command, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				break
			}
			log.Fatal(err)
		}

		commandAndArguments = strings.Split(command[:len(command)-1], " ")
		if command == NewLine && len(vm.lastDebuggerCmdAndArgs) > 0 {
			// If enter is pressed and there's a command already executed,
			// run the last debugger command
			commandAndArguments = make([]string, len(vm.lastDebuggerCmdAndArgs))
			copy(commandAndArguments, vm.lastDebuggerCmdAndArgs)
		}

		if v, ok := debuggerCommands[commandAndArguments[0]]; ok {
			if command != NewLine {
				// FIXME: Exec command acts as Next on the next run
				vm.lastDebuggerCmdAndArgs = make([]string, len(commandAndArguments))
				copy(vm.lastDebuggerCmdAndArgs, commandAndArguments)
			}

			switch v {
			case Next:
				return
			case Continue:
				return
			case StepIn:
				fmt.Println(commandAndArguments[0])
			case StepOut:
				fmt.Println(commandAndArguments[0])
			case Exec:
				vm.debuggerExec = true
				value := vm.evalCode(strings.Join(commandAndArguments[1:], ";"))
				fmt.Printf("< Return: %s\n", value.ToString())
				vm.debuggerExec = false
				return
			case List:
				fmt.Println(vm.printSource())
			case Print:
				// fmt.Println(commandAndArguments[0])
				// fmt.Println(commandAndArguments[1:])
				fmt.Println(vm.prg.values)
			case Help:
				for _, value := range debuggerHelp {
					fmt.Println(value)
				}
			case Quit:
				os.Exit(0)
			default:
				os.Exit(0)
			}
		} else {
			fmt.Println("unknown command")
		}
	}
}
