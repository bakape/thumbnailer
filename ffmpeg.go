package thumbnailer

// #cgo pkg-config: libavcodec libavutil libavformat libswscale
// #cgo CFLAGS: -std=c11 -g
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

	// Input format specifiers for FFmpeg. These save FFmpeg some overhead on
	// format detection and also prevent failure to open input on format
	// detection failure.
	inputFormats = map[string]*C.char{
		"image/jpeg":       C.CString("mjpeg"),
		"image/png":        C.CString("image2"),
		"image/gif":        C.CString("gif"),
		"image/webp":       C.CString("webp"),
		"application/ogg":  C.CString("ogg"),
		"video/webm":       C.CString("webm"),
		"video/x-matroska": C.CString("matroska"),
		"video/mp4":        C.CString("mp4"),
		"video/avi":        C.CString("avi"),
		"video/quicktime":  C.CString("mp4"),
		"video/x-flv":      C.CString("flv"),
		"audio/mpeg":       C.CString("mp3"),
		"audio/aac":        C.CString("aac"),
		"audio/wave":       C.CString("wav"),
		"audio/x-flac":     C.CString("flac"),
	}
)

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
		panic(fmt.Errorf("no handler instance found for pointer: %v", k))
	}
	return handlers
}

// Container for allocated codecs, so we can reuse them
type codecInfo struct {
	stream C.int
	ctx    *C.AVCodecContext
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
	return newFFContextWithFormat(rs, nil)
}

// Like NewFFContext, but optionally specifies the passed input format explicitly.
// inputFormat can be NULL.
func newFFContextWithFormat(rs io.ReadSeeker, inputFormat *C.char,
) (*FFContext, error) {
	ctx := C.avformat_alloc_context()
	this := &FFContext{
		avFormatCtx: ctx,
		codecs:      make(map[FFMediaType]codecInfo),
	}

	this.handlerKey = uintptr(unsafe.Pointer(ctx))
	handlersMap.Set(this.handlerKey, rs)

	err := C.create_context(&this.avFormatCtx, inputFormat)
	if err < 0 {
		this.Close()
		return nil, castError(err)
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
		return codecInfo{}, castError(err)
	}

	ci := codecInfo{
		stream: stream,
		ctx:    ctx,
	}
	c.codecs[typ] = ci
	return ci, nil
}

// CodecName returns the codec name of the best stream of type typ
func (c *FFContext) CodecName(typ FFMediaType) (codec string, err error) {
	ci, err := c.codecContext(typ)
	if err != nil {
		return
	}
	return C.GoString(ci.ctx.codec.name), nil
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

// Length returns the duration of the input
func (c *FFContext) Length() time.Duration {
	return time.Duration(c.avFormatCtx.duration * 1000)
}

// Dims returns dimensions of the best video (or image) stream in the media
func (c *FFContext) Dims() (dims Dims, err error) {
	ci, err := c.codecContext(FFVideo)
	if err != nil {
		return
	}
	dims = Dims{
		Width:  uint(ci.ctx.width),
		Height: uint(ci.ctx.height),
	}
	return
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
