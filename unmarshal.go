package sx

import (
	"fmt"
	"reflect"
	"strconv"
)

type Unmarshaler interface {
	UnmarshalSX(tree []Node) error
}

// returns true if tree's length is 1 and the only node is a scalar
func isTreeScalar(tree []Node) bool {
	if len(tree) != 1 {
		return false
	}
	return tree[0].IsScalar()
}

// returns true if tree's length is 1 and the only node is a list
func isTreeList(tree []Node) bool {
	if len(tree) != 1 {
		return false
	}
	return !tree[0].IsScalar()
}

// Perform an optional tree indirection
func indirectMap(tree []Node) []Node {
	if isTreeList(tree) {
		// I use square brackets here to emphasize what is 'tree'.
		//   (a [(a)]) as is
		//   (a [()]) indirect to (a ([]))
		//   (a [(())]) indirect to (a ([()]))
		if t := tree[0].List; len(t) == 0 || !t[0].IsScalar() {
			return t
		}
	}
	return tree
}

func tryUnmarshaler(tree []Node, v reflect.Value) (bool, error) {
	u, ok := v.Interface().(Unmarshaler)
	if !ok {
		// T doesn't work, try *T as well
		if v.Kind() != reflect.Ptr && v.CanAddr() {
			u, ok = v.Addr().Interface().(Unmarshaler)
		}
	}

	if ok {
		return true, u.UnmarshalSX(tree)
	}
	return false, nil
}

func unmarshalValue(tree []Node, v reflect.Value) error {
	t := v.Type()

	// one level of indirection is supported
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(t.Elem()))
		}
		v = v.Elem()
		t = t.Elem()
	}

	if ok, err := tryUnmarshaler(tree, v); ok {
		return err
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if !isTreeScalar(tree) {
			return fmt.Errorf("scalar node expected")
		}
		num, err := strconv.ParseInt(tree[0].Value, 10, 64)
		if err != nil {
			return fmt.Errorf("node is not an integer")
		}
		if v.OverflowInt(num) {
			return fmt.Errorf("integer overflow")
		}
		v.SetInt(num)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if !isTreeScalar(tree) {
			return fmt.Errorf("scalar node expected")
		}
		num, err := strconv.ParseUint(tree[0].Value, 10, 64)
		if err != nil {
			return fmt.Errorf("node is not an unsigned integer")
		}
		if v.OverflowUint(num) {
			return fmt.Errorf("unsigned integer overflow")
		}
		v.SetUint(num)
	case reflect.Float32, reflect.Float64:
		if !isTreeScalar(tree) {
			return fmt.Errorf("scalar node expected")
		}
		num, err := strconv.ParseFloat(tree[0].Value, 64)
		if err != nil {
			return fmt.Errorf("node is not a floating point number")
		}
		v.SetFloat(num)
	case reflect.Bool:
		if !isTreeScalar(tree) {
			return fmt.Errorf("scalar node expected")
		}
		switch tree[0].Value {
		case "true":
			v.SetBool(true)
		case "false":
			v.SetBool(false)
		default:
			return fmt.Errorf("invalid boolean value, use true|false")
		}
	case reflect.String:
		if !isTreeScalar(tree) {
			return fmt.Errorf("scalar node expected")
		}
		v.SetString(tree[0].Value)
	case reflect.Array, reflect.Slice:
		isArray := v.Kind() == reflect.Array
		if isTreeList(tree) {
			// Indirection for cases like this:
			//   (a (1 2 3)) vs (a 1 2 3)
			tree = tree[0].List
		}
		if !isArray {
			// Create a brand new slice, we don't want to overwrite someone
			// else's slice accident. Sadly, this also means you cannot reuse the
			// slice. Nothing stops you from implementing Unmarshaler interface
			// though.
			v.Set(reflect.MakeSlice(t, len(tree), len(tree)))
		} else {
			if len(tree) > v.Len() {
				tree = tree[:v.Len()]
			}
		}
		for i := range tree {
			if err := unmarshalValue(tree[i:i+1], v.Index(i)); err != nil {
				return err
			}
		}

		// This is only possible when v.Kind() == reflect.Array
		if vlen := v.Len(); len(tree) < vlen {
			zero := reflect.Zero(t.Elem())
			for i := len(tree); i < vlen; i++ {
				v.Index(i).Set(zero)
			}
		}
	case reflect.Map:
		v.Set(reflect.MakeMap(t))
		keyv := reflect.New(t.Key()).Elem()
		valv := reflect.New(t.Elem()).Elem()
		for _, node := range indirectMap(tree) {
			if node.IsScalar() {
				return fmt.Errorf("map element must be represented via (key value...) list")
			}
			list := node.List
			if len(list) < 2 {
				return fmt.Errorf("valid map element list must contain at least two items")
			}
			if err := unmarshalValue(list[:1], keyv); err != nil {
				return fmt.Errorf("key unmarshaling failure: %s", err)
			}
			if err := unmarshalValue(list[1:], valv); err != nil {
				return fmt.Errorf("value unmarshaling failure: %s", err)
			}
			v.SetMapIndex(keyv, valv)
		}
	case reflect.Struct:
		for _, node := range indirectMap(tree) {
			if node.IsScalar() {
				return fmt.Errorf("struct field must be represented via (name value...) list")
			}
			list := node.List
			if len(list) < 2 {
				return fmt.Errorf("valid struct field list must contain at least two items")
			}
			if !list[0].IsScalar() {
				return fmt.Errorf("first element of the struct field list must be scalar")
			}
			name := list[0].Value
			var f reflect.StructField
			var ok bool
			for i, n := 0, t.NumField(); i < n; i++ {
				f = t.Field(i)
				tag := f.Tag.Get("sx")
				if tag == "-" {
					continue
				}
				if f.Anonymous {
					continue
				}
				if ok = tag == name; ok {
					break
				}
				if ok = f.Name == name; ok {
					break
				}
			}
			if ok {
				if f.PkgPath != "" {
					return fmt.Errorf("writing to unexported field")
				}
				if err := unmarshalValue(list[1:], v.FieldByIndex(f.Index)); err != nil {
					return err
				}
			}
		}
	default:
		return fmt.Errorf("unsupported type")
	}
	return nil
}

// Parse and unmarshal sx data into a value pointed to by 'out', hence 'out'
// must be a pointer.
func Unmarshal(data []byte, out interface{}) error {
	tree, err := Parse(data)
	if err != nil {
		return err
	}

	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		// This is a library user mistake, not a usual error
		panic("sx.Unmarshal expects a non-nil pointer as 'out' argument")
	}

	return unmarshalValue(tree, v.Elem())
}
