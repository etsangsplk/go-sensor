package logging

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlattenMap(t *testing.T) {
	type mapStringInterfacePair struct {
		m map[string]interface{}
		f []interface{}
	}
	// Lots of checking and type conversionn into zap Fields already taken care of by
	// ubers' zap,  so just do spot checking.
	anonymousFunc := func() {}
	var nilUrlPointer *url.URL
	testCases := []mapStringInterfacePair{
		{map[string]interface{}{"integer": 1}, []interface{}{"integer", 1}},
		{map[string]interface{}{"boolean": false}, []interface{}{"boolean", false}},
		{map[string]interface{}{"float": 1.0}, []interface{}{"float", 1.0}},
		{map[string]interface{}{"nil": nil}, []interface{}{"nil", nil}},
		{map[string]interface{}{"string": "string"}, []interface{}{"string", "string"}},
		{map[string]interface{}{"": 1}, []interface{}{"", 1}},
		{map[string]interface{}{"urlNilPointer": nilUrlPointer}, []interface{}{"urlNilPointer", nilUrlPointer}},
		{map[string]interface{}{"anonymousfunc": &anonymousFunc}, []interface{}{"anonymousfunc", &anonymousFunc}},
	}
	for i, v := range testCases {
		fieldBuilder := &FieldsBuilder{}
		fieldBuilder.FlattenMapInterface(v.m)
		actual := fieldBuilder.Fields()

		assert.ElementsMatchf(t, v.f, actual, fmt.Sprintf("test case: %v failed", i))
	}

	// Show that fields appending
	fieldBuilder := &FieldsBuilder{}
	testCases = []mapStringInterfacePair{
		{map[string]interface{}{"integer": 1}, []interface{}{"integer", 1}},
		{map[string]interface{}{"boolean": false}, []interface{}{"integer", 1, "boolean", false}},
	}
	for i, v := range testCases {
		fieldBuilder.FlattenMapInterface(v.m)
		actual := fieldBuilder.Fields()
		assert.ElementsMatchf(t, v.f, actual, fmt.Sprintf("test case: %v failed", i))
	}
}

func TestFlattenMapStringString(t *testing.T) {
	type mapStringStringPair struct {
		m map[string]string
		f []interface{}
	}
	testCases := []mapStringStringPair{
		{map[string]string{"integer": "1"}, []interface{}{"integer", "1"}},
		{map[string]string{"string": "string"}, []interface{}{"string", "string"}},
		{map[string]string{"": "1"}, []interface{}{"", "1"}},
	}
	for i, v := range testCases {
		fieldBuilder := &FieldsBuilder{}
		fieldBuilder.FlattenMapString(v.m)
		actual := fieldBuilder.Fields()
		assert.ElementsMatchf(t, v.f, actual, fmt.Sprintf("test case: %v failed", i))
	}

	fieldBuilder := &FieldsBuilder{}
	testCases = []mapStringStringPair{
		{map[string]string{"integer": "1"}, []interface{}{"integer", "1"}},
		{map[string]string{"boolean": "false"}, []interface{}{"integer", "1", "boolean", "false"}},
	}
	for i, v := range testCases {
		fieldBuilder.FlattenMapString(v.m)
		actual := fieldBuilder.Fields()
		assert.ElementsMatchf(t, v.f, actual, fmt.Sprintf("test case: %v failed", i))
	}
}

func TestAddFields(t *testing.T) {
	type arrayInterfacePair struct {
		m []interface{}
		f []interface{}
	}
	// Lots of checking and type conversionn into zap Fields already taken care of by
	// ubers' zap,  so just do spot checking.
	// Wrong number of arguments is also taken care of by uber's zap library.
	fieldBuilder := &FieldsBuilder{}
	fieldBuilder.AddFields("a")
	fieldBuilder.AddFields("a", 1, "b", "b", "e", nil)
	assert.ElementsMatch(t, fieldBuilder.Fields(), []interface{}{"a", "a", 1, "b", "b", "e", nil})
	assert.NotNil(t, func() { fieldBuilder.AddFields() }, "Should not be nil")

	testCases := []arrayInterfacePair{
		{[]interface{}{"a", 1.0}, []interface{}{"a", 1.0}},
		{[]interface{}{"a", 1, "e", "no empty string allowed"}, []interface{}{"a", 1, "e", "no empty string allowed"}},
	}

	for i, v := range testCases {
		fieldBuilder := &FieldsBuilder{}
		fieldBuilder.AddFields(v.m...)
		actual := fieldBuilder.Fields()
		assert.ElementsMatchf(t, v.f, actual, fmt.Sprintf("test case: %v failed", i))
	}
}
