package reflex

import (
	"reflect"
	"unsafe"
	"uuid"
)

// Compile-time proof that uuid.UUID is a plain 16-byte array, which the
// unsafe reinterpretations in ValueUUID/ValueUUIDPtr rely on.
var _ [16]byte = uuid.UUID{}

// TypeToPointer converts a reflect.Type to unsafe.Pointer type.
func TypeToPointer(t reflect.Type) unsafe.Pointer {
	return unsafe.Pointer(reflect.ValueOf(t).Pointer())
}

// ValueToPointer converts a reflect.Value to unsafe.Pointer type.
func ValueToPointer(v reflect.Value) unsafe.Pointer {
	return unsafe.Pointer(v.Pointer())
}

// PointerToObject converts an unsafe.Pointer to a value of T type.
func PointerToObject[T any](ptr unsafe.Pointer) T {
	return *(*T)(ptr)
}

// UnsafeSet sets a value at a specified offset within a struct.
// 'structPtr' is a pointer to the struct, 'offset' is the byte offset
// within the struct where the value should be set, and 'valuePtr' is a
// pointer to the value to be set.
func UnsafeSet(structPtr unsafe.Pointer, offset uintptr, valuePtr unsafe.Pointer) {
	// Calculate the address at the offset and cast it to a pointer to uintptr.
	fieldPtr := (*uintptr)(unsafe.Add(structPtr, offset))

	// Dereference 'valuePtr' as uintptr and store it at the calculated address.
	*fieldPtr = *(*uintptr)(valuePtr)
}

// UnsafeSet2 sets a value at a specified offset within a memory block.
// 'memoryPtr' is a pointer to the start of the memory block, 'offset' is the byte offset
// within the block where the value should be set, 'valuePtr' is a
// pointer to the value to be set and 'size' is the number of bytes to copy.
func UnsafeSet2(memoryPtr unsafe.Pointer, offset uintptr, valuePtr unsafe.Pointer, size uintptr) {
	// Calculate the address at the offset.
	fieldPtr := unsafe.Add(memoryPtr, offset)

	// Perform the byte-wise copy.
	copy(unsafe.Slice((*byte)(fieldPtr), size), unsafe.Slice((*byte)(valuePtr), size))
}

// UnsafeCopy copies a block of memory from 'src' to 'dst'.
// 'dst' and 'src' are pointers to the destination and source memory blocks, respectively,
// and 'size' is the number of bytes to copy.
func UnsafeCopy(dst, src unsafe.Pointer, size uintptr) {
	// Create byte slices that point to the source and destination memory.
	// unsafe.Slice carries no fixed array bound, unlike the pre-Go-1.17
	// (*[1 << 30]byte)(ptr)[:size:size] idiom this replaces, which silently
	// capped size at 1GiB and could not even compile-check on 32-bit targets.
	dstSlice := unsafe.Slice((*byte)(dst), size)
	srcSlice := unsafe.Slice((*byte)(src), size)

	// Perform the byte-wise copy.
	copy(dstSlice, srcSlice)
}

// UnsafeClone creates a new instance of a value by copying the memory block.
// 'src' is a pointer to the source value.
// The function returns a pointer to the newly created value.
// Note: The function does not perform a deep copy of the value.
// It only copies the memory block, so the new value will share pointers with the original value.
func UnsafeClone[T any](src *T) *T {
	// Create a new instance of the value using low-level code.
	// size := unsafe.Sizeof(*src)
	// dst := unsafe.Pointer(&make([]byte, size)[0])
	// UnsafeCopy(dst, unsafe.Pointer(src), size)
	// return (*T)(dst)
	// But let's make it more clear and readable:
	dst := new(T)
	UnsafeCopy(unsafe.Pointer(dst), unsafe.Pointer(src), unsafe.Sizeof(*src))
	return dst
}

// func clonePointers(dst, src unsafe.Pointer, size uintptr) {
// 	for offset := uintptr(0); offset < size; offset += unsafe.Alignof(uintptr(0)) {
// 		fieldPtr := (*uintptr)(unsafe.Pointer(uintptr(src) + offset))
// 	}
// }

