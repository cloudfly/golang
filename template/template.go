package template

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
	"strings"
	"text/template/parse"
)

var (
	zero      reflect.Value
	errorType = reflect.TypeOf((*error)(nil)).Elem()
)

// Data represent a data source, which contains the value used in template. template will find value in it.
type Data interface {
	Value(string) interface{}
}

// Template is a object parsed from template-string
type Template struct {
	raw   string
	nodes []parse.Node
}

// New create a new template from a string
func New(s string) (*Template, error) {
	t := &Template{
		raw: s,
	}
	sets, err := parse.Parse("default", t.raw, "", "", builtins)
	if err != nil {
		return nil, err
	}
	t.nodes = sets["default"].Root.Nodes
	return t, nil
}

// String return the original template string
func (t *Template) String() string {
	return t.raw
}

// Execute return the golang value which the template represented
func (t *Template) Execute(data Data) (interface{}, error) {
	switch len(t.nodes) {
	case 0:
		return t.raw, nil
	case 1:
		switch n := t.nodes[0].(type) {
		case *parse.ActionNode:
			v, err := evalAction(n, data) // execute the value from the template
			if err != nil {
				return nil, err
			}
			return v.Interface(), nil
		case *parse.TextNode:
			return n.String(), nil
		default:
			return nil, errors.Errorf("unvalid node type %d", n.Type())
		}

	default: // having multiple node in text, means it must be a string value, such as "{{ .a }}{{ .b }}" can not return a single golang value, it must be a string.
		s := ""
		for _, node := range t.nodes {
			switch n := node.(type) {
			case *parse.ActionNode:
				v, err := evalAction(n, data)
				if err != nil {
					return nil, err
				}
				s = fmt.Sprintf("%s%v", s, v.Interface())
			case *parse.TextNode:
				s = fmt.Sprintf("%s%s", s, n.String())
			default:
				return nil, errors.Errorf("unvalid node type %d", n.Type())
			}

		}
		return s, nil
	}
}

// Parse a template string directlly, it is same to call New() and then Execute().
func Parse(s string, data Data) (interface{}, error) {
	tmp, err := New(s)
	if err != nil {
		return nil, err
	}
	return tmp.Execute(data)
}

func evalAction(action *parse.ActionNode, data Data) (value reflect.Value, err error) {
	return evalPipeline(action.Pipe, data)
}

// evalPipeline returns the value acquired by evaluating a pipeline. If the
// pipeline has a variable declaration, the variable will be pushed on the
// stack. Callers should therefore pop the stack after they are finished
// executing commands depending on the pipeline value.
func evalPipeline(pipe *parse.PipeNode, data Data) (value reflect.Value, err error) {
	if pipe == nil {
		return value, errors.New("pipenode is nil")
	}
	for _, cmd := range pipe.Cmds {
		value, err = evalCommand(cmd, data, value) // previous value is this one's final arg.
		if err != nil {
			return value, errors.Wrap(err, "fail to eval command")
		}
		if value.Kind() == reflect.Interface && value.Type().NumMethod() == 0 {
			value = reflect.ValueOf(value.Interface()) // lovely!
		}
	}
	return value, nil
}

func evalCommand(cmd *parse.CommandNode, data Data, final reflect.Value) (reflect.Value, error) {
	firstWord := cmd.Args[0]
	switch n := firstWord.(type) {
	case *parse.FieldNode:
		return evalFieldNode(firstWord.(*parse.FieldNode), data, cmd.Args, final)
	case *parse.ChainNode:
		return evalChainNode(n, data, cmd.Args, final)
	case *parse.IdentifierNode:
		// Must be a function.
		return evalFunction(n, data, cmd, cmd.Args, final)
	case *parse.PipeNode:
		// Parenthesized pipeline. The arguments are all inside the pipeline; final is ignored.
		return evalPipeline(n, data)
	}

	// firstWord is not a function, so it can not having argument, check here.
	if len(cmd.Args) > 1 || final.IsValid() {
		return zero, errors.Errorf("can not give argument to non-function %s", firstWord)
	}

	switch word := firstWord.(type) {
	case *parse.BoolNode:
		return reflect.ValueOf(word.True), nil
	case *parse.NumberNode:
		return idealConstant(word)
	case *parse.StringNode:
		return reflect.ValueOf(word.Text), nil
	case *parse.TextNode:
		return reflect.ValueOf(word.String()), nil
	case *parse.NilNode:
		return zero, errors.New("nil is not a command")
	}
	return zero, errors.Errorf("can't evaluate command %q", firstWord)

}

