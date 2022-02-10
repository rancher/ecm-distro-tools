package types

// IntPtr creates an int pointer value and
// returns it to the caller.
func IntPtr(i int) *int {
	v := int(i)
	return &v
}

// Int8Ptr creates an int8 pointer value and
// returns it to the caller.
func Int8Ptr(i int8) *int8 {
	v := int8(i)
	return &v
}

// Int16Ptr creates an int16 pointer value and
// returns it to the caller.
func Int16Ptr(i int16) *int16 {
	v := int16(i)
	return &v
}

// Int32Ptr creates an int32 pointer value and
// returns it to the caller.
func Int32Ptr(i int32) *int32 {
	v := int32(i)
	return &v
}

// Int64Ptr creates an int64 pointer value and
// returns it to the caller.
func Int64Ptr(i int64) *int64 {
	v := int64(i)
	return &v
}

// UintPtr creates a uint pointer value and
// returns it to the caller.
func UintPtr(i uint) *uint {
	v := uint(i)
	return &v
}

// Uint8Ptr creates a uint8 pointer value and
// returns it to the caller.
func Uint8Ptr(i uint8) *uint8 {
	v := uint8(i)
	return &v
}

// Uint16Ptr creates a uint16 pointer value and
// returns it to the caller.
func Uint16Ptr(i uint16) *uint16 {
	v := uint16(i)
	return &v
}

// Uint32Ptr creates a uint32 pointer value and
// returns it to the caller.
func Uint32Ptr(i uint32) *uint32 {
	v := uint32(i)
	return &v
}

// Uint64Ptr creates a uint64 pointer value and
// returns it to the caller.
func Uint64Ptr(i uint64) *uint64 {
	v := uint64(i)
	return &v
}

// StringPtr creates a string pointer value and
// returns it to the caller.
func StringPtr(s string) *string {
	v := string(s)
	return &v
}

// BytePtr creates a byte pointer value and
// returns it to the caller.
func BytePtr(b byte) *byte {
	v := byte(b)
	return &v
}

// Float32Ptr creates a float32 pointer value and
// returns it to the caller.
func Float32Ptr(f float32) *float32 {
	v := float32(f)
	return &v
}

// Float64Ptr creates a float64 pointer value and
// returns it to the caller.
func Float64Ptr(f float64) *float64 {
	v := float64(f)
	return &v
}

// BoolPtr creates a bool pointer value and
// returns it to the caller.
func BoolPtr(b bool) *bool {
	v := bool(b)
	return &v
}
