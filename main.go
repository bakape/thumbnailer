// Package thumbnailer provides a more efficient image/video/audio/PDF
// thumbnailer than available with native Go processing libraries through
// GraphicsMagic and ffmpeg bindings.
package thumbnailer

import (
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/h2non/filetype"
	"gopkg.in/h2non/filetype.v1/types"
)

var (
	// Special file type processors
	mimeProcessors = map[string]MIMEProcessor{}
)

// MIMEProcessor is a specialized file processor for a specific file type
type MIMEProcessor func(Source, Options) (Source, Thumbnail, error)

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

	// Canonical file extension of the source file
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

// UnsupportedMIMEError indicates the MIME type of the file could not be
// detected as a supported type or was not in the AcceptedMimeTypes list, if
// defined.
type UnsupportedMIMEError string

func (u UnsupportedMIMEError) Error() string {
	return fmt.Sprintf("unsupported MIME type: %s", string(u))
}

// RegisterProcessor registers a file processor for a specific MIME type.
// Can be used to add support for additional MIME types or as an override.
// Not safe to use concurrently with file processing.
func RegisterProcessor(mime string, fn MIMEProcessor) {
	mimeProcessors[mime] = fn
}

// Process generates a thumbnail from a file of unknown type and performs some
// basic meta information extraction
func Process(rs io.ReadSeeker, opts Options) (
	src Source, thumb Thumbnail, err error,
) {
	src.Mime, src.Extension, err = detectMimeType(
		nil,
		rs,
		opts.AcceptedMimeTypes,
	)
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
	src.Mime, src.Extension, err = detectMimeType(
		buf,
		nil,
		opts.AcceptedMimeTypes,
	)
	if err != nil {
		return
	}

	src.Data = buf
	return processFile(src, opts)
}

// Can be passed either the full read file as []byte or io.ReadSeeker
func detectMimeType(buf []byte, rs io.ReadSeeker, accepted map[string]bool) (
	mime, ext string, err error,
) {
	var typ types.Type
	if buf != nil {
		typ, err = filetype.Match(buf)
	} else {
		typ, err = filetype.MatchReader(rs)
	}
	if err != nil {
		err = UnsupportedMIMEError(err.Error())
		return
	}
	mime = typ.MIME.Value
	ext = typ.Extension

	// Check if MIME is accepted, if specified
	if accepted != nil && !accepted[mime] {
		err = UnsupportedMIMEError(mime)
		return
	}

	return
}

func processFile(src Source, opts Options) (Source, Thumbnail, error) {
	override := mimeProcessors[src.Mime]
	if override != nil {
		return override(src, opts)
	}

	switch src.Mime {
	case
		"application/pdf",
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
		"image/tiff",
		"image/bmp",
		"image/vnd.adobe.photoshop",
		"image/x-icon":
		return processImage(src, opts)
	case
		"audio/midi",
		"audio/mpeg",
		"audio/m4a",
		"audio/ogg",
		"audio/x-flac",
		"audio/x-wav":
		return processAudio(src, opts)
	case
		"video/mp4",
		"video/x-matroska",
		"video/webm",
		"video/quicktime",
		"video/x-msvideo",
		"video/x-ms-wmv",
		"video/mpeg",
		"flv", "video/x-flv":
		return processVideo(src, opts)
	default:
		return src, Thumbnail{}, UnsupportedMIMEError(src.Mime)
	}
}
