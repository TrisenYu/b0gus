package misc_utils

// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD

import (
	"fmt"
	"reflect"
)

// TurnStruct2Map will use recursion to convert a struct into (nested) map.
//
//	That is <map: string -> {map | string}>.
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
		return nil
	}

	var res = make(map[string]any)

	// TODO: use a queue instead of recursion
	for i := range styp.NumField() {
		child := styp.Field(i).Type
		if len(child.Name()) == 0 {
			// for concealing type, just skip.
			continue
		}
		if sval.Field(i).CanInterface() {
			if child.Kind() == reflect.Struct {
				res[styp.Field(i).Name] = TurnStruct2Map(sval.Field(i).Interface())
				continue
			}
			res[styp.Field(i).Name] = sval.Field(i).Interface()
		}
	}
	return res
}

func GetTypeNameViaType(v any) string {
	t := reflect.TypeOf(v)
	switch t.Kind() {
	case reflect.Pointer:
		return t.String()
	case reflect.Slice:
		return t.String()
	case reflect.Func:
		return t.String()
	case reflect.Map:
		return t.String()
	case reflect.Struct:
		if t.Name() == "" {
			return "concealed-struct"
		}
		return t.Name()
	default:
		return t.Name()
	}
}

func GetFieldValueByName(obj any, fieldName string) (any, error) {
	v := reflect.ValueOf(obj)
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return v.Interface(), nil
	}
	field := v.FieldByName(fieldName)

	if !field.IsValid() {
		return struct{}{}, fmt.Errorf(
			"field '%s' not found in struct %T",
			fieldName, obj,
		)
	}
	if !field.CanAddr() {
		return field.Interface(), nil
	}
	return field.Addr().Interface(), nil
}
