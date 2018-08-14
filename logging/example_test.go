package logging

func ExampleFieldsBuilder_FlattenMapInterface() {
	logger := New("Test_FlattenMapInterface")

	fieldBuilder := &FieldsBuilder{}
	fieldBuilder.FlattenMapInterface(map[string]interface{}{"key1": true})

	logger.Info("Output:", fieldBuilder.Fields()...)
}

func ExampleFieldsBuilder_FlattenMapString() {
	logger := New("Test_FlattenMapString")

	fieldBuilder := &FieldsBuilder{}
	fieldBuilder.FlattenMapString(map[string]string{"key1": "1", "key2": "false"})

	logger.Info("Output:", fieldBuilder.Fields()...)
}

func ExampleFieldsBuilder_AddFields() {
	logger := New("Test_AddFields")

	fieldBuilder := &FieldsBuilder{}
	fieldBuilder.AddFields("key1", 1, "key2", "2", "key3", nil)

	logger.Info("Output:", fieldBuilder.Fields()...)
}
