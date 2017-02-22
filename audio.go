package thumbnailer

// #include "audio.h"
import "C"
import (
	"io"
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
	return C.GoBytes(unsafe.Pointer(img.data), img.size)
}

// DetectMP3 returns if file is an MP3 file
func DetectMP3(rs io.ReadSeeker) (bool, error) {
	c, err := NewFFContext(rs)
	if err != nil {
		// Invalid file that can't even have a context created
		if fferr, ok := err.(ffError); ok && fferr.Code() == -1 {
			return false, nil
		}
		return false, err
	}
	defer c.Close()

	codec, err := c.CodecName(FFAudio)
	if err != nil {
		return false, err
	}
	return codec == "mp3", nil
}
