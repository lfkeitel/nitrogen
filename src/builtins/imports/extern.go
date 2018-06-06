package imports

import (
	"os"
	"path/filepath"

	"github.com/nitrogen-lang/nitrogen/src/ast"
	"github.com/nitrogen-lang/nitrogen/src/compiler"
	"github.com/nitrogen-lang/nitrogen/src/lexer"
	"github.com/nitrogen-lang/nitrogen/src/moduleutils"
	"github.com/nitrogen-lang/nitrogen/src/object"
	"github.com/nitrogen-lang/nitrogen/src/parser"
	"github.com/nitrogen-lang/nitrogen/src/vm"
)

var included map[string]*ast.Program

func init() {
	vm.RegisterBuiltin("include", includeScript)
	vm.RegisterBuiltin("require", requireScript)
	vm.RegisterBuiltin("evalScript", evalScript)

	included = make(map[string]*ast.Program)
}

func includeScript(interpreter object.Interpreter, env *object.Environment, args ...object.Object) object.Object {
	return commonInclude(false, true, interpreter, env, args...)
}

func requireScript(interpreter object.Interpreter, env *object.Environment, args ...object.Object) object.Object {
	return commonInclude(true, true, interpreter, env, args...)
}

func evalScript(interpreter object.Interpreter, env *object.Environment, args ...object.Object) object.Object {
	cleanEnv := object.NewEnvironment()

	envvar, _ := env.Get("_ARGV")
	cleanEnv.CreateConst("_ARGV", envvar.Dup())

	envvar, _ = env.Get("_ENV")
	cleanEnv.CreateConst("_ENV", envvar.Dup())

	return commonInclude(false, false, interpreter, cleanEnv, args...)
}

func commonInclude(require bool, save bool, i object.Interpreter, env *object.Environment, args ...object.Object) object.Object {
	funcName := "include"
	if require {
		funcName = "require"
	}

	if ac := moduleutils.CheckMinArgs(funcName, 1, args...); ac != nil {
		return ac
	}

	filepathArg, ok := args[0].(*object.String)
	if !ok {
		return object.NewException("%s expected a string, got %s", funcName, args[0].Type().String())
	}

	once := false
	if len(args) > 1 {
		includeOnce, ok := args[1].(*object.Boolean)
		if !ok {
			return object.NewException("%s expected a boolean for second argument, got %s", funcName, args[1].Type().String())
		}
		once = includeOnce.Value
	}

	includedFile := filepath.Clean(filepath.Join(filepath.Dir(i.GetCurrentScriptPath()), filepathArg.Value))

	program, exists := included[includedFile]
	if exists {
		if once || program == nil {
			return object.NullConst
		}
		return i.Eval(program, object.NewEnclosedEnv(env))
	}

	l, err := lexer.NewFile(includedFile)
	if err != nil {
		if require {
			return object.NewException("including %s failed %s", includedFile, err.Error())
		}
		return object.NewError("including %s failed %s", includedFile, err.Error())
	}

	p := parser.New(l, moduleutils.ParserSettings)
	program = p.ParseProgram()
	if len(p.Errors()) != 0 {
		if require {
			return object.NewException("including %s failed %s", includedFile, p.Errors()[0])
		}
		return object.NewError("including %s failed %s", includedFile, p.Errors()[0])
	}

	if save {
		if once {
			// Create the key, but don't save the parsed script since we don't need it anymore.
			included[includedFile] = nil
		} else {
			included[includedFile] = program
		}
	}

	switch i := i.(type) {
	case *vm.VirtualMachine:
		code := compiler.Compile(program, "<included>")
		env = object.NewEnclosedEnv(env)
		env.CreateConst("_FILE", object.MakeStringObj(code.Filename))
		return i.RunFrame(i.MakeFrame(code, env), true)
	}

	return object.NewPanic("Invalid interpreter")
}

func fileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}
