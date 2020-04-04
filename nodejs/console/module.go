package console

import (
	"log"

	"github.com/dop251/goja"
	"github.com/dop251/goja/nodejs/require"
	_ "github.com/dop251/goja/nodejs/util"
)

type Console struct {
	runtime *goja.Runtime
	util    *goja.Object
}

func (c *Console) log(call goja.FunctionCall) goja.Value {
	if format, ok := goja.AssertFunction(c.util.Get("format")); ok {
		ret, err := format(c.util, call.Arguments...)
		if err != nil {
			panic(err)
		}

		log.Print(ret.String())
	} else {
		panic(c.runtime.NewTypeError("util.format is not a function"))
	}

	return nil
}

func Require(runtime *goja.Runtime, module *goja.Object) {
	c := &Console{
		runtime: runtime,
	}

	c.util = require.Require(runtime, "util").(*goja.Object)

	o := module.Get("exports").(*goja.Object)
	o.Set("log", c.log)
	o.Set("error", c.log)
	o.Set("warn", c.log)

}

func Enable(runtime *goja.Runtime) {
	runtime.Set("console", require.Require(runtime, "console"))
}

func init() {
	require.RegisterNativeModule("console", Require)
}