func evalFieldNode(field *parse.FieldNode, data Data, args []parse.Node, final reflect.Value) (reflect.Value, error) {
	return evalFieldChain(field, data, field.Ident, args, final, zero)
}

func evalChainNode(chain *parse.ChainNode, data Data, args []parse.Node, final reflect.Value) (reflect.Value, error) {
	if len(chain.Field) == 0 {
		return zero, errors.New("internal error: no fields in evalChainNode")
	}
	if chain.Node.Type() == parse.NodeNil {
		return zero, errors.Errorf("indirection through explicit nil in %s", chain)
	}
	// (pipe).Field1.Field2 has pipe as .Node, fields as .Field. Eval the pipeline, then the fields.
	pipe, err := evalArg(chain.Node, data, nil)
	if err != nil {
		return zero, err
	}
	return evalFieldChain(chain, data, chain.Field, args, final, pipe)
}

// evalFieldChain evaluates .X.Y.Z possibly followed by arguments.
// receiver is the value being walked along the chain.
func evalFieldChain(node parse.Node, data Data, ident []string, args []parse.Node, final, receiver reflect.Value) (reflect.Value, error) {
	n, from := len(ident), 0
	if n == 0 {
		return zero, errors.New("no field defined in fieldNode")
	}
	if !receiver.IsValid() {
		if tmp := data.Value(ident[0]); tmp != nil {
			receiver = reflect.ValueOf(tmp)
		} else {
			return zero, errors.Errorf("value %s not found or is nil", ident[0])
		}
		// TODO can not return it directlly, it may be a function
		if n == 1 {
			// check if it's a function
			if receiver.IsValid() && receiver.Type().Kind() == reflect.Func {
				return evalCall(receiver, args, data, ident[0], final)
			}
			return receiver, nil
		}
		from = 1
	}
	var err error
	for i := from; i < n-1; i++ {
		receiver, err = evalField(node, data, ident[i], nil, zero, receiver)
		if err != nil {
			return zero, err
		}
	}
	// Now if it's a method, it gets the arguments.
	return evalField(node, data, ident[n-1], args, final, receiver)
}

// evalField evaluates an expression like (.Field) or (.Field arg1 arg2).
// The 'final' argument represents the return value from the preceding
// value of the pipeline, if any.
func evalField(node parse.Node, data Data, fieldName string, args []parse.Node, final, receiver reflect.Value) (reflect.Value, error) {
	if !receiver.IsValid() {
		return zero, errors.New("unvalid receiver value")
	}
	typ := receiver.Type()
	receiver, _ = indirect(receiver)
	// Unless it's an interface, need to get to a value of type *T to guarantee
	// we see all methods of T and *T.
	ptr := receiver
	if ptr.Kind() != reflect.Interface && ptr.CanAddr() {
		ptr = ptr.Addr()
	}
	if method := ptr.MethodByName(fieldName); method.IsValid() {
		return evalCall(method, args, data, fieldName, final)
	}
	hasArgs := len(args) > 1 || final.IsValid()
	// It's not a method; must be a field of a struct or an element of a map. The receiver must not be nil.
	receiver, isNil := indirect(receiver)
	if isNil {
		return zero, errors.Errorf("nil pointer evaluating %s.%s", typ, fieldName)
	}
	switch receiver.Kind() {
	case reflect.Struct:
		tField, ok := receiver.Type().FieldByName(fieldName)
		if ok {
			field := receiver.FieldByIndex(tField.Index)
			if tField.PkgPath != "" { // field is unexported
				return zero, errors.Errorf("%s is an unexported field of struct type %s", fieldName, typ)
			}
			// If it's a function, we must call it.
			if hasArgs {
				return zero, errors.Errorf("%s has arguments but cannot be invoked as function", fieldName)
			}
			return field, nil
		}
		return zero, errors.Errorf("%s is not a field of struct type %s", fieldName, typ)
	case reflect.Map:
		// If it's a map, attempt to use the field name as a key.
		nameVal := reflect.ValueOf(fieldName)
		if nameVal.Type().AssignableTo(receiver.Type().Key()) {
			if hasArgs {
				return zero, errors.Errorf("%s is not a method but has arguments", fieldName)
			}
			result := receiver.MapIndex(nameVal)
			if !result.IsValid() {
				return zero, errors.Errorf("map has no entry for key %q", fieldName)
			}
			return result, nil
		}
	}
	return zero, errors.Errorf("can't evaluate field %s in type %s", fieldName, typ)
}

