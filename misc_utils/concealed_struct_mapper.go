package misc_utils

import (
	"fmt"
	"reflect"
)

func TurnStruct2Map(s any) map[string]any {
	sval := reflect.ValueOf(s)
	styp := reflect.TypeOf(s)
	switch styp.Kind() {
	case reflect.Ptr:
		for styp.Kind() == reflect.Ptr {
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
