package opentracing

// A TagsBuilder is a variable-sized array of interface{} with FlattenMapInterface, FlattenMapString,
// AddTags and Tags methods. The zero value for TagsBuilder is an empty tags ready to use.
type TagsBuilder struct {
	tags []interface{} // store tags to be returned by Tags()
}

// FlattenMapInterface adds the key-value pairs from map m to the builder.
// No checks are done for duplicate tags.
func (b *TagsBuilder) FlattenMapInterface(m map[string]interface{}) {
	for k, v := range m {
		b.tags = append(b.tags, k, v)
	}
}

// FlattenMapString adds the key-value pairs from map m to the builder.
// No checks are done for duplicate tags.
func (b *TagsBuilder) FlattenMapString(m map[string]string) {
	for k, v := range m {
		b.tags = append(b.tags, k, v)
	}
}

// AddTags adds the provided tags to the builder.
// No checks are done for duplicate keys or partial key-value pairs.
func (b *TagsBuilder) AddTags(val ...interface{}) {
	b.tags = append(b.tags, val...)
}

// Tags returns an array of interface from TagsBuilder.
// Method does not remove duplicate key.
func (b TagsBuilder) Tags() []interface{} {
	return b.tags
}