func evalFunction(node *parse.IdentifierNode, data Data, cmd parse.Node, args []parse.Node, final reflect.Value) (reflect.Value, error) {
	name := node.Ident
	function, ok := builtins[name]
	if !ok {
		return zero, errors.Errorf("%q is not a defined function", name)
	}
	return evalCall(reflect.ValueOf(function), args, data, name, final)
}

func evalArg(n parse.Node, data Data, typ reflect.Type) (reflect.Value, error) {
	switch arg := n.(type) {
	case *parse.NilNode:
		if canBeNil(typ) {
			return reflect.Zero(typ), nil
		}
		return zero, errors.Errorf("cannot assign nil to %s", typ)
	case *parse.FieldNode:
		tmp, err := evalFieldNode(arg, data, []parse.Node{n}, zero)
		if err != nil {
			return zero, err
		}
		return validateType(tmp, typ)
	case *parse.PipeNode:
		tmp, err := evalPipeline(arg, data)
		if err != nil {
			return zero, err
		}
		return validateType(tmp, typ)
	case *parse.IdentifierNode:
		tmp, err := evalFunction(arg, data, arg, nil, zero)
		if err != nil {
			return zero, err
		}
		return validateType(tmp, typ)
	case *parse.ChainNode:
		tmp, err := evalChainNode(arg, data, nil, zero)
		if err != nil {
			return zero, err
		}
		return validateType(tmp, typ)
	}
	switch typ.Kind() {
	case reflect.Bool:
		return evalBool(typ, n)
	case reflect.Complex64, reflect.Complex128:
		return evalComplex(typ, n)
	case reflect.Float32, reflect.Float64:
		return evalFloat(typ, n)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return evalInteger(typ, n)
	case reflect.Interface:
		if typ.NumMethod() == 0 {
			return evalEmptyInterface(n, data)
		}
	case reflect.String:
		return evalString(typ, n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return evalUnsignedInteger(typ, n)
	}
	return zero, errors.Errorf("can't handle %s for arg of type %s", n, typ)
}

func evalBool(typ reflect.Type, n parse.Node) (reflect.Value, error) {
	if n, ok := n.(*parse.BoolNode); ok {
		value := reflect.New(typ).Elem()
		value.SetBool(n.True)
		return value, nil
	}
	return zero, errors.Errorf("expected bool; found %s", n)
}

func evalString(typ reflect.Type, n parse.Node) (reflect.Value, error) {
	if n, ok := n.(*parse.StringNode); ok {
		value := reflect.New(typ).Elem()
		value.SetString(n.Text)
		return value, nil
	}
	return zero, errors.Errorf("expected string; found %s", n)
}

func evalInteger(typ reflect.Type, n parse.Node) (reflect.Value, error) {
	if n, ok := n.(*parse.NumberNode); ok && n.IsInt {
		value := reflect.New(typ).Elem()
		value.SetInt(n.Int64)
		return value, nil
	}
	return zero, errors.Errorf("expected integer; found %s", n)
}

func evalUnsignedInteger(typ reflect.Type, n parse.Node) (reflect.Value, error) {
	if n, ok := n.(*parse.NumberNode); ok && n.IsUint {
		value := reflect.New(typ).Elem()
		value.SetUint(n.Uint64)
		return value, nil
	}
	return zero, errors.Errorf("expected unsigned integer; found %s", n)
}

func evalFloat(typ reflect.Type, n parse.Node) (reflect.Value, error) {
	if n, ok := n.(*parse.NumberNode); ok && n.IsFloat {
		value := reflect.New(typ).Elem()
		value.SetFloat(n.Float64)
		return value, nil
	}
	return zero, errors.Errorf("expected float; found %s", n)
}

func evalComplex(typ reflect.Type, n parse.Node) (reflect.Value, error) {
	if n, ok := n.(*parse.NumberNode); ok && n.IsComplex {
		value := reflect.New(typ).Elem()
		value.SetComplex(n.Complex128)
		return value, nil
	}
	return zero, errors.Errorf("expected complex; found %s", n)
}

func evalEmptyInterface(n parse.Node, data Data) (reflect.Value, error) {
	switch n := n.(type) {
	case *parse.BoolNode:
		return reflect.ValueOf(n.True), nil
	case *parse.FieldNode:
		return evalFieldNode(n, data, nil, zero)
	case *parse.IdentifierNode:
		return evalFunction(n, data, n, nil, zero)
	case *parse.NilNode:
		// NilNode is handled in evalArg, the only place that calls here.
		return zero, errors.Errorf("evalEmptyInterface: nil (can't happen)")
	case *parse.NumberNode:
		return idealConstant(n)
	case *parse.StringNode:
		return reflect.ValueOf(n.Text), nil
	case *parse.PipeNode:
		return evalPipeline(n, data)
	}
	return zero, errors.Errorf("can't handle assignment of %s to empty interface argument", n)
}

// evalCall executes a function or method call. If it's a method, fun already has the receiver bound, so
// it looks just like a function call.  The arg list, if non-nil, includes (in the manner of the shell), arg[0]
// as the function itself.
func evalCall(fun reflect.Value, args []parse.Node, data Data, name string, final reflect.Value) (reflect.Value, error) {
	var err error
	if args != nil {
		args = args[1:] // Zeroth arg is function name/node; not passed to function.
	}
	typ := fun.Type()
	numIn := len(args)
	if final.IsValid() {
		numIn++
	}
	numFixed := len(args)
	if typ.IsVariadic() {
		numFixed = typ.NumIn() - 1 // last arg is the variadic one.
		if numIn < numFixed {
			return zero, errors.Errorf("wrong number of args for %s: want at least %d got %d", name, typ.NumIn()-1, len(args))
		}
	} else if numIn < typ.NumIn()-1 || !typ.IsVariadic() && numIn != typ.NumIn() {
		return zero, errors.Errorf("wrong number of args for %s: want %d got %d", name, typ.NumIn(), len(args))
	}
	if !goodFunc(typ) {
		// TODO: This could still be a confusing error; maybe goodFunc should provide info.
		return zero, errors.Errorf("can't call method/function %q with %d results", name, typ.NumOut())
	}
	// Build the arg list.
	argv := make([]reflect.Value, numIn)
	// Args must be evaluated. Fixed args first.
	i := 0
	for ; i < numFixed && i < len(args); i++ {
		argv[i], err = evalArg(args[i], data, typ.In(i))
		if err != nil {
			return zero, err
		}
	}
	// Now the ... args.
	if typ.IsVariadic() {
		argType := typ.In(typ.NumIn() - 1).Elem() // Argument is a slice.
		for ; i < len(args); i++ {
			argv[i], err = evalArg(args[i], data, argType)
			if err != nil {
				return zero, err
			}
		}
	}
	// Add final value if necessary.
	if final.IsValid() {
		t := typ.In(typ.NumIn() - 1)
		if typ.IsVariadic() {
			if numIn-1 < numFixed {
				// The added final argument corresponds to a fixed parameter of the function.
				// Validate against the type of the actual parameter.
				t = typ.In(numIn - 1)
			} else {
				// The added final argument corresponds to the variadic part.
				// Validate against the type of the elements of the variadic slice.
				t = t.Elem()
			}
		}
		argv[i], err = validateType(final, t)
		if err != nil {
			return zero, err
		}
	}
	result := fun.Call(argv)
	// If we have an error that is not nil, stop execution and return that error to the caller.
	if len(result) == 2 && !result[1].IsNil() {
		return zero, errors.Errorf("error calling %s: %s", name, result[1].Interface().(error))
	}
	return result[0], nil
}

// idealConstant is called to return the value of a number in a context where
// we don't know the type. In that case, the syntax of the number tells us
// its type, and we use Go rules to resolve.  Note there is no such thing as
// a uint ideal constant in this situation - the value must be of int type.
func idealConstant(constant *parse.NumberNode) (reflect.Value, error) {
	// These are ideal constants but we don't know the type
	// and we have no context.  (If it was a method argument,
	// we'd know what we need.) The syntax guides us to some extent.
	switch {
	case constant.IsComplex:
		return reflect.ValueOf(constant.Complex128), nil // incontrovertible.
	case constant.IsFloat && !isHexConstant(constant.Text) && strings.IndexAny(constant.Text, ".eE") >= 0:
		return reflect.ValueOf(constant.Float64), nil
	case constant.IsInt:
		n := int(constant.Int64)
		if int64(n) != constant.Int64 {
			return zero, errors.Errorf("%s overflows int", constant.Text)
		}
		return reflect.ValueOf(n), nil
	case constant.IsUint:
		return zero, errors.Errorf("%s overflows int", constant.Text)
	}
	return zero, nil
}

// indirect returns the item at the end of indirection, and a bool to indicate if it's nil.
// We indirect through pointers and empty interfaces (only) because
// non-empty interfaces have methods we might need.
func indirect(v reflect.Value) (rv reflect.Value, isNil bool) {
	for ; v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface; v = v.Elem() {
		if v.IsNil() {
			return v, true
		}
		if v.Kind() == reflect.Interface && v.NumMethod() > 0 {
			break
		}
	}
	return v, false
}

// canBeNil reports whether an untyped nil can be assigned to the type. See reflect.Zero.
func canBeNil(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true
	}
	return false
}

