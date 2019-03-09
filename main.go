// Package thumbnailer provides a more efficient media thumbnailer than \
// available with native Go processing libraries through ffmpeg bindings.
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

	// Source dimensions, if file is image or video
	Dims

	// Mime type of the source file
	Mime string

	// Canonical file extension
	Extension string

	// Optional metadata
	Meta
}

// File metadata
type Meta struct {
	Title, Artist string
}

// Dims store the dimensions of an image
type Dims struct {
	Width, Height uint
}

// Options suplied to the Thumbnail function
type Options struct {
	// Maximum source image dimensions. Any image exceeding either will be
	// rejected and return with ErrTooTall or ErrTooWide.
	// If not set, all image processing will not be restricted by that
	// dimension.
	MaxSourceDims Dims

	// Target Maximum dimensions for the thumbnail.
	// Default to 150x150, if unset.
	ThumbDims Dims

	// MIME types to accept for thumbnailing.
	// If nil, all MIME types will be processed.
	//
	// To process MIME types that are a subset of archive files, like
	// "application/x-cbz", "application/x-cbr", "application/x-cb7" and
	// "application/x-cbt", you must accept the corresponding archive type
	// such as "application/zip" or leave this nil.
	AcceptedMimeTypes map[string]bool
}

// Process generates a thumbnail from a file of unknown type and performs some
// basic meta information extraction
func Process(rs io.ReadSeeker, opts Options) (
	src Source, thumb image.Image, err error,
) {
	if opts.ThumbDims.Width == 0 {
		opts.ThumbDims.Width = 150
	}
	if opts.ThumbDims.Height == 0 {
		opts.ThumbDims.Height = 150
	}

	src.Mime, src.Extension, err = DetectMIME(rs, opts.AcceptedMimeTypes)
	if err != nil {
		return
	}
	_, err = rs.Seek(0, 0)
	if err != nil {
		return
	}

	// TODO: PDF Processing
	// TODO: SVG processing

	var fn Processor

	override := overrideProcessors[src.Mime]
	if override != nil {
		fn = override
	} else {
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
			fn = processMedia
		case mimeZip:
			fn = processZip
		case mimeRar:
			fn = processRar
		default:
			err = ErrUnsupportedMIME(src.Mime)
			return
		}
	}

	thumb, err = fn(rs, &src, opts)
	return
}
