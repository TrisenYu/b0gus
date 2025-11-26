// SPDX-LICENSE-IDENTIFIER: 3-Clause-BSD
package misc_utils

import (
	"fmt"
	"reflect"
	// "container/list"
)

func TurnStruct2Map(s any) map[string]any {
	sval := reflect.ValueOf(s)
	styp := reflect.TypeOf(s)
	switch styp.Kind() {
	case reflect.Pointer:
		for styp.Kind() == reflect.Pointer {
			styp = styp.Elem()
			sval = sval.Elem()
		}
	case reflect.Struct:
		// do nothing and fall
	default:
		fmt.Println(styp.Kind())
		return nil
	}

	var res map[string]any = make(map[string]any)

	// TODO: use a queue instead of recursion
	for i := range styp.NumField() {
		child := styp.Field(i).Type
		if child.Kind() == reflect.Struct {
			res[styp.Field(i).Name] = TurnStruct2Map(sval.Field(i).Interface())
			continue
		}
		res[styp.Field(i).Name] = sval.Field(i).Interface()
	}
	return res
}
