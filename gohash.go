// Package gohash implements [From] method that can generate a unique hash for almost any Golang type.
package gohash

import (
	"encoding/binary"
	"fmt"
	"hash"
	"reflect"
	"sort"
)

type Hash []byte

// compareMapKeys compares two reflect.Values for sorting.
func compareMapKeys(a, b reflect.Value) int {
	// First sort by type name for stability
	aType := a.Type().String()
	bType := b.Type().String()
	if aType != bType {
		if aType < bType {
			return -1
		}
		return 1
	}

	// Same type - compare values
	switch a.Kind() {
	case reflect.String:
		aStr := a.String()
		bStr := b.String()
		if aStr < bStr {
			return -1
		} else if aStr > bStr {
			return 1
		}
		return 0

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		aInt := a.Int()
		bInt := b.Int()
		if aInt < bInt {
			return -1
		} else if aInt > bInt {
			return 1
		}
		return 0

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		aUint := a.Uint()
		bUint := b.Uint()
		if aUint < bUint {
			return -1
		} else if aUint > bUint {
			return 1
		}
		return 0

	case reflect.Float32, reflect.Float64:
		aFloat := a.Float()
		bFloat := b.Float()
		if aFloat < bFloat {
			return -1
		} else if aFloat > bFloat {
			return 1
		}
		return 0

	case reflect.Bool:
		aBool := a.Bool()
		bBool := b.Bool()
		if !aBool && bBool {
			return -1
		} else if aBool && !bBool {
			return 1
		}
		return 0

	default:
		// Fallback to string representation
		aStr := fmt.Sprintf("%v", a.Interface())
		bStr := fmt.Sprintf("%v", b.Interface())
		if aStr < bStr {
			return -1
		} else if aStr > bStr {
			return 1
		}
		return 0
	}
}

// walkObject recursively iterates over structs, maps and arrays and adds values into hasher.
func walkObject(input any, hasher hash.Hash, depth int, visited map[uintptr]bool) error {
	if depth > 100 {
		return fmt.Errorf("depth exceeded for type %T", input)
	}

	// Preallocate reflect
	rv := reflect.ValueOf(input)
	kind := rv.Kind()

	// Check for cycles on pointer-like types
	if kind == reflect.Ptr || kind == reflect.Map ||
		kind == reflect.Slice || kind == reflect.Interface {
		if rv.IsNil() {
			// Write type information for nil pointers to distinguish nil *int from nil *string
			if kind == reflect.Ptr {
				hasher.Write([]byte(rv.Type().String()))
			}
			return nil
		}

		// Skip if we already visited that node
		ptr := rv.Pointer()
		if visited[ptr] {
			return nil
		}
		visited[ptr] = true
	}

	// [reflect.Kind] is actually a uint, therefore we can use it directly
	var vt [1]byte
	var rt reflect.Type

	// If input is a pointer, then dereference until we get an actual value
	pDepth := 0
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

	// Process value based on it's type
	switch kind {

	// If we received a reflect.Invalid, then we're getting an empty pointer
	case reflect.Invalid:
		// We cannot infer type of nil, however, we need to treat nil and *nil as same values
		if rt != nil && rt.Kind() != reflect.Interface {
			hasher.Write([]byte(rt.String()))
		}

		return nil

	// Unsigned integers
	case reflect.Uint, reflect.Uintptr, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		hasher.Write(vt[:1])
		if err := binary.Write(hasher, binary.LittleEndian, rv.Uint()); err != nil {
			return fmt.Errorf("writer: %w", err)
		}
		return nil

		// Signed integers
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		hasher.Write(vt[:1])
		if err := binary.Write(hasher, binary.LittleEndian, rv.Int()); err != nil {
			return fmt.Errorf("writer: %w", err)
		}
		return nil

	// Floats
	case reflect.Float32, reflect.Float64:
		hasher.Write(vt[:1])
		if err := binary.Write(hasher, binary.LittleEndian, rv.Float()); err != nil {
			return fmt.Errorf("writer: %w", err)
		}
		return nil

	// Complex
	case reflect.Complex64, reflect.Complex128:
		hasher.Write(vt[:1])
		if err := binary.Write(hasher, binary.LittleEndian, rv.Complex()); err != nil {
			return fmt.Errorf("writer: %w", err)
		}
		return nil

	// Boolean
	case reflect.Bool:
		hasher.Write(vt[:1])
		if err := binary.Write(hasher, binary.LittleEndian, rv.Bool()); err != nil {
			return fmt.Errorf("writer: %w", err)
		}
		return nil

	// String
	case reflect.String:
		hasher.Write(vt[:1])
		if err := binary.Write(hasher, binary.LittleEndian, []byte(rv.String())); err != nil {
			return fmt.Errorf("writer: %w", err)
		}
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
			if err := walkObject(rv.Index(i).Interface(), hasher, depth+1, visited); err != nil {
				return fmt.Errorf("slice: %w", err)
			}
		}

		return nil

	// Maps
	//
	// NOTE: map keys may be produced in a random order,
	// something like `map[interface{}]interface{}{"foo": "bar", "bar": 0}` is especially complicated, because it's hard to get the key
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

		// Sort
		sort.Slice(keys, func(i, j int) bool {
			return compareMapKeys(keys[i], keys[j]) < 0
		})

		for _, key := range keys {
			if err := walkObject(key.Interface(), hasher, depth+1, visited); err != nil {
				return fmt.Errorf("map value: %w", err)
			}
			value := rv.MapIndex(key)

			if err := walkObject(value.Interface(), hasher, depth+1, visited); err != nil {
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

			if err := walkObject(field.Interface(), hasher, depth+1, visited); err != nil {
				return fmt.Errorf("struct: %w", err)
			}
		}

		return nil
	}

	return nil
}

// From accepts any golang value including pointers and recursively converts
// then to a unique hash value.
func From(input any, hasher hash.Hash) (Hash, error) {
	visited := make(map[uintptr]bool)

	if err := walkObject(input, hasher, 0, visited); err != nil {
		return nil, fmt.Errorf("from: %w", err)
	}

	hash := hasher.Sum(nil)
	return hash, nil
}
