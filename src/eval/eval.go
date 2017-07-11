package eval

import (
	"fmt"

	"github.com/nitrogen-lang/nitrogen/src/ast"
	"github.com/nitrogen-lang/nitrogen/src/object"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {
	// Statements
	case *ast.Program:
		return evalProgram(node, env)
	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)
	case *ast.BlockStatement:
		return evalBlockStatements(node, env)
	case *ast.ReturnStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}
	case *ast.DefStatement:
		if node.Const {
			return assignConstIdentValue(node.Name, node.Value, env)
		}
		return assignIdentValue(node.Name, node.Value, true, env)
	case *ast.AssignStatement:
		return evalAssignment(node, env)

	// Expressions
	case *ast.NullLiteral:
		return NULL
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}
	case *ast.StringLiteral:
		return &object.String{Value: node.Value}
	case *ast.FloatLiteral:
		return &object.Float{Value: node.Value}
	case *ast.Array:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}
	case *ast.Boolean:
		return nativeBoolToBooleanObj(node.Value)
	case *ast.HashLiteral:
		return evalHashLiteral(node, env)
	case *ast.Identifier:
		return evalIdent(node, env)
	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)
	case *ast.InfixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}

		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}

		return evalInfixExpression(node.Operator, left, right)
	case *ast.IfExpression:
		return evalIfExpression(node, env)
	case *ast.FunctionLiteral:
		return &object.Function{
			Parameters: node.Parameters,
			Body:       node.Body,
			Env:        env,
		}
	case *ast.CallExpression:
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}

		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}

		return applyFunction(function, args)
	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}

		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}
		return evalIndexExpression(left, index)
	}

	return nil
}

func evalAssignment(stmt *ast.AssignStatement, env *object.Environment) object.Object {
	if left, ok := stmt.Left.(*ast.IndexExpression); ok {
		return assignIndexedValue(left, stmt.Value, env)
	}

	ident, ok := stmt.Left.(*ast.Identifier)
	if !ok {
		return newError("Invalid variable name, expected identifier, got %s",
			stmt.Left.String())
	}

	return assignIdentValue(ident, stmt.Value, false, env)
}

func assignIdentValue(
	name *ast.Identifier,
	val ast.Expression,
	new bool,
	env *object.Environment) object.Object {
	// Protect builtin functions
	if builtin := getBuiltin(name.Value); builtin != nil {
		return newError(
			"Attempted redeclaration of builtin function '%s'",
			name.Value,
		)
	}

	if !new {
		if _, exists := env.Get(name.Value); !exists {
			return newError("Assignment to uninitialized variable %s", name.Value)
		}
	}

	if env.IsConst(name.Value) {
		return newError("Assignment to declared constant %s", name.Value)
	}

	evaled := Eval(val, env)
	if isError(evaled) {
		return evaled
	}

	// Ignore error since we check for consant above
	env.Set(name.Value, evaled)
	return NULL
}

func assignConstIdentValue(
	name *ast.Identifier,
	val ast.Expression,
	env *object.Environment) object.Object {
	// Protect builtin functions
	if builtin := getBuiltin(name.Value); builtin != nil {
		return newError(
			"Attempted redeclaration of builtin function '%s'",
			name.Value,
		)
	}

	if env.IsConst(name.Value) {
		return newError("Assignment to declared constant %s", name.Value)
	}

	evaled := Eval(val, env)
	if isError(evaled) {
		return evaled
	}

	if !objectIs(evaled, object.INTEGER_OBJ, object.FLOAT_OBJ, object.STRING_OBJ, object.NULL_OBJ, object.BOOLEAN_OBJ) {
		return newError("Constants must be int, float, string, bool or null")
	}

	// Ignore error since we check above
	env.SetConst(name.Value, evaled)
	return NULL
}

func assignIndexedValue(
	e *ast.IndexExpression,
	val ast.Expression,
	env *object.Environment) object.Object {
	indexed := Eval(e.Left, env)
	if isError(indexed) {
		return indexed
	}

	index := Eval(e.Index, env)
	if isError(indexed) {
		return indexed
	}

	switch indexed.Type() {
	case object.ARRAY_OBJ:
		assignArrayIndex(indexed.(*object.Array), index, val, env)
	case object.HASH_OBJ:
		assignHashMapIndex(indexed.(*object.Hash), index, val, env)
	}
	return NULL
}

func assignArrayIndex(
	array *object.Array,
	index object.Object,
	val ast.Expression,
	env *object.Environment) object.Object {

	in, ok := index.(*object.Integer)
	if !ok {
		return newError("Invalid array index type %s", index.(object.Object).Type())
	}

	value := Eval(val, env)
	if isError(value) {
		return value
	}

	if in.Value < 0 || in.Value > int64(len(array.Elements)-1) {
		return newError("Index out of bounds: %s", index.Inspect())
	}

	array.Elements[in.Value] = value
	return NULL
}

