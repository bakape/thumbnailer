// Package thumbnailer provides a more efficient image/video/audio/PDF
// thumbnailer than available with native Go processing libraries through
// GraphicsMagic and ffmpeg bindings.
package thumbnailer

import (
	"image"
	"io"
	"time"
)

// Source stores information about the source file
type Source struct {
	// Some containers may or may not have either
	HasAudio, HasVideo bool

	// Length of the stream. Applies to audio and video files.
	Length time.Duration

	// Mime type of the source file
	Mime string

	// optional metadata
	Title, Artist string

	// Canonical file extension
	Extension string

	Dims
}

// Dims store the dimensions of an image
type Dims struct {
	Width, Height uint
}

// Options suplied to the Thumbnail function
type Options struct {
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
	src Source, thumb image.Image, err error,
) {
	src.Mime, src.Extension, err = DetectMIME(rs, opts.AcceptedMimeTypes)
	if err != nil {
		return
	}

	override := overrideProcessors[src.Mime]
	if override != nil {
		thumb, err = override(rs, &src, opts)
		return
	}

	switch src.Mime {
	case
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
		"application/ogg",
		"video/webm",
		"video/x-matroska",
		"video/mp4",
		"video/avi",
		"video/quicktime",
		"video/x-ms-wmv",
		"video/x-flv",
		"audio/mpeg",
		"audio/aac",
		"audio/wave",
		"audio/x-flac",
		"audio/midi":
		thumb, err = processMedia(rs, &src, opts)
	default:
		err = UnsupportedMIMEError(src.Mime)
	}
	return
}
