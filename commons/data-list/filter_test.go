package data_list

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestExpandFilters_ValidFilters(t *testing.T) {
	recordSchema := map[string]*schema.Schema{
		"name": {
			Type: schema.TypeString,
		},
		"age": {
			Type: schema.TypeInt,
		},
	}

	rawFilters := []interface{}{
		map[string]interface{}{
			"key":      "name",
			"values":   []interface{}{"John"},
			"all":      false,
			"match_by": "exact",
		},
		map[string]interface{}{
			"key":      "age",
			"values":   []interface{}{"30"},
			"all":      true,
			"match_by": "exact",
		},
	}

	expectedFilters := []commonFilter{
		{
			key:     "name",
			values:  []interface{}{"John"},
			all:     false,
			matchBy: "exact",
		},
		{
			key:     "age",
			values:  []interface{}{30},
			all:     true,
			matchBy: "exact",
		},
	}

	filters, err := expandFilters(recordSchema, rawFilters)
	assert.NoError(t, err)
	assert.Equal(t, expectedFilters, filters)
}

func TestExpandFilters_InvalidKey(t *testing.T) {
	recordSchema := map[string]*schema.Schema{
		"name": {
			Type: schema.TypeString,
		},
	}

	rawFilters := []interface{}{
		map[string]interface{}{
			"key":      "invalid_key",
			"values":   []interface{}{"John"},
			"all":      false,
			"match_by": "exact",
		},
	}

	_, err := expandFilters(recordSchema, rawFilters)
	assert.Error(t, err)
	assert.Equal(t, "field 'invalid_key' does not exist in record schema", err.Error())
}

func TestExpandFilters_InvalidValueType(t *testing.T) {
	recordSchema := map[string]*schema.Schema{
		"age": {
			Type: schema.TypeInt,
		},
	}

	rawFilters := []interface{}{
		map[string]interface{}{
			"key":      "age",
			"values":   []interface{}{"invalid_int"},
			"all":      false,
			"match_by": "exact",
		},
	}

	_, err := expandFilters(recordSchema, rawFilters)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse value as integer")
}

func TestExpandFilters_EmptyFilters(t *testing.T) {
	recordSchema := map[string]*schema.Schema{
		"name": {
			Type: schema.TypeString,
		},
	}

	var rawFilters []interface{}

	filters, err := expandFilters(recordSchema, rawFilters)
	assert.NoError(t, err)
	assert.Empty(t, filters)
}

func TestExpandPrimitiveFilterValue_StringExact(t *testing.T) {
	value, err := expandPrimitiveFilterValue("test", schema.TypeString, "exact")
	assert.NoError(t, err)
	assert.Equal(t, "test", value)
}

func TestExpandPrimitiveFilterValue_StringSubstring(t *testing.T) {
	value, err := expandPrimitiveFilterValue("test", schema.TypeString, "substring")
	assert.NoError(t, err)
	assert.Equal(t, "test", value)
}

func TestExpandPrimitiveFilterValue_StringRegex(t *testing.T) {
	value, err := expandPrimitiveFilterValue("test.*", schema.TypeString, "re")
	assert.NoError(t, err)
	assert.IsType(t, &regexp.Regexp{}, value)
	assert.True(t, value.(*regexp.Regexp).MatchString("test123"))
}

func TestExpandPrimitiveFilterValue_StringRegexInvalid(t *testing.T) {
	_, err := expandPrimitiveFilterValue("[invalid", schema.TypeString, "re")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse value as regular expression")
}

func TestExpandPrimitiveFilterValue_Bool(t *testing.T) {
	value, err := expandPrimitiveFilterValue("true", schema.TypeBool, "exact")
	assert.NoError(t, err)
	assert.Equal(t, true, value)
}

func TestExpandPrimitiveFilterValue_BoolInvalid(t *testing.T) {
	_, err := expandPrimitiveFilterValue("notabool", schema.TypeBool, "exact")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse value as bool")
}

func TestExpandPrimitiveFilterValue_Int(t *testing.T) {
	value, err := expandPrimitiveFilterValue("123", schema.TypeInt, "exact")
	assert.NoError(t, err)
	assert.Equal(t, 123, value)
}

