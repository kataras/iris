package httprouter

func joinPathArguments(args ...interface{}) []interface{} {
	arguments := args[0:]
	for i, v := range arguments {
		if arr, ok := v.([]string); ok {
			if len(arr) > 0 {
				interfaceArr := make([]interface{}, len(arr))
				for j, sv := range arr {
					interfaceArr[j] = sv
				}
				// replace the current slice
				// with the first string element (always as interface{})
				arguments[i] = interfaceArr[0]
				// append the rest of them to the slice itself
				// the range is not affected by these things in go,
				// so we are safe to do it.
				arguments = append(args, interfaceArr[1:]...)
			}
		}
	}
	return arguments
}
