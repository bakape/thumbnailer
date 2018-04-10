package thumbnailer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const sniffSize = 4 << 10

// Matching code partially adapted from "net/http/sniff.go"

// Mime type prefix magic number matchers and canonical extensions
var matchers = []Matcher{
	// Probably most common types, this library will be used for, first.
	// More expensive checks are also positioned lower.
	&exactSig{"jpg", "image/jpeg", []byte("\xFF\xD8\xFF")},
	&exactSig{"png", "image/png", []byte("\x89\x50\x4E\x47\x0D\x0A\x1A\x0A")},
	&exactSig{"gif", "image/gif", []byte("GIF87a")},
	&exactSig{"gif", "image/gif", []byte("GIF89a")},
	&maskedSig{
		"webp",
		"image/webp",
		[]byte("\xFF\xFF\xFF\xFF\x00\x00\x00\x00\xFF\xFF\xFF\xFF\xFF\xFF"),
		[]byte("RIFF\x00\x00\x00\x00WEBPVP"),
	},
	&maskedSig{
		"ogg",
		"application/ogg",
		[]byte("OggS\x00"),
		[]byte("\x4F\x67\x67\x53\x00"),
	},
	MatcherFunc(matchWebmOrMKV),
	&exactSig{"pdf", "application/pdf", []byte("%PDF-")},
	&maskedSig{
		"mp3",
		"audio/mpeg",
		[]byte("\xFF\xFF\xFF"),
		[]byte("ID3"),
	},
	MatcherFunc(matchMP4),
	&exactSig{"aac", "audio/aac", []byte("ÿñ")},
	&exactSig{"aac", "audio/aac", []byte("ÿù")},
	&exactSig{"bmp", "image/bmp", []byte("BM")},
	&maskedSig{
		"wav",
		"audio/wave",
		[]byte("\xFF\xFF\xFF\xFF\x00\x00\x00\x00\xFF\xFF\xFF\xFF"),
		[]byte("RIFF\x00\x00\x00\x00WAVE"),
	},
	&maskedSig{
		"avi",
		"video/avi",
		[]byte("\xFF\xFF\xFF\xFF\x00\x00\x00\x00\xFF\xFF\xFF\xFF"),
		[]byte("RIFF\x00\x00\x00\x00AVI "),
	},
	&exactSig{"psd", "image/photoshop", []byte("8BPS")},
	&exactSig{"flac", "audio/x-flac", []byte("fLaC")},
	&exactSig{"tiff", "image/tiff", []byte("II*\x00")},
	&exactSig{"tiff", "image/tiff", []byte("MM\x00*")},
	&exactSig{"mov", "video/quicktime", []byte("\x00\x00\x00\x14ftyp")},
	&exactSig{
		"wmv",
		"video/x-ms-wmv",
		[]byte{0x30, 0x26, 0xB2, 0x75, 0x8E, 0x66, 0xCF, 0x11, 0xA6, 0xD9},
	},
	&exactSig{"flv", "video/x-flv", []byte("FLV\x01")},
	&exactSig{"ico", "image/x-icon", []byte("\x00\x00\x01\x00")},
	&maskedSig{
		"midi",
		"audio/midi",
		[]byte("\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF"),
		[]byte("MThd\x00\x00\x00\x06"),
	},
}

var (
	// User-defined MIME type processors
	overrideProcessors = map[string]Processor{}
)

// Processor is a specialized file processor for a specific file type
type Processor func(Source, Options) (Source, Thumbnail, error)

// Matcher takes up to the first 512 bytes of a file and returns the MIME type
// and canonical extension, that were matched. Empty string indicates no match.
type Matcher interface {
	Match([]byte) (mime string, extension string)
}

// MatcherFunc is an adapter that allows using functions as Matcher
type MatcherFunc func([]byte) (string, string)

// Match implements Matcher
func (fn MatcherFunc) Match(data []byte) (string, string) {
	return fn(data)
}

type exactSig struct {
	ext, mime string
	sig       []byte
}

func (e *exactSig) Match(data []byte) (string, string) {
	if bytes.HasPrefix(data, e.sig) {
		return e.mime, e.ext
	}
	return "", ""
}

type maskedSig struct {
	ext, mime string
	mask, pat []byte
}

func (m *maskedSig) Match(data []byte) (string, string) {
	if len(data) < len(m.mask) {
		return "", ""
	}
	for i, mask := range m.mask {
		db := data[i] & mask
		if db != m.pat[i] {
			return "", ""
		}
	}
	return m.mime, m.ext
}

