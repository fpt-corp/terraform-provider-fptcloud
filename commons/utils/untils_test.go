package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNameValidation_AllowsValidName(t *testing.T) {
	warns, errs := ValidateName("ValidName", "")
	assert.Empty(t, warns)
	assert.Empty(t, errs)
}

func TestNameValidation_RejectsNameWithWhitespace(t *testing.T) {
	warns, errs := ValidateName("Invalid Name", "")
	assert.Empty(t, warns)
	assert.NotEmpty(t, errs)
	assert.Equal(t, "name cannot contain whitespace. Got Invalid Name", errs[0].Error())
}

func TestNameValidation_RejectsNonStringInput(t *testing.T) {
	warns, errs := ValidateName(123, "")
	assert.Empty(t, warns)
	assert.NotEmpty(t, errs)
	assert.Equal(t, "expected name to be string", errs[0].Error())
}

func TestToQueryParams_ConvertsStructToQueryParams(t *testing.T) {
	type TestStruct struct {
		Name  string  `json:"name"`
		Age   int     `json:"age"`
		Score float64 `json:"score"`
	}
	data := TestStruct{Name: "John", Age: 30, Score: 95.5}
	expected := "?age=30&name=John&score=95.5"
	result := ToQueryParams(data)
	assert.Equal(t, expected, result)
}

func TestToQueryParams_HandlesEmptyStruct(t *testing.T) {
	type EmptyStruct struct{}
	data := EmptyStruct{}
	expected := "?"
	result := ToQueryParams(data)
	assert.Equal(t, expected, result)
}

func TestToQueryParams_HandlesNilPointer(t *testing.T) {
	type TestStruct struct {
		Name *string `json:"name"`
	}
	var name *string
	data := TestStruct{Name: name}
	expected := "?"
	result := ToQueryParams(data)
	assert.Equal(t, expected, result)
}

func TestToQueryParams_HandlesPointerFields(t *testing.T) {
	type TestStruct struct {
		Name *string `json:"name"`
	}
	name := "John"
	data := TestStruct{Name: &name}
	expected := "?name=John"
	result := ToQueryParams(data)
	assert.Equal(t, expected, result)
}

func TestCommaSeparatedAllowedKeys_ReturnsSortedKeys(t *testing.T) {
	keys := []string{"key3", "key1", "key2"}
	expected := "`key1`, `key2`, `key3`"
	result := GetCommaSeparatedAllowedKeys(keys)
	assert.Equal(t, expected, result)
}

func TestCommaSeparatedAllowedKeys_HandlesEmptyKeys(t *testing.T) {
	var keys []string
	expected := ""
	result := GetCommaSeparatedAllowedKeys(keys)
	assert.Equal(t, expected, result)
}
