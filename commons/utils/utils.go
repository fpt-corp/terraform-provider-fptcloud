package utils

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func ValidateName(v interface{}, _ string) (ws []string, es []error) {
	var errs []error
	var warns []string
	value, ok := v.(string)
	if !ok {
		errs = append(errs, fmt.Errorf("expected name to be string"))
		return warns, errs
	}
	whiteSpace := regexp.MustCompile(`\s+`)
	if whiteSpace.Match([]byte(value)) {
		errs = append(errs, fmt.Errorf("name cannot contain whitespace. Got %s", value))
		return warns, errs
	}
	return warns, errs
}

// ToQueryParams converts a struct to URL query parameters.
func ToQueryParams(data interface{}) string {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	params := url.Values{}
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		tag := fieldType.Tag.Get("json")
		if tag == "" || tag == "-" {
			tag = fieldType.Name
		}
		tag = strings.Split(tag, ",")[0]

		if value := fieldValueToString(field); value != "" {
			params.Add(tag, value)
		}
	}

	return "?" + params.Encode()
}

// fieldValueToString converts a reflection value to a string representation.
func fieldValueToString(field reflect.Value) string {
	switch field.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Ptr:
		if !field.IsNil() {
			return fieldValueToString(field.Elem())
		}
		return ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(field.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(field.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(field.Float(), 'f', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(field.Bool())
	default:
		return ""
	}
}

// GetCommaSeparatedAllowedKeys is used by "tfplugindocs" CLI to generate Markdown docs
func GetCommaSeparatedAllowedKeys(allowedKeys []string) string {
	var res []string
	for _, ak := range allowedKeys {
		res = append(res, fmt.Sprintf("`%s`", ak))
	}
	sort.Strings(res)
	return strings.Join(res, ", ")
}
