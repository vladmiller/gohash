package gohash

import (
	"encoding/binary"
	"fmt"
	"hash"
	"reflect"
	"sort"
)

type Hash []byte

var (
	typeStringer = []byte("stringer")
)

// hashObject recursively processes input and adds variables to [hash.Hash].
//
// Each value is annotated with a type, so false is not equal to 0.
//
// It processes all of the types defined in the spec, except interfaces and channels.
// https://go.dev/ref/spec#Types
func hashObject(input any, hasher hash.Hash, depth int) error {
	if depth > 100 {
		return fmt.Errorf("depth exceeded for type %T. Limit is 100.", input)
	}

	// Preallocate reflect
	rv := reflect.ValueOf(input)
	kind := rv.Kind()

	// [reflect.Kind] is actually a uint, therefore we can use it directly
	var vt [1]byte

	var rt reflect.Type

	pDepth := 0

	// Dereference pointer until we reach an underlying value
	for {
		if pDepth >= 100 {
			return fmt.Errorf("input '%v' of type %T has too many pointers", input, input)
		}

		if kind == reflect.Pointer {
			pDepth++

			rt = rv.Type().Elem()
			rv = rv.Elem()
			kind = rv.Kind()

			continue
		}

		// If we received an interface, for example for variables that are *interface {}(string)
		// we should process them again
		if kind == reflect.Interface {
			rv = rv.Elem()
			if rv.IsValid() {
				rt = rv.Type()
			}
			kind = rv.Kind()
			continue
		}

		break
	}

	vt[0] = byte(kind)

	// If input implements stringer interface, then use it directly
	if rv.IsValid() {
		if s, ok := rv.Interface().(fmt.Stringer); ok {
			hasher.Write(typeStringer)
			hasher.Write([]byte(s.String()))
			return nil
		}
	}

	// Process value
	switch kind {
	// If we received a reflect.Invalid, then we're getting an empty pointer
	case reflect.Invalid:
		// We cannot infer type of nil, however, we need to treat nil and *nil as same values
		if !(rt == nil || rt.Kind() == reflect.Interface) {
			hasher.Write([]byte(rt.String()))
		}

		return nil

	// Unsigned integers
	case reflect.Uint, reflect.Uintptr, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		hasher.Write(vt[:1])
		binary.Write(hasher, binary.LittleEndian, rv.Uint())
		return nil

	// Signed integers
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		hasher.Write(vt[:1])
		binary.Write(hasher, binary.LittleEndian, rv.Int())
		return nil

	// Floats
	case reflect.Float32, reflect.Float64:
		hasher.Write(vt[:1])
		binary.Write(hasher, binary.LittleEndian, rv.Float())
		return nil

	// Complex
	case reflect.Complex64, reflect.Complex128:
		hasher.Write(vt[:1])
		binary.Write(hasher, binary.LittleEndian, rv.Complex())
		return nil

	// Boolean
	case reflect.Bool:
		hasher.Write(vt[:1])
		binary.Write(hasher, binary.LittleEndian, rv.Bool())
		return nil

	// String
	case reflect.String:
		hasher.Write(vt[:1])
		binary.Write(hasher, binary.LittleEndian, rv.String())
		return nil

	// Slice
	case reflect.Array, reflect.Slice:
		// If empty, then do not record any data.
		//
		// In order not to loose type information, we need to use reflection. This produces
		// additional insignificant allocations.
		//
		// This ensures that []byte{} != []string{}
		if rv.Len() == 0 {
			rt := []byte(rv.Type().String())
			hasher.Write(rt)
			return nil
		}

		// Otherwise iterate over items and hash them recursively
		hasher.Write(vt[:1])

		for i := range rv.Len() {
			if err := hashObject(rv.Index(i).Interface(), hasher, depth+1); err != nil {
				return fmt.Errorf("slice: %w", err)
			}
		}

		return nil

	// Maps
	case reflect.Map:
		// If empty, then do not record any data.
		//
		// In order not to loose type information, we need to use reflection. This produces
		// additional insignificant allocations.
		//
		// This ensures that map[string]string != map[int]string
		if rv.Len() == 0 {
			rt := []byte(rv.Type().String())
			hasher.Write(rt)
			return nil
		}

		hasher.Write(vt[:1])

		// Otherwise, sort the keys
		keys := rv.MapKeys()
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		})

		for _, key := range keys {
			hasher.Write([]byte(key.String()))
			value := rv.MapIndex(key)

			if err := hashObject(value.Interface(), hasher, depth+1); err != nil {
				return fmt.Errorf("map value: %w", err)
			}
		}

		return nil

	// Structs
	case reflect.Struct:
		rt := rv.Type()

		// Write type directly
		//
		// We will not be writing key names to save on performance, therefore type name
		// and package path should be enough to uniquely identify the structure.
		hasher.Write([]byte(rt.String()))
		hasher.Write([]byte(rt.PkgPath()))

		nFields := rv.NumField()

		// If empty, then do not record any data.
		//
		// In order not to loose type information, we need to use reflection. This produces
		// additional insignificant allocations.
		//
		// This ensures that map[string]string != map[int]string
		if nFields == 0 {
			return nil
		}

		// Otherwise iterate over items and hash them recursively
		hasher.Write(vt[:1])

		for i := range nFields {
			field := rv.Field(i)

			// Silentry ignore private variables
			if !field.CanInterface() {
				continue
			}

			if err := hashObject(field.Interface(), hasher, depth+1); err != nil {
				return fmt.Errorf("struct: %w", err)
			}
		}

		return nil

	default:
		return fmt.Errorf("unsupported type %T for value %v", input, input)
	}
}

// From accepts any golang value including pointers and recursively converts
// then to a unique hash value.
func From(input any, hasher hash.Hash) (Hash, error) {
	if err := hashObject(input, hasher, 0); err != nil {
		return nil, fmt.Errorf("from: %w", err)
	}

	hash := hasher.Sum(nil)
	return hash, nil
}
