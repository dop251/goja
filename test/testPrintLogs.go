package main

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

func ScriptEval() {
	vm := goja.New()
	new(require.Registry).Enable(vm)
	script := `
		var a = 5;
		log("Consoling from log method - statement 1")
		log("Consoling from log method - statement 2")
		log("Variable value "+ (a+5))
		a
	`
	prog, err := goja.Compile("", script, true)
	if err != nil {
		fmt.Printf("Error compiling the script %v ", err)
		return
	}
	fmt.Println("Running ... \n ")
	response, _ := vm.RunProgram(prog)
	fmt.Println(response.Result)
	fmt.Println("---------")
	fmt.Printf("%v", response.ConsoleLogs)
}

func main() {
	ScriptEval()
}
