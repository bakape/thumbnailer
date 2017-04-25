package thumbnailer

// #include "audio.h"
import "C"
import (
	"bytes"
	"errors"
	"unsafe"
)

// ErrNoCoverArt denotes no cover art has been found in the audio file, or a
// multipurpose media container file contained only audio and no cover art.
var ErrNoCoverArt = errors.New("no cover art found")

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
	return C.GoBytes(unsafe.Pointer(img.data), img.size)
}

func processAudio(src Source, opts Options) (Source, Thumbnail, error) {
	c, err := NewFFContext(bytes.NewReader(src.Data))
	if err != nil {
		return src, Thumbnail{}, err
	}
	defer c.Close()

	src.Length = c.Duration()
	c.ExtractMeta(&src)

	if !c.HasCoverArt() {
		return src, Thumbnail{}, ErrNoCoverArt
	}

	original := src.Data
	src.HasAudio = true
	src.Data = c.CoverArt()
	src, thumb, err := processImage(src, opts)
	src.Data = original
	return src, thumb, err
}
