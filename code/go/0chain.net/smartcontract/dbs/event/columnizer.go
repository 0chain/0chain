package event

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

var (
	firstCapRegex 	= regexp.MustCompile("(.)([A-Z][a-z]+)")
	allCapRegex 	= regexp.MustCompile("([a-z0-9])([A-Z])")
	gormColumRegex 	= regexp.MustCompile("(?m)column:([a-zA-Z0-9_]+);")
)


func toSnakeCase(str string) string {
    snake := firstCapRegex.ReplaceAllString(str, "${1}_${2}")
    snake = allCapRegex.ReplaceAllString(snake, "${1}_${2}")
    return strings.ToLower(snake)
}

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
			
			// Resolve field key in this order: gorm.column Tag > snake-casing FieldName
			var fkey string
			if matches := gormColumRegex.FindStringSubmatch(f.Tag.Get("gorm")); len(matches) > 1 {
				fkey = matches[1]
			}
			if fkey == "" {
				fkey = toSnakeCase(f.Name)
			}

			if _, ok := columns[fkey]; !ok {
				columns[fkey] = make([]interface{}, 0, len(objects))
			}
			fvalue := v.Field(fidx).Interface()
			columns[fkey] = append(columns[fkey], fvalue)
		}
		
	}

	return columns, nil
}
