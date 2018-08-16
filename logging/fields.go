package logging

// A FieldsBuilder is a variable-sized array of interface{} with FlattenMapInterface, FlattenMapString,
// AddFields and Fields methods. The zero value for FieldsBuilder is an empty fields ready to use.
type FieldsBuilder struct {
	fields []interface{} // store fields to be returned by Fields()
}

// FlattenMapInterface adds the key-value pairs from map m to the builder.
// No checks are done for duplicate fields.
func (b *FieldsBuilder) FlattenMapInterface(m map[string]interface{}) {
	for k, v := range m {
		b.fields = append(b.fields, k, v)
	}
}

// FlattenMapString adds the key-value pairs from map m to the builder.
// No checks are done for duplicate fields.
func (b *FieldsBuilder) FlattenMapString(m map[string]string) {
	for k, v := range m {
		b.fields = append(b.fields, k, v)
	}
}

// AddFields adds the provided fields to the builder.
// No checks are done for duplicate keys or partial key-value pairs.
func (b *FieldsBuilder) AddFields(val ...interface{}) {
	b.fields = append(b.fields, val...)
}

// Fields returns an array of interface from FieldsBuilder.
// Method does not remove duplicate key.
func (b FieldsBuilder) Fields() []interface{} {
	return b.fields
}
