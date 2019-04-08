package main

import (
	"errors"
	_ "fmt"
	"reflect"
)

type Simple2 struct {
	ID       int
	Username string
	Active   bool
}

func i2s(in interface{}, out interface{}) error {
	// todo
	// fmt.Println("IN", in, "OUT", out)

	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr {	// ?
		// the input structure MUST be a pointer, PANIC otherwise
		return errors.New("The input is not a pointer")
	}

	// ---

	elem := v.Elem()	// pointer inner

	switch elem.Kind() {
	case reflect.Slice:
		// we are given a slice (array)
		// slice of what?

		T := elem.Type().Elem()	// array -> single elem type
		newRecord := reflect.New(T).Elem()	// create new of type of the slice

		idField := newRecord.FieldByName("ID")
		idField.SetInt(123)

		usernameField := newRecord.FieldByName("Username")
		usernameField.SetString("Test")

		activeField := newRecord.FieldByName("Active")
		activeField.SetBool(false)

		elem.Set(reflect.Append(elem, newRecord))	// add elem to struct
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