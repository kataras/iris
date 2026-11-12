package encoding

type (
	// StringEntry is a key-value pair with string key and string value.
	StringEntry = Entry[string]
	// StringEntries is a slice of key-value pairs with string key and string value.
	StringEntries = Entries[string]
)
