package main

import (
	"errors"
	"reflect"
)

func i2s(in interface{}, out interface{}) error {
	Vout := reflect.ValueOf(out)
	if Vout.Kind() != reflect.Ptr { // ?
		// the input structure MUST be a pointer, PANIC otherwise
		return errors.New("the input is not a pointer")
	}

	// ---

	Tin := reflect.TypeOf(in)
	Vin := reflect.ValueOf(in) // true value behind the `interface`
	Kin := Tin.Kind() // higher ~type?

	V := Vout.Elem()	// -> reflect.Value (pointer de-reference)
	Kout := V.Kind()	// Kind of output elem (that it points to)

	switch Kout {
	case reflect.Struct:
		// we are given a structure
		if Kin != reflect.Map {
			return errors.New("struct out expects a map in")
		}

		for i := 0; i < V.NumField(); i++ {
			field := V.Type().Field(i)
			fieldName := field.Name	// -> string

			mapRef := Vin.MapIndex(reflect.ValueOf(fieldName))
			mapValue := mapRef.Elem() // entry ref
			mapValueType := mapValue.Type().String()

			f := V.FieldByName(fieldName)	// field object

			// fmt.Println(i, fieldName, mapValue, mapValue.Kind())
			// type correction. e.g float64 -> int

			switch f.Kind() {
			case reflect.Int:
				if mapValueType != "float64" {
					return errors.New("incompatible types")
				}

				v := int(mapValue.Float())	// float64 -> int trick
				// alternatively, we could use 'SetInt'
				f.Set(reflect.ValueOf(v))
			case reflect.String:
				if mapValueType != "string" {
					return errors.New("incompatible types")
				}

				v := mapValue.String()
				// alternatively, we could use 'SetString'
				f.Set(reflect.ValueOf(v))
			case reflect.Bool:
				if mapValueType != "bool" {
					return errors.New("incompatible types")
				}
				v := mapValue.Bool()

				// alternatively, we could use 'SetBool'
				f.Set(reflect.ValueOf(v))

			default:
				// recursion?
				recIn := mapValue.Interface()
				recOut := f.Addr().Interface()	// '.Addr' -> pointer (ref-type anyway?)

				z := i2s(recIn, recOut)
				if z != nil {
					return z
				}
			}
		}

	// special
	case reflect.Slice:
		// ~array
		if Tin.Kind() != reflect.Slice {
			return errors.New("slice expected")
		}

		for i := 0; i < Vin.Len(); i++ {
			elementType := V.Type().Elem()	// single element type
			// 'reflect.Zero' should NOT be used as not addressable/settable
			newElement := reflect.New(elementType)	// returns a pointer (addressable/settable)

			// fmt.Println("NEW ELEMENT: ", newElement)
			err := i2s(Vin.Index(i).Interface(), newElement.Interface())
			if err != nil {
				return err
			}

			// and, finally, append back to the original array
			V.Set(reflect.Append(V, newElement.Elem()))
		}
	}

	return nil	// no errors have occurred
}

func main() {
	// not used
}
