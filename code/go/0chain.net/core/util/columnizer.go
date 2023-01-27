package util

import (
	"fmt"
	"reflect"
)

func Columnize[T any](objects []T) (map[string][]interface{}, error) {
	columns := make(map[string][]interface{})
	
	for _, obj := range objects {
		v := reflect.ValueOf(obj)

		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v.Kind() != reflect.Struct {
			return nil, fmt.Errorf("columnize error: type invalid")
		}

		t := v.Type()

		for fidx := 0; fidx < t.NumField(); fidx++ {
			f := t.Field(fidx)
			fkey := f.Tag.Get("json")
			if fkey == "" {
				return nil, fmt.Errorf("columnize error: No json tag for field %v::%v", t.Name(), f.Name)
			}

			if _, ok := columns[fkey]; !ok {
				columns[fkey] = make([]interface{}, 0, len(objects))
			}
			fvalue := v.Field(fidx).Interface()
			fmt.Printf("%v => %v\n", fkey, fvalue)
			columns[fkey] = append(columns[fkey], fvalue)
		}
		
	}

	return columns, nil
}
