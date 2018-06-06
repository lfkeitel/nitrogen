package main

import (
	"strings"

	"github.com/nitrogen-lang/nitrogen/src/moduleutils"
	"github.com/nitrogen-lang/nitrogen/src/object"
	"github.com/nitrogen-lang/nitrogen/src/vm"
)

func init() {
	stringsModule := &object.Module{
		Name: ModuleName,
		Methods: map[string]object.BuiltinFunction{
			"splitN":    strSplitN,
			"trimSpace": strTrim,
			"dedup":     strDedup,
		},
		Vars: map[string]object.Object{
			"name": object.MakeStringObj(ModuleName),
		},
	}

	vm.RegisterModule(ModuleName, stringsModule)
}

func main() {}

var ModuleName = "strings"

func strSplitN(interpreter object.Interpreter, env *object.Environment, args ...object.Object) object.Object {
	if ac := moduleutils.CheckArgs("strSplitN", 3, args...); ac != nil {
		return ac
	}

	target, ok := args[0].(*object.String)
	if !ok {
		return object.NewException("strSplitN expected a string, got %s", args[0].Type().String())
	}

	sep, ok := args[1].(*object.String)
	if !ok {
		return object.NewException("strSplitN expected a string, got %s", args[1].Type().String())
	}

	count, ok := args[2].(*object.Integer)
	if !ok {
		return object.NewException("strSplitN expected an int, got %s", args[1].Type().String())
	}

	return object.MakeStringArray(strings.SplitN(target.Value, sep.Value, int(count.Value)))
}

func strTrim(interpreter object.Interpreter, env *object.Environment, args ...object.Object) object.Object {
	if ac := moduleutils.CheckArgs("strTrim", 1, args...); ac != nil {
		return ac
	}

	target, ok := args[0].(*object.String)
	if !ok {
		return object.NewException("strTrim expected a string, got %s", args[0].Type().String())
	}

	return object.MakeStringObj(strings.TrimSpace(target.Value))
}

func strDedup(interpreter object.Interpreter, env *object.Environment, args ...object.Object) object.Object {
	if ac := moduleutils.CheckArgs("strDedup", 2, args...); ac != nil {
		return ac
	}

	target, ok := args[0].(*object.String)
	if !ok {
		return object.NewException("strDedup expected a string, got %s", args[0].Type().String())
	}

	dedup, ok := args[1].(*object.String)
	if !ok {
		return object.NewException("strDedup expected a string, got %s", args[0].Type().String())
	}

	if len(dedup.Value) != 1 {
		return object.NewException("Dedup string must be one byte")
	}

	return object.MakeStringObj(dedupString(target.Value, dedup.Value[0]))
}

func dedupString(str string, c byte) string {
	bstr := []byte(str)
	newstr := make([]byte, 0, int(float32(len(str))*0.75))

	var lastc byte
	for _, char := range bstr {
		if char == c && char == lastc {
			continue
		}
		newstr = append(newstr, char)
		lastc = char
	}

	return string(newstr)
}
