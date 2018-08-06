package thumbnailer

// #include <string.h>
import "C"
import (
	"io"
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
//
// Return the buffer to the pool using ReturnBuffer() when it is no longer being
// used.
func GetBuffer() []byte {
	return bufPool.Get().([]byte)
}

// Like GetBuffer() but the capacity of the returned buffer will be at least
// `capacity`. This can greatly reduce reallocation, if you already have a hint
// of the end size of the buffer.
func GetBufferCap(capacity int) []byte {
	if capacity < MinBufSize {
		capacity = MinBufSize
	}
	b := GetBuffer()
	if cap(b) < capacity {
		ReturnBuffer(b)
		b = make([]byte, 0, capacity)
	}
	return b
}

// Reads data from r until EOF and appends it to the buffer, growing the buffer
// as needed. The return value n is the number of bytes read. Any error except
// io.EOF encountered during the read is also returned. If the buffer becomes
// too large, ReadFrom will panic with ErrTooLarge.
//
// Unlike similar functions in the standard library this function uses the
// internal memory pool for reducing reallocations and decreasing heap sizes
// during heavy thumbnailing workloads.
//
// Return the buffer to the pool using ReturnBuffer() when it is no longer being
// used.
func ReadFrom(r io.Reader) ([]byte, error) {
	return ReadInto(GetBuffer(), r)
}

// Like ReadFrom() but the suplied buffer is used for reading. The supplied
// buffer should not be used after this call.
func ReadInto(buf []byte, r io.Reader) ([]byte, error) {
	for {
		// Make for more room and return old buffer to the pool for reuse
		if cap(buf)-len(buf) < 512 {
			new := make([]byte, len(buf), cap(buf)*2)
			copy(new, buf)
			ReturnBuffer(buf)
			buf = new
		}

		n, err := r.Read(buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+n]
		switch err {
		case nil:
		case io.EOF:
			return buf, nil
		default:
			ReturnBuffer(buf)
			return nil, err
		}
	}
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
	buf := GetBufferCap(int(size))
	buf = buf[0:int(size)]
	C.memcpy(unsafe.Pointer(&buf[0]), data, C.size_t(size))
	return buf
}