// UnsafeInterface is a mimic of the internal eface structure,
// made after a whole day of research.
// It is used to access the internal data of an any.
// It is not safe to use and sometimes its performance is worse than the safe way (casting),
// use it only if you know what you are doing.
//
// Data is unsafe.Pointer rather than uintptr on purpose, mirroring the runtime's
// own eface. A uintptr is a plain integer that the garbage collector does not
// trace, so storing an address in one and converting it back to a pointer later
// is invalid under the unsafe.Pointer rules. Declaring it as a pointer keeps the
// word visible to the collector and makes Ptr a field read instead of a cast.
type UnsafeInterface struct {
	Type uintptr        // (*[2]uintptr)(unsafe.Pointer(&v))[0]
	Data unsafe.Pointer // (*[2]uintptr)(unsafe.Pointer(&v))[1]
}

// UnsafeInterfaceOf returns the (unsafe) interface of the data.
func UnsafeInterfaceOf(v any) *UnsafeInterface {
	return (*UnsafeInterface)(unsafe.Pointer(&v))
}

// UnsafeKind represents the (unsafe) kind of the data.
// See `UnsafeInterface.UnsafeKind` method.
type UnsafeKind uint8

const (
	// UnsafeInvalidKind is the invalid kind.
	UnsafeInvalidKind UnsafeKind = iota
	// UnsafeBoolKind is the bool kind.
	UnsafeBoolKind
	// UnsafeIntKind is the int kind.
	UnsafeIntKind
	// UnsafeInt8Kind is the int8 kind.
	UnsafeInt8Kind
	// UnsafeInt16Kind is the int16 kind.
	UnsafeInt16Kind
	// UnsafeInt32Kind is the int32 kind.
	UnsafeInt32Kind
	// UnsafeInt64Kind is the int64 kind.
	UnsafeInt64Kind
	// UnsafeUintKind is the uint kind.
	UnsafeUintKind
	// UnsafeUint8Kind is the uint8 kind.
	UnsafeUint8Kind
	// UnsafeUint16Kind is the uint16 kind.
	UnsafeUint16Kind
	// UnsafeUint32Kind is the uint32 kind.
	UnsafeUint32Kind
	// UnsafeUint64Kind is the uint64 kind.
	UnsafeUint64Kind
	// UnsafeUintptrKind is the uintptr kind.
	UnsafeUintptrKind
	// UnsafeFloat32Kind is the float32 kind.
	UnsafeFloat32Kind
	// UnsafeFloat64Kind is the float64 kind.
	UnsafeFloat64Kind
	// UnsafeComplex64Kind is the complex64 kind.
	UnsafeComplex64Kind
	// UnsafeComplex128Kind is the complex128 kind.
	UnsafeComplex128Kind
	// UnsafeArrayIntKind is the array(int) kind.
	UnsafeArrayIntKind
	// UnsafeChanIntKind is the chan(int) kind.
	UnsafeChanIntKind
	// UnsafeFuncKind is the func kind.
	UnsafeFuncKind
	// UnsafeInterfaceKind is the interface kind.
	UnsafeInterfaceKind
	// UnsafeMapKind is the map[int]int kind.
	UnsafeMapKind
	// UnsafePtrKind is the ptr kind.
	UnsafePtrKind
	// UnsafeSliceKind is the slice kind.
	UnsafeSliceKind
	// UnsafeStringKind is the string kind.
	UnsafeStringKind
	// UnsafeStructKind is the struct kind.
	UnsafeStructKind
	// UnsafeUnsafePointerKind is the unsafe.Pointer kind.
	UnsafeUnsafePointerKind
	// UnsafeUUIDKind is the uuid.UUID kind.
	UnsafeUUIDKind
)

// UnsafeKindOf returns the (unsafe) kind of the data.
func UnsafeKindOf(v any) UnsafeKind {
	return rtypeKind[(*[2]uintptr)(unsafe.Pointer(&v))[0]]
}

