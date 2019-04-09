package main

import (
	"errors"
	"fmt"
	"reflect"
)

func i2s(in interface{}, out interface{}) error {
	// todo
	// fmt.Println("IN", in, "OUT", out)

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
		fmt.Println("This is a struct!", in)
		fmt.Println(inVal)

		if inVal.Kind() == reflect.Map {
			for i := 0; i < outElem.NumField(); i++ { // what to use as the 'lowest denominator', if needed at all?
				field := outElem.Type().Field(i)
				fieldName := field.Name
				fieldKeyValue := reflect.ValueOf(fieldName) // construct the 'Value' object
				fmt.Println("Getting value for", fieldKeyValue)
				fieldRef := inVal.MapIndex(fieldKeyValue)
				if fieldRef.Kind() == reflect.Invalid {
					fmt.Println("INVALID!", fieldName)
				} else {
					fieldValue := fieldRef.Elem() // entry ref
					fmt.Println(i, fieldName, fieldValue, fieldValue.Kind())
				}
			}

			fmt.Println("---")
		}

	case reflect.Slice:
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