func TestExpandPrimitiveFilterValue_IntInvalid(t *testing.T) {
	_, err := expandPrimitiveFilterValue("notanint", schema.TypeInt, "exact")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse value as integer")
}

func TestExpandPrimitiveFilterValue_Float(t *testing.T) {
	value, err := expandPrimitiveFilterValue("123.45", schema.TypeFloat, "exact")
	assert.NoError(t, err)
	assert.Equal(t, 123.45, value)
}

func TestExpandPrimitiveFilterValue_FloatInvalid(t *testing.T) {
	_, err := expandPrimitiveFilterValue("notafloat", schema.TypeFloat, "exact")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse value as floating point")
}

func TestApplyFilters_StringExactMatch(t *testing.T) {
	recordSchema := map[string]*schema.Schema{
		"name": {Type: schema.TypeString},
	}

	records := []map[string]interface{}{
		{"name": "John"},
		{"name": "Jane"},
	}

	filters := []commonFilter{
		{key: "name", values: []interface{}{"John"}, all: false, matchBy: "exact"},
	}

	expected := []map[string]interface{}{
		{"name": "John"},
	}

	result := applyFilters(recordSchema, records, filters)
	assert.Equal(t, expected, result)
}

func TestApplyFilters_StringSubstringMatch(t *testing.T) {
	recordSchema := map[string]*schema.Schema{
		"name": {Type: schema.TypeString},
	}

	records := []map[string]interface{}{
		{"name": "John"},
		{"name": "Jane"},
	}

	filters := []commonFilter{
		{key: "name", values: []interface{}{"Jo"}, all: false, matchBy: "substring"},
	}

	expected := []map[string]interface{}{
		{"name": "John"},
	}

	result := applyFilters(recordSchema, records, filters)
	assert.Equal(t, expected, result)
}

func TestApplyFilters_BoolMatch(t *testing.T) {
	recordSchema := map[string]*schema.Schema{
		"active": {Type: schema.TypeBool},
	}

	records := []map[string]interface{}{
		{"active": true},
		{"active": false},
	}

	filters := []commonFilter{
		{key: "active", values: []interface{}{true}, all: false, matchBy: "exact"},
	}

	expected := []map[string]interface{}{
		{"active": true},
	}

	result := applyFilters(recordSchema, records, filters)
	assert.Equal(t, expected, result)
}

func TestApplyFilters_IntMatch(t *testing.T) {
	recordSchema := map[string]*schema.Schema{
		"age": {Type: schema.TypeInt},
	}

	records := []map[string]interface{}{
		{"age": 30},
		{"age": 25},
	}

	filters := []commonFilter{
		{key: "age", values: []interface{}{30}, all: false, matchBy: "exact"},
	}

	expected := []map[string]interface{}{
		{"age": 30},
	}

	result := applyFilters(recordSchema, records, filters)
	assert.Equal(t, expected, result)
}

func TestApplyFilters_FloatMatch(t *testing.T) {
	recordSchema := map[string]*schema.Schema{
		"score": {Type: schema.TypeFloat},
	}

	records := []map[string]interface{}{
		{"score": 95.5},
		{"score": 89.0},
	}

	filters := []commonFilter{
		{key: "score", values: []interface{}{95.5}, all: false, matchBy: "exact"},
	}

	expected := []map[string]interface{}{
		{"score": 95.5},
	}

	result := applyFilters(recordSchema, records, filters)
	assert.Equal(t, expected, result)
}

func TestApplyFilters_AllMatch(t *testing.T) {
	recordSchema := map[string]*schema.Schema{
		"tags": {Type: schema.TypeList, Elem: &schema.Schema{Type: schema.TypeString}},
	}

	records := []map[string]interface{}{
		{"tags": []interface{}{"tag1", "tag2"}},
		{"tags": []interface{}{"tag1"}},
	}

	filters := []commonFilter{
		{key: "tags", values: []interface{}{"tag1", "tag2"}, all: true, matchBy: "exact"},
	}

	expected := []map[string]interface{}{
		{"tags": []interface{}{"tag1", "tag2"}},
	}

	result := applyFilters(recordSchema, records, filters)
	assert.Equal(t, expected, result)
}