// validateType guarantees that the value is valid and assignable to the type.
func validateType(value reflect.Value, typ reflect.Type) (reflect.Value, error) {
	if !value.IsValid() {
		if typ == nil || canBeNil(typ) {
			// An untyped nil interface{}. Accept as a proper nil value.
			return reflect.Zero(typ), nil
		}
		return zero, errors.Errorf("invalid value; expected %s", typ)
	}
	if typ != nil && !value.Type().AssignableTo(typ) {
		if value.Kind() == reflect.Interface && !value.IsNil() {
			value = value.Elem()
			if value.Type().AssignableTo(typ) {
				return value, nil
			}
			// fallthrough
		}
		// Does one dereference or indirection work? We could do more, as we
		// do with method receivers, but that gets messy and method receivers
		// are much more constrained, so it makes more sense there than here.
		// Besides, one is almost always all you need.
		switch {
		case value.Kind() == reflect.Ptr && value.Type().Elem().AssignableTo(typ):
			value = value.Elem()
			if !value.IsValid() {
				return zero, errors.Errorf("dereference of nil pointer of type %s", typ)
			}
		case reflect.PtrTo(value.Type()).AssignableTo(typ) && value.CanAddr():
			value = value.Addr()
		default:
			return zero, errors.Errorf("wrong type for value; expected %s; got %s", typ, value.Type())
		}
	}
	return value, nil
}

func isHexConstant(s string) bool {
	return len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X')
}
