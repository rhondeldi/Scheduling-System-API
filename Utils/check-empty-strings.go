package Utils

import (
	"fmt"
	"reflect"
)

func CheckForEmptyStrings(data interface{}, currentPath string) []string {
	var paths []string
	v := reflect.ValueOf(data)

	// handle pointers by dereferencing them
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)

			// skip unexported fields
			if field.PkgPath != "" {
				continue
			}

			// build path using field name
			fieldPath := fmt.Sprintf("%s.%s", currentPath, field.Name)
			paths = append(paths, CheckForEmptyStrings(v.Field(i).Interface(), fieldPath)...)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			// include index in the path
			indexPath := fmt.Sprintf("%s[%d]", currentPath, i)
			paths = append(paths, CheckForEmptyStrings(v.Index(i).Interface(), indexPath)...)
		}
	case reflect.String:
		if v.String() == "" {
			// record the path if an empty string is found
			paths = append(paths, currentPath)
		}
	}
	return paths
}
