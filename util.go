package thumbnailer

// #include "thumbnailer.h"
// #include "string.h"
import "C"
import "unsafe"

// Copy a C buffer into a Go buffer from the pool
func copyCBuffer(src C.struct_Buffer) []byte {
	buf := make([]byte, int(src.size))
	C.memcpy(unsafe.Pointer(&buf[0]), unsafe.Pointer(src.data), src.size)
	return buf
}