// String returns the string representation of the (unsafe) kind.
func (k UnsafeKind) String() string {
	switch k {
	case UnsafeInvalidKind:
		return "invalid"
	case UnsafeBoolKind:
		return "bool"
	case UnsafeIntKind:
		return "int"
	case UnsafeInt8Kind:
		return "int8"
	case UnsafeInt16Kind:
		return "int16"
	case UnsafeInt32Kind:
		return "int32"
	case UnsafeInt64Kind:
		return "int64"
	case UnsafeUintKind:
		return "uint"
	case UnsafeUint8Kind:
		return "uint8"
	case UnsafeUint16Kind:
		return "uint16"
	case UnsafeUint32Kind:
		return "uint32"
	case UnsafeUint64Kind:
		return "uint64"
	case UnsafeUintptrKind:
		return "uintptr"
	case UnsafeFloat32Kind:
		return "float32"
	case UnsafeFloat64Kind:
		return "float64"
	case UnsafeComplex64Kind:
		return "complex64"
	case UnsafeComplex128Kind:
		return "complex128"
	case UnsafeArrayIntKind:
		return "array(int)"
	case UnsafeChanIntKind:
		return "chan(int)"
	case UnsafeFuncKind:
		return "func"
	case UnsafeInterfaceKind:
		return "interface"
	case UnsafeMapKind:
		return "map"
	case UnsafePtrKind:
		return "ptr"
	case UnsafeSliceKind:
		return "slice"
	case UnsafeStringKind:
		return "string"
	case UnsafeStructKind:
		return "struct"
	case UnsafeUnsafePointerKind:
		return "unsafe.Pointer"
	case UnsafeUUIDKind:
		return "uuid.UUID"
	default:
		return "unknown"
	}
}

var rtypeKind = map[uintptr]UnsafeKind{
	// complete all types for fast-access.
	UnsafeInterfaceOf(true).Type:                UnsafeBoolKind,
	UnsafeInterfaceOf(int(0)).Type:              UnsafeIntKind,
	UnsafeInterfaceOf(int8(0)).Type:             UnsafeInt8Kind,
	UnsafeInterfaceOf(int16(0)).Type:            UnsafeInt16Kind,
	UnsafeInterfaceOf(int32(0)).Type:            UnsafeInt32Kind,
	UnsafeInterfaceOf(int64(0)).Type:            UnsafeInt64Kind,
	UnsafeInterfaceOf(uint(0)).Type:             UnsafeUintKind,
	UnsafeInterfaceOf(uint8(0)).Type:            UnsafeUint8Kind,
	UnsafeInterfaceOf(uint16(0)).Type:           UnsafeUint16Kind,
	UnsafeInterfaceOf(uint32(0)).Type:           UnsafeUint32Kind,
	UnsafeInterfaceOf(uint64(0)).Type:           UnsafeUint64Kind,
	UnsafeInterfaceOf(uintptr(0)).Type:          UnsafeUintptrKind,
	UnsafeInterfaceOf(float32(0)).Type:          UnsafeFloat32Kind,
	UnsafeInterfaceOf(float64(0)).Type:          UnsafeFloat64Kind,
	UnsafeInterfaceOf(complex64(0)).Type:        UnsafeComplex64Kind,
	UnsafeInterfaceOf(complex128(0)).Type:       UnsafeComplex128Kind,
	UnsafeInterfaceOf(make(chan int)).Type:      UnsafeChanIntKind,
	UnsafeInterfaceOf(func() {}).Type:           UnsafeFuncKind,
	UnsafeInterfaceOf(any(nil)).Type:            UnsafeInterfaceKind,
	UnsafeInterfaceOf(map[int]int{}).Type:       UnsafeMapKind,
	UnsafeInterfaceOf(new(int)).Type:            UnsafePtrKind,
	UnsafeInterfaceOf([]int{}).Type:             UnsafeSliceKind,
	UnsafeInterfaceOf("").Type:                  UnsafeStringKind,
	UnsafeInterfaceOf(struct{}{}).Type:          UnsafeStructKind,
	UnsafeInterfaceOf(unsafe.Pointer(nil)).Type: UnsafeUnsafePointerKind,
	UnsafeInterfaceOf(uuid.UUID{}).Type:         UnsafeUUIDKind,
}

// UnsafeKind returns the (unsafe) kind of the data.
func (i *UnsafeInterface) UnsafeKind() UnsafeKind {
	return rtypeKind[i.Type]
}

// Ptr returns the unsafe.Pointer of the data.
func (i *UnsafeInterface) Ptr() unsafe.Pointer {
	return i.Data
}

// ValueBool returns the bool value of the data.
func (i *UnsafeInterface) ValueBool() bool {
	return *(*bool)(i.Ptr())
}

// ValueInt returns the int value of the data.
func (i *UnsafeInterface) ValueInt() int {
	return *(*int)(i.Ptr())
}

