// Package thumbnailer provides a more efficient image/video/audio/PDF
// thumbnailer than available with native Go processing libraries through
// GraphicsMagic and ffmpeg bindings.
package thumbnailer

import (
	"io"
	"io/ioutil"
	"time"
)

// Thumbnail stores a processed thumbnail.
// Take note, that in case an audio file with no cover art is passed, this
// struct will be unassigned.
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

	// optional metadata
	Title, Artist *string

	// Canonical file extension
	Extension string

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

	// MIME types to accept for thumbnailing. If nil, all MIME types will be
	// processed.
	AcceptedMimeTypes map[string]bool
}

// Process generates a thumbnail from a file of unknown type and performs some
// basic meta information extraction
func Process(rs io.ReadSeeker, opts Options) (
	src Source, thumb Thumbnail, err error,
) {
	src.Mime, src.Extension, err = DetectMIME(rs, opts.AcceptedMimeTypes)
	if err != nil {
		return
	}

	_, err = rs.Seek(0, 0)
	if err != nil {
		return
	}
	src.Data, err = ioutil.ReadAll(rs)
	if err != nil {
		return
	}

	return processFile(src, opts)
}

// ProcessBuffer is like Process, but takes []byte as input. More efficient,
// if you already have the file buffered into memory.
func ProcessBuffer(buf []byte, opts Options) (
	src Source, thumb Thumbnail, err error,
) {
	src.Mime, src.Extension, err = DetectMIMEBuffer(buf, opts.AcceptedMimeTypes)
	if err != nil {
		return
	}

	src.Data = buf
	return processFile(src, opts)
}