func matchWebmOrMKV(data []byte) (string, string) {
	switch {
	case len(data) < 8 || !bytes.HasPrefix(data, []byte("\x1A\x45\xDF\xA3")):
		return "", ""
	case bytes.Contains(data[4:], []byte("webm")):
		return "video/webm", "webm"
	case bytes.Contains(data[4:], []byte("matroska")):
		return "video/x-matroska", "mkv"
	default:
		return "", ""
	}
}

func matchMP4(data []byte) (string, string) {
	if len(data) < 12 {
		return "", ""
	}

	boxSize := int(binary.BigEndian.Uint32(data[:4]))
	nope := boxSize%4 != 0 ||
		len(data) < boxSize ||
		!bytes.Equal(data[4:8], []byte("ftyp"))
	if nope {
		return "", ""
	}

	for st := 8; st < boxSize; st += 4 {
		if st == 12 {
			// minor version number
			continue
		}
		if bytes.Equal(data[st:st+3], []byte("mp4")) ||
			bytes.Equal(data[st:st+4], []byte("dash")) {
			return "video/mp4", "mp4"
		}
	}
	return "", ""
}

// MP3 is a retarded standard, that will not always even have a magic number.
// Need to detect with FFMPEG as a last resort.
func matchMP3(data []byte) (mime string, ext string) {
	c, err := NewFFContext(bytes.NewReader(data))
	if err != nil {
		return
	}
	defer c.Close()

	codec, err := c.CodecName(FFAudio)
	if err != nil {
		return
	}
	if codec == "mp3" {
		return "audio/mpeg", "mp3"
	}
	return
}

// UnsupportedMIMEError indicates the MIME type of the file could not be
// detected as a supported type or was not in the AcceptedMimeTypes list, if
// defined.
type UnsupportedMIMEError string

func (u UnsupportedMIMEError) Error() string {
	return fmt.Sprintf("unsupported MIME type: %s", string(u))
}

// RegisterMatcher adds an extra magic prefix-based MIME type matcher to the
// default set with an included canonical file extension.
// Not safe to use concurrently with file processing.
func RegisterMatcher(m Matcher) {
	matchers = append(matchers, m)
}

// RegisterProcessor registers a file processor for a specific MIME type.
// Can be used to add support for additional MIME types or as an override.
// Not safe to use concurrently with file processing.
func RegisterProcessor(mime string, fn Processor) {
	overrideProcessors[mime] = fn
}

// DetectMIME  detects the MIME typ of the r. r must be at starting position.
// accepted, if not nil, specifies MIME types to not reject with
// UnsupportedMIMEError.
func DetectMIME(r io.Reader, accepted map[string]bool) (string, string, error) {
	buf := make([]byte, sniffSize)
	read, err := r.Read(buf)
	if err != nil {
		return "", "", err
	}
	if read < sniffSize {
		buf = buf[:read]
	}
	return detectMimeType(buf, accepted)
}

// DetectMIMEBuffer is like DetectMIME, but accepts a []byte slice already
// loaded into memory.
func DetectMIMEBuffer(buf []byte, accepted map[string]bool) (
	string, string, error,
) {
	if len(buf) > sniffSize {
		buf = buf[:sniffSize]
	}
	return detectMimeType(buf, accepted)
}

// Can be passed either the full read file as []byte or io.ReadSeeker
func detectMimeType(buf []byte, accepted map[string]bool) (
	mime, ext string, err error,
) {
	for _, m := range matchers {
		mime, ext = m.Match(buf)
		if mime != "" {
			break
		}
	}

	if mime == "" {
		if accepted == nil || accepted["audio/mpeg"] {
			mime, ext = matchMP3(buf)
		}
	}

	switch {
	case mime == "":
		err = UnsupportedMIMEError("application/octet-stream")
	// Check if MIME is accepted, if specified
	case accepted != nil && !accepted[mime]:
		err = UnsupportedMIMEError(mime)
	}
	return
}

func processFile(src Source, opts Options) (Source, Thumbnail, error) {
	override := overrideProcessors[src.Mime]
	if override != nil {
		return override(src, opts)
	}

	switch src.Mime {
	case
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
		"application/pdf",
		"image/bmp",
		"image/photoshop",
		"image/tiff",
		"image/x-icon":
		return processImage(src, opts)
	case
		"audio/mpeg",
		"audio/aac",
		"audio/wave",
		"audio/x-flac",
		"audio/midi":
		return processAudio(src, opts)
	case
		"application/ogg",
		"video/webm",
		"video/x-matroska",
		"video/mp4",
		"video/avi",
		"video/quicktime",
		"video/x-ms-wmv",
		"video/x-flv":
		return processVideo(src, opts)
	default:
		return src, Thumbnail{}, UnsupportedMIMEError(src.Mime)
	}
}
