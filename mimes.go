package thumbnailer

import (
	"bytes"
	"encoding/binary"
	"image"
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
	&exactSig{"mp3", "audio/mpeg", []byte("\xFF\xFB")},
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
	&exactSig{"zip", mimeZip, []byte("\x50\x4B\x03\x04")},
	&exactSig{"rar", mimeRar, []byte("\x52\x61\x72\x20\x1A\x07\x00")},

	// RAR v5 archive
	&exactSig{"rar", mimeRar, []byte("\x52\x61\x72\x21\x1A\x07\x01\x00")},

	&exactSig{"7z", mime7Zip, []byte{'7', 'z', 0xBC, 0xAF, 0x27, 0x1C}},
}

var (
	// User-defined MIME type processors
	overrideProcessors = map[string]Processor{}
)

// Processor is a specialized file processor for a specific file type.
// Returns thumbnail and error.
//
// io.ReadSeeker is the start position, when passed to Processor.
type Processor func(io.ReadSeeker, *Source, Options) (image.Image, error)

// Matcher takes up to the first 4 KB of a file and returns the MIME type
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
	if codec == "mp3" || codec == "mp3float" {
		return "audio/mpeg", "mp3"
	}
	return
}

// RegisterMatcher adds an extra magic prefix-based MIME type matcher to the
// default set with an included canonical file extension.
//
// Not safe to use concurrently with file processing.
func RegisterMatcher(m Matcher) {
	matchers = append(matchers, m)
}

// RegisterProcessor registers a file processor for a specific MIME type.
// Can be used to add support for additional MIME types or as an override.
//
// Not safe to use concurrently with file processing.
func RegisterProcessor(mime string, fn Processor) {
	overrideProcessors[mime] = fn
}

// DetectMIME  detects the MIME typ of the rs.
//
// accepted: if not nil, specifies MIME types to not reject with
// ErrUnsupportedMIME
func DetectMIME(rs io.ReadSeeker, accepted map[string]bool,
) (
	mime, ext string, err error,
) {
	_, err = rs.Seek(0, 0)
	if err != nil {
		return
	}
	buf := make([]byte, sniffSize)
	read, err := rs.Read(buf)
	if err != nil {
		return
	}
	if read < sniffSize {
		buf = buf[:read]
	}

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
		err = ErrUnsupportedMIME("application/octet-stream")
	// Check if MIME is accepted, if specified
	case accepted != nil && !accepted[mime]:
		err = ErrUnsupportedMIME(mime)
	}
	return
}
