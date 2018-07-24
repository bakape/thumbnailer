package thumbnailer

// #cgo pkg-config: GraphicsMagick
// #cgo CFLAGS: -std=c11 -D_POSIX_C_SOURCE -O0 -g3
// #include "init.h"
// #include "thumbnailer.h"
// #include <stdlib.h>
import "C"
import (
	"errors"
	"strings"
	"unsafe"
)

// Image stores an image of known dimensions.
// To decrease allocations call ReturnBuffer() on Data, after you are done using
// Image.
type Image struct {
	Data []byte
	Dims
}

func init() {
	C.magickInit()
}

// processImage generates a thumbnail from a source image buffer. If width and
// height are non-zero, buf is assumed to be a raw RGBA image.
func processImage(src Source, opts Options) (Source, Thumbnail, error) {
	srcC := C.struct_Buffer{
		data:   (*C.uint8_t)(C.CBytes(src.Data)),
		size:   C.size_t(len(src.Data)),
		width:  C.ulong(src.Width),
		height: C.ulong(src.Height),
	}

	optsC := C.struct_Options{
		JPEGCompression: C.uint8_t(opts.JPEGQuality),
		PNGCompression: C.struct_CompressionRange{
			min: C.uint8_t(opts.PNGQuality.Min),
			max: C.uint8_t(opts.PNGQuality.Max),
		},
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
	errC := C.thumbnail(&srcC, &thumb, optsC)
	defer func() {
		if thumb.img.data != nil {
			C.free(unsafe.Pointer(thumb.img.data))
		}
	}()
	if errC != nil {
		var err error
		s := C.GoString(errC)
		C.free(unsafe.Pointer(errC))
		switch {
		case s == "too wide":
			err = ErrTooWide
		case s == "too tall":
			err = ErrTooTall
		case strings.HasPrefix(s, "Magick: Corrupt image"):
			err = ErrCorruptImage(s)
		default:
			err = errors.New(s)
		}
		return src, Thumbnail{}, err
	} else if thumb.img.data == nil {
		return src, Thumbnail{}, ErrThumbnailingUnknown
	}

	src.Width = uint(srcC.width)
	src.Height = uint(srcC.height)
	thumbnail := Thumbnail{
		IsPNG: bool(thumb.isPNG),
		Image: Image{
			Data: copyCBuffer(
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
