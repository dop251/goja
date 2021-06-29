package goja

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/dop251/goja/parser"
	"github.com/dop251/goja/unistring"
)

type Debugger struct {
	vm *vm

	currentLine  int
	lastLines    []int
	breakpoints  []Breakpoint
	activationCh chan chan string
	active       bool
}

type Result struct {
	Value interface{}
	Err   error
}

func NewDebugger(vm *vm) *Debugger {
	dbg := &Debugger{
		vm:           vm,
		activationCh: make(chan chan string),
		active:       false,
	}
	dbg.lastLines = append(dbg.lastLines, 0)
	return dbg
}

const ( // TODO constants enum
	BreakpointActivation        string = "b"
	DebuggerStatementActivation        = "d"
)

func (dbg *Debugger) activate(s string) {
	dbg.active = true
	ch := <-dbg.activationCh // get channel from waiter
	ch <- s                  // send what activated it
	<-ch                     // wait for deactivation
	dbg.active = false
}

// WaitToActivate  returns what activated debugger and a function to deactivate it resume normal execution/continue
func (dbg *Debugger) WaitToActivate() (string, func()) {
	ch := make(chan string)
	dbg.activationCh <- ch
	r := <-ch
	return r, func() { close(ch) }
}

type Breakpoint struct {
	Filename string
	Line     int
}

func (dbg *Debugger) Wait() *Breakpoint {
	// TODO: implement this
	return &Breakpoint{}
}

func (dbg *Debugger) GetPC() int {
	return dbg.vm.pc
}

func (dbg *Debugger) SetBreakpoint(fileName string, line int) error {
	b := Breakpoint{Filename: fileName, Line: line}
	for _, elem := range dbg.breakpoints {
		if elem == b {
			return errors.New("breakpoint exists")
		}
	}

	dbg.breakpoints = append(dbg.breakpoints, b)

	return nil
}

func (dbg *Debugger) ClearBreakpoint(fileName string, line int) error {
	if len(dbg.breakpoints) == 0 {
		return errors.New("no breakpoints set")
	}

	b := Breakpoint{Filename: fileName, Line: line}
	for idx, elem := range dbg.breakpoints {
		if elem == b {
			dbg.breakpoints = append(dbg.breakpoints[:idx], dbg.breakpoints[idx+1:]...)
			return nil
		}
	}

	return errors.New("cannot set breakpoints")
}

func (dbg *Debugger) Breakpoints() ([]Breakpoint, error) {
	if dbg.breakpoints == nil {
		return nil, errors.New("no breakpoints")
	}

	return dbg.breakpoints, nil
}

func (dbg *Debugger) Next() Result {
	cmd := NextCommand{}
	return cmd.execute(dbg)
}

func (dbg *Debugger) Continue() Result {
	cmd := ContinueCommand{}
	return cmd.execute(dbg)
}

func (dbg *Debugger) StepIn() Result {
	cmd := StepInCommand{}
	return cmd.execute(dbg)
}

func (dbg *Debugger) StepOut() Result {
	cmd := StepOutCommand{}
	return cmd.execute(dbg)
}

func (dbg *Debugger) Exec(expr string) Result {
	cmd := ExecCommand{expression: expr}
	return cmd.execute(dbg)
}

func (dbg *Debugger) Print(varName string) Result {
	cmd := PrintCommand{varName: varName}
	return cmd.execute(dbg)
}

func (dbg *Debugger) List() Result {
	cmd := ListCommand{}
	return cmd.execute(dbg)
}

type Command interface {
	execute() (interface{}, error)
}

type NextCommand struct{}

func (*NextCommand) execute(dbg *Debugger) Result {
	// TODO: implement proper error propagation
	lastLine := dbg.Line()
	dbg.updateCurrentLine()
	if dbg.getLastLine() != dbg.Line() {
		// dbg.REPL(dbg, false)
		// TODO: wait for command
	}
	nextLine := dbg.getNextLine()
	for dbg.isSafeToRun() && dbg.Line() != nextLine {
		dbg.updateCurrentLine()
		if dbg.isDebuggerStatement() {
			break
		}
		dbg.vm.prg.code[dbg.vm.pc].exec(dbg.vm)
	}
	dbg.updateLastLine(lastLine)
	return Result{Value: nil, Err: nil}
}

type ContinueCommand struct{}

func (*ContinueCommand) execute(dbg *Debugger) Result {
	// TODO: implement proper error propagation
	lastLine := dbg.Line()
	dbg.updateCurrentLine()
	for dbg.isSafeToRun() && !dbg.isDebuggerStatement() {
		if dbg.isBreakpoint() {
			// dbg.REPL(dbg, false)
			// TODO: wait for command
			dbg.updateCurrentLine()
			dbg.updateLastLine(lastLine)
			return Result{Value: nil, Err: nil}
		}
		dbg.vm.prg.code[dbg.vm.pc].exec(dbg.vm)
		dbg.updateCurrentLine()
	}
	dbg.updateLastLine(lastLine)
	return Result{Value: nil, Err: nil}
}

type StepInCommand struct{}

func (*StepInCommand) execute(dbg *Debugger) Result {
	return Result{Value: nil, Err: errors.New("not implemented yet")}
}

type StepOutCommand struct{}

func (*StepOutCommand) execute(dbg *Debugger) Result {
	return Result{Value: nil, Err: errors.New("not implemented yet")}
}

type ExecCommand struct {
	expression string
}

func (e *ExecCommand) execute(dbg *Debugger) Result {
	if e.expression == "" {
		return Result{Value: nil, Err: errors.New("nothing to execute")}
	}
	val, err := dbg.eval(e.expression)

	lastLine := dbg.Line()
	dbg.updateLastLine(lastLine)
	return Result{Value: val, Err: err}
}

