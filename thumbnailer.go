package thumbnailer

// #cgo pkg-config: GraphicsMagick
// #cgo CFLAGS: -std=c11 -D_POSIX_C_SOURCE
// #include "init.h"
// #include "thumbnailer.h"
// #include <stdlib.h>
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

// Thumbnailing errors
var (
	ErrTooWide             = errors.New("image too wide")
	ErrTooTall             = errors.New("image too tall")
	ErrThumbnailingUnknown = errors.New("unknown thumbnailing error")
)

// Image stores an image of known dimensions
type Image struct {
	Data []byte
	Dims
}

func init() {
	C.magickInit()
}

// processImage generates a thumbnail from a source image buffer. If width and
// height are non-zero, buf is assumed to be a raw Y420 image.
func processImage(src Source, opts Options) (Source, Thumbnail, error) {
	srcC := C.struct_Buffer{
		data:   (*C.uint8_t)(C.CBytes(src.Data)),
		size:   C.size_t(len(src.Data)),
		width:  C.ulong(src.Width),
		height: C.ulong(src.Height),
	}
	defer C.free(unsafe.Pointer(srcC.data))

	var ex C.ExceptionInfo
	defer C.DestroyExceptionInfo(&ex)

	optsC := C.struct_Options{
		JPEGCompression: C.uint8_t(opts.JPEGQuality),
		maxSrcDims: C.struct_Dims{
			width:  C.ulong(opts.MaxSourceDims.Width),
			height: C.ulong(opts.MaxSourceDims.Height),
		},
		thumbDims: C.struct_Dims{
			width:  C.ulong(opts.ThumbDims.Width),
			height: C.ulong(opts.ThumbDims.Height),
		},
	}

	var thumb C.struct_Thumbnail
	errCode := C.thumbnail(&srcC, &thumb, optsC, &ex)
	defer func() {
		if thumb.img.data != nil {
			C.free(unsafe.Pointer(thumb.img.data))
		}
	}()
	var err error
	if ex.reason != nil {
		err = extractError(ex)
	} else {
		switch errCode {
		case 0:
		case 1:
			err = ErrThumbnailingUnknown
		case 2:
			err = ErrTooWide
		case 3:
			err = ErrTooTall
		}
	}
	if err != nil {
		return src, Thumbnail{}, err
	}

	src.Width = uint(srcC.width)
	src.Height = uint(srcC.height)
	thumbnail := Thumbnail{
		IsPNG: bool(thumb.isPNG),
		Image: Image{
			Data: C.GoBytes(
				unsafe.Pointer(thumb.img.data),
				C.int(thumb.img.size),
			),
			Dims: Dims{
				Width:  uint(thumb.img.width),
				Height: uint(thumb.img.height),
			},
		},
	}
	return src, thumbnail, nil
}

func extractError(ex C.ExceptionInfo) error {
	r := C.GoString(ex.reason)
	d := C.GoString(ex.description)
	return fmt.Errorf(`thumbnailer: %s: %s`, r, d)
}
