package thumbnailer

import (
	"C"
	"sync"
	"unsafe"
)

// Initial minimum size of pooled buffer
const MinBufSize = 10 << 10

var bufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, MinBufSize)
	},
}

// Retrieve a buffer from the memory pool or allocate a new one.
// Use this to reduce large allocations in your side of the application.
func GetBuffer() []byte {
	return bufPool.Get().([]byte)
}

// Return or put a buffer into the pool.
// It is recommended not to put buffers with capacity smaller than MinBufSize
// into the pool.
// The buffer must not be used after this call.
func ReturnBuffer(buf []byte) {
	bufPool.Put(buf[:0])
}

// Copy a C buffer into a Go buffer from the pool
func copyCBuffer(data unsafe.Pointer, size C.int) []byte {
	buf := GetBuffer()
	if cap(buf) < int(size) {
		buf = make([]byte, 0, int(size))
	}
	buf = buf[0:int(size)]
	for i := 0; i < int(size); i++ {
		buf[i] = *(*byte)(unsafe.Pointer(uintptr(data) + uintptr(i)))
	}
	return buf
}
