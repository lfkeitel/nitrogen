package dis

import (
	"fmt"

	"github.com/nitrogen-lang/nitrogen/src/moduleutils"
	"github.com/nitrogen-lang/nitrogen/src/object"
	"github.com/nitrogen-lang/nitrogen/src/vm"
)

func init() {
	vm.RegisterBuiltin("dis", disassemble)
}

func disassemble(machine object.Interpreter, env *object.Environment, args ...object.Object) object.Object {
	if ac := moduleutils.CheckMinArgs("dis", 1, args...); ac != nil {
		return ac
	}

	fn, ok := args[0].(*vm.VMFunction)
	if !ok {
		return object.NewException("dis expected a func, got %s", args[0].Type().String())
	}

	cb := fn.Body

	fmt.Printf("Name: %s\nFilename: %s\nLocalCount: %d\nMaxStackSize: %d\nMaxBlockSize: %d\n",
		cb.Name, cb.Filename, cb.LocalCount, cb.MaxStackSize, cb.MaxBlockSize)
	cb.Print(" ")
	return object.NullConst
}