// ValueInt8 returns the int8 value of the data.
func (i *UnsafeInterface) ValueInt8() int8 {
	return *(*int8)(i.Ptr())
}

// ValueInt16 returns the int16 value of the data.
func (i *UnsafeInterface) ValueInt16() int16 {
	return *(*int16)(i.Ptr())
}

// ValueInt32 returns the int32 value of the data.
func (i *UnsafeInterface) ValueInt32() int32 {
	return *(*int32)(i.Ptr())
}

// ValueInt64 returns the int64 value of the data.
func (i *UnsafeInterface) ValueInt64() int64 {
	return *(*int64)(i.Ptr())
}

// ValueUint returns the uint value of the data.
func (i *UnsafeInterface) ValueUint() uint {
	return *(*uint)(i.Ptr())
}

// ValueUint8 returns the uint8 value of the data.
func (i *UnsafeInterface) ValueUint8() uint8 {
	return *(*uint8)(i.Ptr())
}

// ValueUint16 returns the uint16 value of the data.
func (i *UnsafeInterface) ValueUint16() uint16 {
	return *(*uint16)(i.Ptr())
}

// ValueUint32 returns the uint32 value of the data.
func (i *UnsafeInterface) ValueUint32() uint32 {
	return *(*uint32)(i.Ptr())
}

// ValueUint64 returns the uint64 value of the data.
func (i *UnsafeInterface) ValueUint64() uint64 {
	return *(*uint64)(i.Ptr())
}

// ValueUintptr returns the uintptr value of the data.
func (i *UnsafeInterface) ValueUintptr() uintptr {
	return *(*uintptr)(i.Ptr())
}

// ValueFloat32 returns the float32 value of the data.
func (i *UnsafeInterface) ValueFloat32() float32 {
	return *(*float32)(i.Ptr())
}

// ValueFloat64 returns the float64 value of the data.
func (i *UnsafeInterface) ValueFloat64() float64 {
	return *(*float64)(i.Ptr())
}

// ValueComplex64 returns the complex64 value of the data.
func (i *UnsafeInterface) ValueComplex64() complex64 {
	return *(*complex64)(i.Ptr())
}

// ValueComplex128 returns the complex128 value of the data.
func (i *UnsafeInterface) ValueComplex128() complex128 {
	return *(*complex128)(i.Ptr())
}

// ValueFunc returns the func value of the data.
func (i *UnsafeInterface) ValueFunc() func() {
	return *(*func())(i.Ptr())
}

// ValueInterface returns the any value of the data.
func (i *UnsafeInterface) ValueInterface() any {
	return *(*any)(i.Ptr())
}

// ValueString returns the string value of the data.
func (i *UnsafeInterface) ValueString() string {
	return *(*string)(i.Ptr())
}

// ValueUnsafePointer returns the unsafe.Pointer value of the data.
func (i *UnsafeInterface) ValueUnsafePointer() unsafe.Pointer {
	return *(*unsafe.Pointer)(i.Ptr())
}

// ValueUUID returns the uuid.UUID value of the data.
func (i *UnsafeInterface) ValueUUID() uuid.UUID {
	return *(*uuid.UUID)(i.Ptr())
}

// ValueUUIDPtr returns the *uuid.UUID value of the data.
func (i *UnsafeInterface) ValueUUIDPtr() *uuid.UUID {
	return (*uuid.UUID)(i.Ptr())
}

// UnsafeType returns the (unsafe) type of the data.
func (t *UnsafeInterface) UnsafeType() *UnsafeType {
	return (*UnsafeType)(t.Ptr())
}

// UnsafeType is a mimic of the internal rtype structure,
// made after a whole day of research.
// It is used to access the internal type's data of an any value.
// It is not safe to use, but it is useful for performance.
type UnsafeType struct {
	size       uintptr
	ptrdata    uintptr
	hash       uint32
	tflag      uint8
	align      uint8
	fieldAlign uint8
	kind       uint8
	alg        *uintptr
	gcdata     *byte
	str        int32
	ptrToThis  int32
}

// Size returns the size of the type.
func (t *UnsafeType) Size() uintptr {
	return t.size
}

// Kind returns the reflect.Kind of the type.
func (t *UnsafeType) Kind() reflect.Kind {
	return reflect.Kind(t.kind)
}
