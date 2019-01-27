package thumbnailer

// #cgo pkg-config: libavcodec libavutil libavformat libswscale
// #cgo CFLAGS: -std=c11
// #include "ffmpeg.h"
import "C"
import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
	"unsafe"
)

// FFMediaType correspond to the AVMediaType enum in ffmpeg
type FFMediaType int8

// Correspond to the AVMediaType enum in ffmpeg
const (
	FFUnknown FFMediaType = iota - 1
	FFVideo
	FFAudio
)

var (
	// Global map of AVIOHandlers. One handlers struct per format context.
	// Using AVFormatContext pointer address as a key.
	handlersMap = handlerMap{
		m: make(map[uintptr]io.ReadSeeker),
	}

	// ErrStreamNotFound denotes no steam of this media type was found
	ErrStreamNotFound = errors.New("no stream of this type found")
)

func init() {
	C.init()
}

// C can not retain any pointers to Go memory after the cgo call returns. We
// still need a way to bind AVFormatContext instances to Go I/O functions. To do
// that we convert the AVFormatContext pointer to a uintptr and use it as a key
// to look up the respective handlers on each call.
type handlerMap struct {
	sync.RWMutex
	m map[uintptr]io.ReadSeeker
}

func (h *handlerMap) Set(k uintptr, rs io.ReadSeeker) {
	h.Lock()
	h.m[k] = rs
	h.Unlock()
}

func (h *handlerMap) Delete(k uintptr) {
	h.Lock()
	delete(h.m, k)
	h.Unlock()
}

func (h *handlerMap) Get(k unsafe.Pointer) io.ReadSeeker {
	h.RLock()
	handlers, ok := h.m[uintptr(k)]
	h.RUnlock()
	if !ok {
		panic(fmt.Sprintf(
			"no handlers instance found, according to pointer: %v",
			k,
		))
	}
	return handlers
}

// Container for allocated codecs, so we can reuse them
type codecInfo struct {
	stream C.int
	ctx    *C.AVCodecContext
}

// ffError converts an FFmpeg error code to a Go error with a human-readable
// error message
type ffError C.int

// Error formats the FFmpeg error in human-readable format
func (f ffError) Error() string {
	buf := C.malloc(1024)
	defer C.free(buf)
	C.av_strerror(C.int(f), (*C.char)(buf), 1024)
	return fmt.Sprintf("ffmpeg: %s", C.GoString((*C.char)(buf)))
}

// Code returns the underlying FFmpeg error code
func (f ffError) Code() C.int {
	return C.int(f)
}

// FFContext is a wrapper for passing Go I/O interfaces to C
type FFContext struct {
	avFormatCtx *C.struct_AVFormatContext
	handlerKey  uintptr
	codecs      map[FFMediaType]codecInfo
}

// NewFFContext constructs a new AVIOContext and AVFormatContext.
// It is the responsibility of the caller to call Close() after finishing
// using the context.
func NewFFContext(rs io.ReadSeeker) (*FFContext, error) {
	ctx := C.avformat_alloc_context()
	this := &FFContext{
		avFormatCtx: ctx,
		codecs:      make(map[FFMediaType]codecInfo),
	}

	this.handlerKey = uintptr(unsafe.Pointer(ctx))
	handlersMap.Set(this.handlerKey, rs)

	err := C.create_context(&this.avFormatCtx)
	if err < 0 {
		this.Close()
		return nil, ffError(err)
	}
	if this.avFormatCtx == nil {
		this.Close()
		return nil, errors.New("unknown context creation error")
	}

	return this, nil
}

// Close closes and frees memory allocated for c. c should not be used after
// this point.
func (c *FFContext) Close() {
	for _, ci := range c.codecs {
		C.avcodec_free_context(&ci.ctx)
	}
	if c.avFormatCtx != nil {
		C.av_free(unsafe.Pointer(c.avFormatCtx.pb.buffer))
		c.avFormatCtx.pb.buffer = nil
		C.av_free(unsafe.Pointer(c.avFormatCtx.pb))
		C.av_free(unsafe.Pointer(c.avFormatCtx))
	}
	handlersMap.Delete(c.handlerKey)
}

// Allocate a codec context for the best stream of the passed FFMediaType, if
// not allocated already
func (c *FFContext) codecContext(typ FFMediaType) (codecInfo, error) {
	if ci, ok := c.codecs[typ]; ok {
		return ci, nil
	}

	var (
		ctx    *C.struct_AVCodecContext
		stream C.int
	)
	err := C.codec_context(&ctx, &stream, c.avFormatCtx, int32(typ))
	switch {
	case err == C.AVERROR_STREAM_NOT_FOUND:
		return codecInfo{}, ErrStreamNotFound
	case err < 0:
		return codecInfo{}, ffError(err)
	}

	ci := codecInfo{
		stream: stream,
		ctx:    ctx,
	}
	c.codecs[typ] = ci
	return ci, nil
}

// CodecName returns the codec name of the best stream of type typ
func (c *FFContext) CodecName(typ FFMediaType) (string, error) {
	ci, err := c.codecContext(typ)
	if err == nil {
		return C.GoString(ci.ctx.codec.name), nil
	}
	fferr, ok := err.(ffError)
	if ok && fferr.Code() == C.AVERROR_STREAM_NOT_FOUND {
		err = ErrStreamNotFound
	}
	return "", err
}

// HasStream returns, if the file has a decodeable stream of the passed type
func (c *FFContext) HasStream(typ FFMediaType) (bool, error) {
	_, err := c.codecContext(typ)
	switch err {
	case nil:
		return true, nil
	case ErrStreamNotFound:
		return false, nil
	default:
		return false, err
	}
}

// Duration returns the duration of the input
func (c *FFContext) Duration() time.Duration {
	return time.Duration(c.avFormatCtx.duration * 1000)
}

//export readCallBack
func readCallBack(opaque unsafe.Pointer, buf *C.uint8_t, bufSize C.int) C.int {
	s := (*[1 << 30]byte)(unsafe.Pointer(buf))[:bufSize:bufSize]
	n, err := handlersMap.Get(opaque).Read(s)
	if err != nil {
		return -1
	}
	return C.int(n)
}

//export seekCallBack
func seekCallBack(
	opaque unsafe.Pointer,
	offset C.int64_t,
	whence C.int,
) C.int64_t {
	n, err := handlersMap.Get(opaque).Seek(int64(offset), int(whence))
	if err != nil {
		return -1
	}
	return C.int64_t(n)
}
