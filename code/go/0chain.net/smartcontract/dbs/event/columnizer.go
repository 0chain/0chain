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
	gormForeignKeyRegex 	= regexp.MustCompile("(?m)foreignKey:[a-zA-Z0-9_]+;")
)

type FieldWithValue struct {
	Field reflect.StructField
	Value reflect.Value
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(str string) string {
    snake := firstCapRegex.ReplaceAllString(str, "${1}_${2}")
    snake = allCapRegex.ReplaceAllString(snake, "${1}_${2}")
    return strings.ToLower(snake)
}

// Columnize converts a slice of objects into a map of columns. Unwraps nested structs/pointers.
// If the object has a gorm.column tag, it will use that as the column name. Otherwise, it will snake_case the field name.
// Ignores struct/slice field that are gorm.foreignKey
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

		colStack := make([]FieldWithValue, 0, t.NumField())
		for fidx := 0; fidx < t.NumField(); fidx++ {
			colStack = append(colStack, FieldWithValue{t.Field(fidx), v.Field(fidx)})
		}

		for len(colStack) > 0 {
			fwv := colStack[len(colStack)-1]
			colStack = colStack[:len(colStack)-1]
			fmt.Printf("%v => %v\n", fwv.Field.Name, fwv.Field.Type.Kind())
			
			switch fwv.Field.Type.Kind() {
			case reflect.Ptr:
					if fwv.Value.IsNil() {
						continue
					}
					v := fwv.Value.Elem()
					for fidx := 0; fidx < fwv.Field.Type.Elem().NumField(); fidx++ {
						colStack = append(colStack, FieldWithValue{fwv.Field.Type.Elem().Field(fidx), v.Field(fidx)})
					}
			break
			case reflect.Struct:
				v := fwv.Value
				if matches := gormForeignKeyRegex.FindStringSubmatch(fwv.Field.Tag.Get("gorm")); len(matches) > 0 {
					continue
				}
				for fidx := 0; fidx < fwv.Field.Type.NumField(); fidx++ {
					colStack = append(colStack, FieldWithValue{fwv.Field.Type.Field(fidx), v.Field(fidx)})
				}
			break
			case reflect.Slice:
				if matches := gormColumRegex.FindStringSubmatch(fwv.Field.Tag.Get("gorm")); len(matches) > 0 {
					continue
				}
			default:
				// Resolve field key in this order: gorm.column Tag > snake-casing FieldName
				f := fwv.Field
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
				fvalue := fwv.Value.Interface()
				columns[fkey] = append(columns[fkey], fvalue)
			}
		}
	}

	return columns, nil
}
