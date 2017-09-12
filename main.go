// Package thumbnailer provides a more efficient image/video/audio/PDF
// thumbnailer than available with native Go processing libraries through
// GraphicsMagic and ffmpeg bindings.
package thumbnailer

import (
	"io"
	"time"
)

// Thumbnail stores a processed thumbnail.
// Take note, that in case an audio file with no cover art is passed, this
// struct will be unassigned.
type Thumbnail struct {
	// Thumbnails can be either JPEG or PNG. Only images with transparency will
	// be PNG.
	IsPNG bool

	Data []byte
	Dims
}

// Source stores the source image, including information about the source file
type Source struct {
	// Some containers may or may not have either
	HasAudio, HasVideo, HasCoverArt bool

	// Length of the stream. Applies to audio and video files.
	Length time.Duration

	// Mime type of the source file
	Mime string

	// optional metadata
	Title, Artist string

	// Canonical file extension
	Extension string

	Data io.ReadSeeker
	Dims
}

// Dims store the dimensions of an image
type Dims struct {
	Width, Height uint64
}

// Options suplied to the Thumbnail function
type Options struct {
	// JPEG thumbnail quality to use. [0,100]. Defaults to 75.
	JPEGQuality uint8

	// Lossy PNG compression quality. [0,100]. Defaults to 30.
	PNGQuality uint8

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
func Process(r io.ReadSeeker, opts Options) (Source, Thumbnail, error) {
	if opts.JPEGQuality == 0 {
		opts.JPEGQuality = 75
	}
	if opts.PNGQuality == 0 {
		opts.PNGQuality = 30
	}

	src := Source{
		Data: r,
	}
	thumb, err := processFile(&src, opts)
	return src, thumb, err
}