type PrintCommand struct {
	varName string
}

func (p *PrintCommand) execute(dbg *Debugger) Result {
	if p.varName == "" {
		return Result{Value: "", Err: errors.New("please specify variable name")}
	}
	val, err := dbg.getValue(p.varName)

	if val == Undefined() {
		return Result{Value: fmt.Sprint(dbg.vm.prg.values), Err: err}
	} else {
		// FIXME: val.ToString() causes debugger to exit abruptly
		return Result{Value: fmt.Sprint(val), Err: err}
	}
}

type ListCommand struct{}

func (*ListCommand) execute(dbg *Debugger) Result {
	// TODO probably better to get only some of the lines, but fine for now
	val, err := StringToLines(dbg.vm.prg.src.Source())
	return Result{Value: val, Err: err}
}

type (
	EmptyCommand   struct{}
	NewLineCommand struct{}
)

func StringToLines(s string) (lines []string, err error) {
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	err = scanner.Err()
	return
}

func (dbg *Debugger) isDebuggerStatement() bool {
	return dbg.vm.prg.code[dbg.vm.pc] == debugger
}

func (dbg *Debugger) isNextDebuggerStatement() bool {
	return dbg.vm.pc+1 < len(dbg.vm.prg.code) && dbg.vm.prg.code[dbg.vm.pc+1] == debugger
}

func (dbg *Debugger) isBreakpoint() bool {
	currentLine := dbg.Line()
	currentFilename := dbg.Filename()

	b := Breakpoint{Filename: currentFilename, Line: currentLine}
	for _, elem := range dbg.breakpoints { // TODO have them as map of files to breakpoint list
		if elem == b {
			return true
		}
	}
	return false
}

func (dbg *Debugger) getLastLine() int {
	if len(dbg.lastLines) > 0 {
		return dbg.lastLines[len(dbg.lastLines)-1]
	}
	// First executed line (current line) is considered the last line
	return dbg.Line()
}

func (dbg *Debugger) updateLastLine(lineNumber int) {
	if len(dbg.lastLines) > 0 && dbg.lastLines[len(dbg.lastLines)-1] != lineNumber {
		dbg.lastLines = append(dbg.lastLines, lineNumber)
	}
}

func (dbg *Debugger) Line() int {
	// FIXME: Some lines are skipped, which causes this function to report incorrect lines
	return dbg.vm.prg.src.Position(dbg.vm.prg.sourceOffset(dbg.vm.pc)).Line
}

func (dbg *Debugger) Filename() string {
	return dbg.vm.prg.src.Position(dbg.vm.prg.sourceOffset(dbg.vm.pc)).Filename
}

func (dbg *Debugger) updateCurrentLine() {
	dbg.currentLine = dbg.Line()
}

func (dbg *Debugger) getNextLine() int {
	for idx := range dbg.vm.prg.code[dbg.vm.pc:] {
		nextLine := dbg.vm.prg.src.Position(dbg.vm.prg.sourceOffset(dbg.vm.pc + idx + 1)).Line
		if nextLine > dbg.Line() {
			return nextLine
		}
	}
	return 0
}

func (dbg *Debugger) isSafeToRun() bool {
	return dbg.vm.pc < len(dbg.vm.prg.code)
}

func (dbg *Debugger) eval(expr string) (v Value, err error) {
	prg, err := parser.ParseFile(nil, "<eval>", expr, 0)
	if err != nil {
		return nil, &CompilerSyntaxError{
			CompilerError: CompilerError{
				Message: err.Error(),
			},
		}
	}

	c := newCompiler()

	defer func() {
		if x := recover(); x != nil {
			c.p = nil
			switch ex := x.(type) {
			case *CompilerSyntaxError:
				err = ex
			default:
				err = fmt.Errorf("cannot recover from exception %s", ex)
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
				err = fmt.Errorf("cannot recover from exception %s", x)
			}
		}
		dbg.vm.popCtx()
		dbg.vm.halt = false
		dbg.vm.sp -= 1
	}()

	dbg.vm.pushCtx()
	dbg.vm.prg = c.p
	dbg.vm.pc = 0
	dbg.vm.args = 0
	dbg.vm.result = _undefined
	dbg.vm.sb = dbg.vm.sp
	dbg.vm.push(this)
	dbg.vm.run()
	v = dbg.vm.result
	return v, err
}

func (dbg *Debugger) IsBreakOnStart() bool {
	return dbg.vm.pc <= 3 && dbg.vm.prg.code[2] == debugger
}

func (dbg *Debugger) getValue(varName string) (val Value, err error) {
	name := unistring.String(varName)

	defer func() {
		if x := recover(); x != nil { // TODO better catch exception
			if ex, ok := x.(*uncatchableException); ok {
				err = ex.err
			} else {
				err = fmt.Errorf("cannot recover from exception %s", x)
			}
		}
	}()
	// First try
	for stash := dbg.vm.stash; stash != nil; stash = stash.outer {
		if v, exists := stash.getByName(name); exists {
			val = v
			break
		}
	}

	if val != nil {
		return val, err
	}

	err = errors.New("variable doesn't exist in the global scope")

	// Second try
	if dbg.vm.sb >= 0 {
		val = dbg.vm.stack[dbg.vm.sb]
	}

	if val != nil {
		return val, err
	}

	err = errors.New("variable doesn't exist in the local scope")

	// Third (last) try
	val = dbg.vm.r.globalObject.self.getStr(name, nil)
	if val != nil {
		return val, err
	}

	val = valueUnresolved{r: dbg.vm.r, ref: name}
	err = errors.New("cannot resolve variable")
	return val, err
}
