package checks

import "reflect"

func IsNilInterface(i any) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	kind := v.Kind()
	if kind == reflect.Ptr || kind == reflect.Slice || kind == reflect.Map || kind == reflect.Chan || kind == reflect.Func || kind == reflect.Interface {
		return v.IsNil()
	}
	return false
}
