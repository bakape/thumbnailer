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
	"time"
	"unsafe"
)

// Thumbnailing errors
var (
	ErrTooWide             = errors.New("image too wide") // No such thing
	ErrTooTall             = errors.New("image too tall")
	ErrThumbnailingUnknown = errors.New("unknown thumbnailing error")
)

// Image stores an image of known dimensions
type Image struct {
	Data []byte
	Dims
}

// Thumbnail stores a processed thumbnail
type Thumbnail struct {
	// Thumbnails can be either JPEG or PNG. Only images with transparency will
	// be PNG.
	IsPNG bool
	Image
}

// Source stores the source image, including information about the source file
type Source struct {
	// Some containers may or may not have either
	HasAudio, HasVideo bool

	// Length of the stream. Applies to audio and video files.
	Length time.Duration

	// Mime type of the source file
	Mime string

	Image
}

// Dims store the dimensions of an image
type Dims struct {
	Width, Height uint
}

// Options suplied to the Thumbnail function
type Options struct {
	// JPEG thumbnail quality to use. [0,100]
	JPEGQuality uint8

	// Maximum source image dimensions. Any image exceeding either will be
	// rejected and return with ErrTooTall or ErrTooWide. If not set, all images
	// are processed.
	MaxSourceDims Dims

	// Target Maximum dimensions for the thumbnail
	ThumbDims Dims

	// MIME types to accept for thumbnailing. If nil, all MIME types will
	// processed.
	AcceptedMimeTypes []string
}

func init() {
	C.magickInit()
}

// processImage generates a thumbnail from a source image buffer. If width and
// height are non-zero, buf is assumed to be a raw Y420 image.
func processImage(src *Source, opts Options) (Thumbnail, error) {
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
		return Thumbnail{}, err
	}

	src.Width = uint(srcC.width)
	src.Height = uint(srcC.height)
	return Thumbnail{
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
	}, nil
}

func extractError(ex C.ExceptionInfo) error {
	r := C.GoString(ex.reason)
	d := C.GoString(ex.description)
	return fmt.Errorf(`thumbnailer: %s: %s`, r, d)
}
