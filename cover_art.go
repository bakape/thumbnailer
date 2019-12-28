package thumbnailer

// #include "cover_art.h"
import "C"
import (
	"bytes"
	"image"
	"unsafe"
)

// HasCoverArt return whether file has cover art in it
func (c *FFContext) HasCoverArt() bool {
	return C.find_cover_art(c.avFormatCtx) != -1
}

// CoverArt extracts any attached image
func (c *FFContext) CoverArt() []byte {
	img := C.retrieve_cover_art(c.avFormatCtx)
	if img.size <= 0 || img.data == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(img.data))
	return copyCBuffer(C.struct_Buffer{
		data: img.data,
		size: C.size_t(img.size),
	})
}

func processCoverArt(buf []byte, opts Options) (thumb image.Image, err error) {
	// Accept anything processable for cover art
	opts.AcceptedMimeTypes = nil

	_, thumb, err = Process(bytes.NewReader(buf), opts)

	// Propagate allowed failure errors for retry on the container itself
	// and wrap all other errors.
	switch err {
	case nil:
	case ErrTooWide, ErrTooTall, ErrCantThumbnail, ErrGetFrame:
		err = ErrCantThumbnail
	default:
		switch err.(type) {
		case ErrUnsupportedMIME, ErrInvalidImage:
			err = ErrCantThumbnail
		default:
			err = ErrCoverArt{err}
		}
	}
	return
}
