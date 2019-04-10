package main

import (
	"errors"
	"fmt"
	"reflect"
)

type I interface{} // alias

func extractValue(v reflect.Value) I {
	fmt.Println("extractValue", v, v.Kind())
	// ---
	switch v.Kind() {
	case reflect.Float64: // ?; and not sure about 'float'
		return int64(v.Float())
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return v.Bool()
	}

	return nil
}

func i2s(in interface{}, out interface{}) error {
	// todo
	fmt.Println("IN:", in, "| OUT:", out)

	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr { // ?
		// the input structure MUST be a pointer, PANIC otherwise
		return errors.New("The input is not a pointer")
	}

	// ---

	inType := reflect.TypeOf(in).Kind()
	inVal := reflect.ValueOf(in)

	// ---

	outElem := v.Elem() // pointer inner

	// fmt.Println(inType, inVal, outElem, outElem.Kind())

	switch outElem.Kind() {
	case reflect.Struct:
		fmt.Println("Input->Struct!")
		fmt.Println(inVal, inVal.Kind(), inType)

		if inVal.Kind() == reflect.Map {
			for i := 0; i < outElem.NumField(); i++ { // what to use as the 'lowest denominator', if needed at all?
				field := outElem.Type().Field(i) // 'description' of the field
				fieldName := field.Name
				fieldKeyValue := reflect.ValueOf(fieldName) // construct the string key 'Value' object
				input := inVal.MapIndex(fieldKeyValue)      // get the value

				// RECURSION CALL
				fmt.Println("CALL")
				output := outElem.Field(i)
				callInput := map[I]I{fieldName: input.Elem()} // single elem
				if err := i2s(callInput, &output); err != nil {
					return err
				}
			}

			fmt.Println("---")
		} else {
			fmt.Println("WRONG", inVal.Kind())
		}
	case reflect.Slice:
		fmt.Println("SLICE IT IS")

		// we are given a slice (array)
		// slice of what?
		T := outElem.Type().Elem() // array -> single elem type

		// if the 'out' = slice, then the 'in' must also be slice (to match)
		if inType != reflect.Slice {
			outElem.Set(reflect.Zero(outElem.Type())) // empty slice?
			return errors.New("mismatching input and ouput types")
		}

		// inval = slice => .Len() is available
		for i := 0; i < inVal.Len(); i++ { // correct way to iterate
			sliceElem := inVal.Index(i).Elem() // entry
			if sliceElem.Kind() != reflect.Map {
				panic("NO!") // should not happen, but still
			}

			newRecord := reflect.New(T).Elem() // create new of type of the slice

			// change/populate the map 'record' keys
			for _, mapKey := range sliceElem.MapKeys() {
				keyName := mapKey.String()
				field := newRecord.FieldByName(keyName) // get the ref in the 'out' structure
				switch field.Kind() {
				case reflect.Int: // ?; and not sure about 'float'
					innerValue := sliceElem.MapIndex(mapKey).Elem()
					field.SetInt(int64(innerValue.Float())) // float -> int trick
				case reflect.String:
					innerValue := sliceElem.MapIndex(mapKey).Elem()
					field.SetString(innerValue.String())
				case reflect.Bool:
					innerValue := sliceElem.MapIndex(mapKey).Elem()
					field.SetBool(innerValue.Bool())
				default:
					// not sure about this part, probably recursion
					fmt.Println("default")
				}
			}

			// add the new element/entry/record to the output slice
			outElem.Set(reflect.Append(outElem, newRecord)) // add elem to struct
		}

	default:
		// recursion base case?
		fmt.Println("SOMETHING")
	}

	// ---

	// if elem.Kind() == reflect.Struct {
	// 	fmt.Println("STRUCT IT IS")
	// 	for i := 0; i < elem.NumField(); i++ {
	// 		f := elem.Field(i)
	// 		fmt.Printf("%d: %s %s = %v\n", i, typeOfT.Field(i).Name, f.Type(), f.Interface())
	// 	}
	// }

	return nil
}

func main() {
	// not used
}
