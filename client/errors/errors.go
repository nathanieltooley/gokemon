package errors

// Returns the value passed in if there is no error, otherwise it will panic
// Very similar to Rust's Unwrap method on an Option or Err enum
func Must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}

	return value
}
