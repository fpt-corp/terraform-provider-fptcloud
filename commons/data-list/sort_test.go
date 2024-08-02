package data_list

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestExpandSorts_ValidInput(t *testing.T) {
	rawSorts := []interface{}{
		map[string]interface{}{
			"key":       "name",
			"direction": "asc",
		},
		map[string]interface{}{
			"key":       "age",
			"direction": "desc",
		},
	}

	expected := []commonSort{
		{key: "name", direction: "asc"},
		{key: "age", direction: "desc"},
	}

	result := expandSorts(rawSorts)
	assert.Equal(t, expected, result)
}

func TestExpandSorts_EmptyInput(t *testing.T) {
	rawSorts := []interface{}{}

	expected := []commonSort{}

	result := expandSorts(rawSorts)
	assert.Equal(t, expected, result)
}

func TestApplySorts_SingleSortAsc(t *testing.T) {
	recordSchema := map[string]*schema.Schema{
		"name": {Type: schema.TypeString},
	}

	records := []map[string]interface{}{
		{"name": "John"},
		{"name": "Jane"},
	}

	sorts := []commonSort{
		{key: "name", direction: "asc"},
	}

	expected := []map[string]interface{}{
		{"name": "Jane"},
		{"name": "John"},
	}

	result := applySorts(recordSchema, records, sorts)
	assert.Equal(t, expected, result)
}

func TestApplySorts_SingleSortDesc(t *testing.T) {
	recordSchema := map[string]*schema.Schema{
		"name": {Type: schema.TypeString},
	}

	records := []map[string]interface{}{
		{"name": "John"},
		{"name": "Jane"},
	}

	sorts := []commonSort{
		{key: "name", direction: "desc"},
	}

	expected := []map[string]interface{}{
		{"name": "John"},
		{"name": "Jane"},
	}

	result := applySorts(recordSchema, records, sorts)
	assert.Equal(t, expected, result)
}

func TestApplySorts_MultipleSorts(t *testing.T) {
	recordSchema := map[string]*schema.Schema{
		"name": {Type: schema.TypeString},
		"age":  {Type: schema.TypeInt},
	}

	records := []map[string]interface{}{
		{"name": "John", "age": 30},
		{"name": "Jane", "age": 25},
		{"name": "John", "age": 25},
	}

	sorts := []commonSort{
		{key: "name", direction: "asc"},
		{key: "age", direction: "desc"},
	}

	expected := []map[string]interface{}{
		{"name": "Jane", "age": 25},
		{"name": "John", "age": 30},
		{"name": "John", "age": 25},
	}

	result := applySorts(recordSchema, records, sorts)
	assert.Equal(t, expected, result)
}

func TestApplySorts_EmptyRecords(t *testing.T) {
	recordSchema := map[string]*schema.Schema{
		"name": {Type: schema.TypeString},
	}

	records := []map[string]interface{}{}

	sorts := []commonSort{
		{key: "name", direction: "asc"},
	}

	expected := []map[string]interface{}{}

	result := applySorts(recordSchema, records, sorts)
	assert.Equal(t, expected, result)
}
