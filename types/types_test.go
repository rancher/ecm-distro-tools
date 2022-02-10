package types

import (
	"reflect"
	"testing"
)

func TestIntPtr(t *testing.T) {
	type args struct {
		i int
	}
	tests := []struct {
		name      string
		args      args
		wantType  string
		wantValue int
	}{
		{
			name: "int",
			args: args{
				i: 9,
			},
			wantType:  "*int",
			wantValue: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IntPtr(tt.args.i)
			gotType := reflect.TypeOf(got).String()
			gotValue := *got
			if gotType != tt.wantType {
				t.Errorf("IntPtr() = %v, want %v", gotType, tt.wantType)
			}
			if gotValue != tt.wantValue {
				t.Errorf("IntPtr() = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
func TestInt8Ptr(t *testing.T) {
	type args struct {
		i int8
	}
	tests := []struct {
		name      string
		args      args
		wantType  string
		wantValue int8
	}{
		{
			name: "int8",
			args: args{
				i: 9,
			},
			wantType:  "*int8",
			wantValue: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Int8Ptr(tt.args.i)
			gotType := reflect.TypeOf(got).String()
			gotValue := *got
			if gotType != tt.wantType {
				t.Errorf("Int8Ptr() = %v, want %v", gotType, tt.wantType)
			}
			if gotValue != tt.wantValue {
				t.Errorf("Int8Ptr() = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
func TestInt16Ptr(t *testing.T) {
	type args struct {
		i int16
	}
	tests := []struct {
		name      string
		args      args
		wantType  string
		wantValue int16
	}{
		{
			name: "int16",
			args: args{
				i: 9,
			},
			wantType:  "*int16",
			wantValue: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Int16Ptr(tt.args.i)
			gotType := reflect.TypeOf(got).String()
			gotValue := *got
			if gotType != tt.wantType {
				t.Errorf("Int16Ptr() = %v, want %v", gotType, tt.wantType)
			}
			if gotValue != tt.wantValue {
				t.Errorf("Int16Ptr() = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
func Test32IntPtr(t *testing.T) {
	type args struct {
		i int32
	}
	tests := []struct {
		name      string
		args      args
		wantType  string
		wantValue int32
	}{
		{
			name: "int32",
			args: args{
				i: 9,
			},
			wantType:  "*int32",
			wantValue: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Int32Ptr(tt.args.i)
			gotType := reflect.TypeOf(got).String()
			gotValue := *got
			if gotType != tt.wantType {
				t.Errorf("Int32Ptr() = %v, want %v", gotType, tt.wantType)
			}
			if gotValue != tt.wantValue {
				t.Errorf("Int32Ptr() = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
func TestInt64Ptr(t *testing.T) {
	type args struct {
		i int64
	}
	tests := []struct {
		name      string
		args      args
		wantType  string
		wantValue int64
	}{
		{
			name: "int64",
			args: args{
				i: 9,
			},
			wantType:  "*int64",
			wantValue: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Int64Ptr(tt.args.i)
			gotType := reflect.TypeOf(got).String()
			gotValue := *got
			if gotType != tt.wantType {
				t.Errorf("Int64Ptr() = %v, want %v", gotType, tt.wantType)
			}
			if gotValue != tt.wantValue {
				t.Errorf("IntPtr() = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
func TestUIntPtr(t *testing.T) {
	type args struct {
		i uint
	}
	tests := []struct {
		name      string
		args      args
		wantType  string
		wantValue uint
	}{
		{
			name: "uint",
			args: args{
				i: 9,
			},
			wantType:  "*uint",
			wantValue: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UintPtr(tt.args.i)
			gotType := reflect.TypeOf(got).String()
			gotValue := *got
			if gotType != tt.wantType {
				t.Errorf("UintPtr() = %v, want %v", gotType, tt.wantType)
			}
			if gotValue != tt.wantValue {
				t.Errorf("UintPtr() = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
func TestUint8Ptr(t *testing.T) {
	type args struct {
		i uint8
	}
	tests := []struct {
		name      string
		args      args
		wantType  string
		wantValue uint8
	}{
		{
			name: "uint8",
			args: args{
				i: 9,
			},
			wantType:  "*uint8",
			wantValue: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Uint8Ptr(tt.args.i)
			gotType := reflect.TypeOf(got).String()
			gotValue := *got
			if gotType != tt.wantType {
				t.Errorf("Uint8Ptr() = %v, want %v", gotType, tt.wantType)
			}
			if gotValue != tt.wantValue {
				t.Errorf("Uint8Ptr() = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
func TestUint16Ptr(t *testing.T) {
	type args struct {
		i uint16
	}
	tests := []struct {
		name      string
		args      args
		wantType  string
		wantValue uint16
	}{
		{
			name: "uint16",
			args: args{
				i: 9,
			},
			wantType:  "*uint16",
			wantValue: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Uint16Ptr(tt.args.i)
			gotType := reflect.TypeOf(got).String()
			gotValue := *got
			if gotType != tt.wantType {
				t.Errorf("Uint16Ptr() = %v, want %v", gotType, tt.wantType)
			}
			if gotValue != tt.wantValue {
				t.Errorf("Uint16Ptr() = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
func TestUint32Ptr(t *testing.T) {
	type args struct {
		i uint32
	}
	tests := []struct {
		name      string
		args      args
		wantType  string
		wantValue uint32
	}{
		{
			name: "uint32",
			args: args{
				i: 9,
			},
			wantType:  "*uint32",
			wantValue: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Uint32Ptr(tt.args.i)
			gotType := reflect.TypeOf(got).String()
			gotValue := *got
			if gotType != tt.wantType {
				t.Errorf("Uint32Ptr() = %v, want %v", gotType, tt.wantType)
			}
			if gotValue != tt.wantValue {
				t.Errorf("Uint32Ptr() = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
func TestUint64Ptr(t *testing.T) {
	type args struct {
		i uint64
	}
	tests := []struct {
		name      string
		args      args
		wantType  string
		wantValue uint64
	}{
		{
			name: "uint64",
			args: args{
				i: 9,
			},
			wantType:  "*uint64",
			wantValue: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Uint64Ptr(tt.args.i)
			gotType := reflect.TypeOf(got).String()
			gotValue := *got
			if gotType != tt.wantType {
				t.Errorf("Uint64Ptr() = %v, want %v", gotType, tt.wantType)
			}
			if gotValue != tt.wantValue {
				t.Errorf("Uint64Ptr() = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
func TestStringPtr(t *testing.T) {
	type args struct {
		i string
	}
	tests := []struct {
		name      string
		args      args
		wantType  string
		wantValue string
	}{
		{
			name: "string",
			args: args{
				i: "Test",
			},
			wantType:  "*string",
			wantValue: "Test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringPtr(tt.args.i)
			gotType := reflect.TypeOf(got).String()
			gotValue := *got
			if gotType != tt.wantType {
				t.Errorf("StringPtr() = %v, want %v", gotType, tt.wantType)
			}
			if gotValue != tt.wantValue {
				t.Errorf("StringPtr() = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
func TestBytePtr(t *testing.T) {
	type args struct {
		i byte
	}
	tests := []struct {
		name      string
		args      args
		wantType  string
		wantValue byte
	}{
		{
			name: "byte",
			args: args{
				i: 1,
			},
			wantType:  "*uint8",
			wantValue: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BytePtr(tt.args.i)
			gotType := reflect.TypeOf(got).String()
			gotValue := *got
			if gotType != tt.wantType {
				t.Errorf("BytePtr() = %v, want %v", gotType, tt.wantType)
			}
			if gotValue != tt.wantValue {
				t.Errorf("BytePtr() = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
func TestFloat32Ptr(t *testing.T) {
	type args struct {
		i float32
	}
	tests := []struct {
		name      string
		args      args
		wantType  string
		wantValue float32
	}{
		{
			name: "float32",
			args: args{
				i: 1.2,
			},
			wantType:  "*float32",
			wantValue: 1.2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Float32Ptr(tt.args.i)
			gotType := reflect.TypeOf(got).String()
			gotValue := *got
			if gotType != tt.wantType {
				t.Errorf("Float32Ptr() = %v, want %v", gotType, tt.wantType)
			}
			if gotValue != tt.wantValue {
				t.Errorf("Float32Ptr() = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
func TestFloat64Ptr(t *testing.T) {
	type args struct {
		i float64
	}
	tests := []struct {
		name      string
		args      args
		wantType  string
		wantValue float64
	}{
		{
			name: "float64",
			args: args{
				i: 1.2,
			},
			wantType:  "*float64",
			wantValue: 1.2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Float64Ptr(tt.args.i)
			gotType := reflect.TypeOf(got).String()
			gotValue := *got
			if gotType != tt.wantType {
				t.Errorf("Float64Ptr() = %v, want %v", gotType, tt.wantType)
			}
			if gotValue != tt.wantValue {
				t.Errorf("Float64Ptr() = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
func TestBoolPtr(t *testing.T) {
	type args struct {
		i bool
	}
	tests := []struct {
		name      string
		args      args
		wantType  string
		wantValue bool
	}{
		{
			name: "bool",
			args: args{
				i: true,
			},
			wantType:  "*bool",
			wantValue: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BoolPtr(tt.args.i)
			gotType := reflect.TypeOf(got).String()
			gotValue := *got
			if gotType != tt.wantType {
				t.Errorf("BoolPtr() = %v, want %v", gotType, tt.wantType)
			}
			if gotValue != tt.wantValue {
				t.Errorf("BoolPtr() = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