func assignHashMapIndex(
	hashmap *object.Hash,
	index object.Object,
	val ast.Expression,
	env *object.Environment) object.Object {

	hashable, ok := index.(object.Hashable)
	if !ok {
		return newError("Invalid index type %s", index.Type())
	}

	value := Eval(val, env)
	if isError(value) {
		return value
	}

	hashmap.Pairs[hashable.HashKey()] = object.HashPair{
		Key:   index,
		Value: value,
	}
	return NULL
}

func evalProgram(p *ast.Program, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range p.Statements {
		result = Eval(statement, env)

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value
		case *object.Error:
			return result
		}
	}

	return result
}

func evalBlockStatements(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = Eval(statement, env)

		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

func evalExpressions(exps []ast.Expression, env *object.Environment) []object.Object {
	var result []object.Object

	for _, e := range exps {
		evaled := Eval(e, env)
		if isError(evaled) {
			return []object.Object{evaled}
		}
		result = append(result, evaled)
	}

	return result
}

func nativeBoolToBooleanObj(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func evalPrefixExpression(op string, right object.Object) object.Object {
	switch op {
	case "!":
		return evalBangOpExpression(right)
	case "-":
		return evalMinusPreOpExpression(right)
	}

	return newError("unknown operator: %s%s", op, right.Type())
}

func evalBangOpExpression(right object.Object) object.Object {
	if right == FALSE || right == NULL {
		return TRUE
	}

	if right.Type() == object.INTEGER_OBJ && right.(*object.Integer).Value == 0 {
		return TRUE
	}

	return FALSE
}

func evalMinusPreOpExpression(right object.Object) object.Object {
	switch right.Type() {
	case object.INTEGER_OBJ:
		value := right.(*object.Integer).Value
		return &object.Integer{Value: -value}
	case object.FLOAT_OBJ:
		value := right.(*object.Float).Value
		return &object.Float{Value: -value}
	}

	return newError("unknown operator: -%s", right.Type())
}

func evalInfixExpression(op string, left, right object.Object) object.Object {
	switch {
	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s", left.Type(), op, right.Type())
	case typesEqualTo(object.INTEGER_OBJ, left, right):
		return evalIntegerInfixExpression(op, left, right)
	case typesEqualTo(object.FLOAT_OBJ, left, right):
		return evalFloatInfixExpression(op, left, right)
	case typesEqualTo(object.STRING_OBJ, left, right):
		return evalStringInfixExpression(op, left, right)
	case typesEqualTo(object.ARRAY_OBJ, left, right):
		return evalArrayInfixExpression(op, left, right)
	case op == "==":
		return nativeBoolToBooleanObj(left == right)
	case op == "!=":
		return nativeBoolToBooleanObj(left != right)
	}

	return newError("unknown operator: %s %s %s", left.Type(), op, right.Type())
}

func evalIntegerInfixExpression(op string, left, right object.Object) object.Object {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value

	switch op {
	case "+":
		return &object.Integer{Value: leftVal + rightVal}
	case "-":
		return &object.Integer{Value: leftVal - rightVal}
	case "*":
		return &object.Integer{Value: leftVal * rightVal}
	case "/":
		return &object.Integer{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToBooleanObj(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObj(leftVal > rightVal)
	case "==":
		return nativeBoolToBooleanObj(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObj(leftVal != rightVal)
	}

	return newError("unknown operator: %s %s %s", left.Type(), op, right.Type())
}

func evalFloatInfixExpression(op string, left, right object.Object) object.Object {
	leftVal := left.(*object.Float).Value
	rightVal := right.(*object.Float).Value

	switch op {
	case "+":
		return &object.Float{Value: leftVal + rightVal}
	case "-":
		return &object.Float{Value: leftVal - rightVal}
	case "*":
		return &object.Float{Value: leftVal * rightVal}
	case "/":
		return &object.Float{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToBooleanObj(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObj(leftVal > rightVal)
	case "==":
		return nativeBoolToBooleanObj(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObj(leftVal != rightVal)
	}

	return newError("unknown operator: %s %s %s", left.Type(), op, right.Type())
}

func evalStringInfixExpression(op string, left, right object.Object) object.Object {
	leftVal := left.(*object.String).Value
	rightVal := right.(*object.String).Value

	switch op {
	case "+":
		return &object.String{Value: leftVal + rightVal}
	case "==":
		return nativeBoolToBooleanObj(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObj(leftVal != rightVal)
	}

	return newError("unknown operator: %s %s %s", left.Type(), op, right.Type())
}

func evalArrayInfixExpression(op string, left, right object.Object) object.Object {
	leftVal := left.(*object.Array)
	rightVal := right.(*object.Array)

	switch op {
	case "+":
		leftLen := len(leftVal.Elements)
		rightLen := len(rightVal.Elements)
		newElements := make([]object.Object, leftLen+rightLen, leftLen+rightLen)
		copy(newElements, leftVal.Elements)
		copy(newElements[leftLen:], rightVal.Elements)
		return &object.Array{Elements: newElements}
	}

	return newError("unknown operator: %s %s %s", left.Type(), op, right.Type())
}

func evalIfExpression(ie *ast.IfExpression, env *object.Environment) object.Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	}

	return NULL
}

func evalIdent(node *ast.Identifier, env *object.Environment) object.Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}
	if builtin := getBuiltin(node.Value); builtin != nil {
		return builtin
	}
	return newError("identifier not found: %s", node.Value)
}

func evalIndexExpression(left, index object.Object) object.Object {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return evalArrayIndexExpression(left, index)
	case left.Type() == object.HASH_OBJ:
		return evalHashIndexExpression(left, index)
	}
	return newError("Index operator not allowed: %s", left.Type())
}

func evalArrayIndexExpression(array, index object.Object) object.Object {
	arrObj := array.(*object.Array)
	idx := index.(*object.Integer).Value
	max := int64(len(arrObj.Elements))

	if idx > max-1 { // Check upper bound
		return NULL
	}

	if idx < 0 { // Check lower bound
		// Convert a negative index to positive
		idx = max + idx

		if idx < 0 { // Check lower bound again
			return NULL
		}
	}

	return arrObj.Elements[idx]
}

func evalHashIndexExpression(hash, index object.Object) object.Object {
	hashObj := hash.(*object.Hash)

	key, ok := index.(object.Hashable)
	if !ok {
		return newError("Invalid map key: %s", index.Type())
	}

	pair, ok := hashObj.Pairs[key.HashKey()]
	if !ok {
		return NULL
	}

	return pair.Value
}

func evalHashLiteral(node *ast.HashLiteral, env *object.Environment) object.Object {
	pairs := make(map[object.HashKey]object.HashPair)

	for keyNode, valueNode := range node.Pairs {
		key := Eval(keyNode, env)
		if isError(key) {
			return key
		}

		hashKey, ok := key.(object.Hashable)
		if !ok {
			return newError("Invalid map key: %s", key.Type())
		}

		value := Eval(valueNode, env)
		if isError(value) {
			return value
		}

		hashed := hashKey.HashKey()
		pairs[hashed] = object.HashPair{Key: key, Value: value}
	}

	return &object.Hash{Pairs: pairs}
}

func applyFunction(fn object.Object, args []object.Object) object.Object {
	switch fn := fn.(type) {
	case *object.Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaled := Eval(fn.Body, extendedEnv)
		return unwrapReturnValue(evaled)
	case *object.Builtin:
		return fn.Fn(args...)
	}

	return newError("%s is not a function", fn.Type())
}

func extendFunctionEnv(fn *object.Function, args []object.Object) *object.Environment {
	env := object.NewEnclosedEnv(fn.Env)

	for i, param := range fn.Parameters {
		env.Set(param.Value, args[i])
	}

	return env
}

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}
	return obj
}

func isTruthy(obj object.Object) bool {
	// Null or false is immediately not true
	if obj == NULL || obj == FALSE {
		return false
	}

	// True is immediately true
	if obj == TRUE {
		return true
	}

	// If the object is an INT, it's truthy if it doesn't equal 0
	if obj.Type() == object.INTEGER_OBJ {
		return (obj.(*object.Integer).Value != 0)
	}

	// Same as above if but with floats
	if obj.Type() == object.FLOAT_OBJ {
		return (obj.(*object.Float).Value != 0.0)
	}

	// Empty string is false, non-empty is true
	if obj.Type() == object.STRING_OBJ {
		return (obj.(*object.String).Value != "")
	}

	// Assume value is false
	return false
}

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func isError(obj object.Object) bool {
	return (obj != nil && obj.Type() == object.ERROR_OBJ)
}

func typesEqualTo(t object.ObjectType, a, b object.Object) bool {
	return (a.Type() == t && b.Type() == t)
}

func objectIs(o object.Object, t ...object.ObjectType) bool {
	for _, ot := range t {
		if o.Type() == ot {
			return true
		}
	}
	return false
}
