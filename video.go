package thumbnailer

// #cgo pkg-config: libavcodec libavutil libavformat
// #cgo CFLAGS: -std=c11
// #include "video.h"
import "C"
import (
	"errors"
	"unsafe"
)

// Thumbnail extracts the first frame of the video
func (c *FFContext) Thumbnail() (
	buf []byte, width uint, height uint, err error,
) {
	ci, err := c.codecContext(FFVideo)
	if err != nil {
		return
	}

	var img C.struct_Buffer
	eErr := C.extract_video_image(&img, c.avFormatCtx, ci.ctx, ci.stream)
	switch {
	case eErr != 0:
		err = ffError(eErr)
		return
	case img.data == nil:
		err = errors.New("failed to get frame")
		return
	default:
		buf = C.GoBytes(unsafe.Pointer(img.data), C.int(img.size))
		C.free(unsafe.Pointer(img.data))
		return buf, uint(img.width), uint(img.height), nil
	}
}
