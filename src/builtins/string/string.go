package string

import (
	"strings"

	"github.com/nitrogen-lang/nitrogen/src/moduleutils"
	"github.com/nitrogen-lang/nitrogen/src/object"
	"github.com/nitrogen-lang/nitrogen/src/vm"
)

var moduleName = "string"

func init() {
	vm.RegisterModule(moduleName, &object.Module{
		Name:    moduleName,
		Methods: map[string]object.BuiltinFunction{},
		Vars: map[string]object.Object{
			"String": &vm.BuiltinClass{
				Fields: map[string]object.Object{
					"str": object.NullConst,
				},
				VMClass: &vm.VMClass{
					Name:   "string",
					Parent: nil,
					Methods: map[string]object.ClassMethod{
						"init":      vm.MakeBuiltinMethod(vmStringInit),
						"splitN":    vm.MakeBuiltinMethod(vmStrSplitN),
						"trimSpace": vm.MakeBuiltinMethod(vmStrTrim),
						"dedup":     vm.MakeBuiltinMethod(vmStrDedup),
						"format":    vm.MakeBuiltinMethod(vmStrFormat),
					},
				},
			},
		},
	})
}

func dedupString(str []rune, c rune) string {
	newstr := make([]rune, 0, int(float32(len(str))*0.75))

	var lastc rune
	for _, char := range str {
		if char == c && char == lastc {
			continue
		}
		newstr = append(newstr, char)
		lastc = char
	}

	return string(newstr)
}

func vmStringInit(interpreter *vm.VirtualMachine, self *vm.VMInstance, env *object.Environment, args ...object.Object) object.Object {
	_, ok := args[0].(*object.String)
	if !ok {
		return object.NewException("string expected a string, got %s", args[1].Type().String())
	}

	env.Set("str", args[0])
	return nil
}

func vmStrSplitN(interpreter *vm.VirtualMachine, self *vm.VMInstance, env *object.Environment, args ...object.Object) object.Object {
	if ac := moduleutils.CheckArgs("strSplitN", 2, args...); ac != nil {
		return ac
	}

	selfStr, _ := self.Fields.Get("str")
	target, ok := selfStr.(*object.String)
	if !ok {
		return object.NewException("str field is not a string")
	}

	sep, ok := args[0].(*object.String)
	if !ok {
		return object.NewException("splitN expected a string, got %s", args[1].Type().String())
	}

	count, ok := args[1].(*object.Integer)
	if !ok {
		return object.NewException("splitN expected an int, got %s", args[1].Type().String())
	}

	return object.MakeStringArray(strings.SplitN(target.String(), sep.String(), int(count.Value)))
}

func vmStrTrim(interpreter *vm.VirtualMachine, self *vm.VMInstance, env *object.Environment, args ...object.Object) object.Object {
	selfStr, _ := self.Fields.Get("str")
	target, ok := selfStr.(*object.String)
	if !ok {
		return object.NewException("str field is not a string")
	}

	return object.MakeStringObj(strings.TrimSpace(target.String()))
}

func vmStrDedup(interpreter *vm.VirtualMachine, self *vm.VMInstance, env *object.Environment, args ...object.Object) object.Object {
	if ac := moduleutils.CheckArgs("strDedup", 1, args...); ac != nil {
		return ac
	}

	selfStr, _ := self.Fields.Get("str")
	target, ok := selfStr.(*object.String)
	if !ok {
		return object.NewException("str field is not a string")
	}

	dedup, ok := args[0].(*object.String)
	if !ok {
		return object.NewException("strDedup expected a string, got %s", args[0].Type().String())
	}

	if len(dedup.Value) != 1 {
		return object.NewException("Dedup string must be one byte")
	}

	return object.MakeStringObj(dedupString(target.Value, dedup.Value[0]))
}

func vmStrFormat(interpreter *vm.VirtualMachine, self *vm.VMInstance, env *object.Environment, args ...object.Object) object.Object {
	selfStr, _ := self.Fields.Get("str")
	target, ok := selfStr.(*object.String)
	if !ok {
		return object.NewException("str field is not a string")
	}

	t := string(target.Value)

	for _, arg := range args {
		if !strings.Contains(t, "{}") {
			break
		}

		s := objectToString(arg, interpreter)
		t = strings.Replace(t, "{}", s, 1)
	}

	return object.MakeStringObj(t)
}

func objectToString(obj object.Object, machine *vm.VirtualMachine) string {
	if instance, ok := obj.(*vm.VMInstance); ok {
		toString := instance.GetBoundMethod("toString")
		if toString != nil {
			machine.CallFunction(0, toString, true, nil)
			return objectToString(machine.PopStack(), machine)
		}
	}

	return obj.Inspect()
}