package thumbnailer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
)

var ErrUnsupportedMIME = errors.New("file MIME type not supported")

// Size of buffer to perform MIME sniffing on
const sniffSize = 512

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
		exactSig{
			"webp",
			"image/webp",
			[]byte("RIFF\x00\x00\x00\x00WEBPVP"),
		},
		[]byte("\xFF\xFF\xFF\xFF\x00\x00\x00\x00\xFF\xFF\xFF\xFF\xFF\xFF"),
	},
	&maskedSig{
		exactSig{
			"ogg",
			"application/ogg",
			[]byte("\x4F\x67\x67\x53\x00"),
		},
		[]byte("OggS\x00"),
	},

	// Webm is a subset of MKV, so match Webm first
	NewFuncMatcher("video/webm", "webm", matchWebm),
	NewFuncMatcher("video/x-matroska", "mkv", matchMkv),

	&exactSig{"pdf", "application/pdf", []byte("%PDF-")},
	&maskedSig{
		exactSig{
			"mp3",
			"audio/mpeg",
			[]byte("ID3"),
		},
		[]byte("\xFF\xFF\xFF"),
	},
	NewFuncMatcher("video/mp4", "mp4", matchMP4),
	&exactSig{"aac", "audio/aac", []byte("ÿñ")},
	&exactSig{"aac", "audio/aac", []byte("ÿù")},
	&exactSig{"bmp", "image/bmp", []byte("BM")},
	&maskedSig{
		exactSig{
			"wav",
			"audio/wave",
			[]byte("RIFF\x00\x00\x00\x00WAVE"),
		},
		[]byte("\xFF\xFF\xFF\xFF\x00\x00\x00\x00\xFF\xFF\xFF\xFF"),
	},
	&maskedSig{
		exactSig{
			"avi",
			"video/avi",
			[]byte("RIFF\x00\x00\x00\x00AVI "),
		},
		[]byte("\xFF\xFF\xFF\xFF\x00\x00\x00\x00\xFF\xFF\xFF\xFF"),
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
		exactSig{
			"midi",
			"audio/midi",
			[]byte("MThd\x00\x00\x00\x06"),
		},
		[]byte("\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF"),
	},
	NewFuncMatcher("audio/mpeg", "mp3", matchMp3),
}

var (
	// User-defined MIME type processors
	overrideProcessors = map[string]Processor{}
)

// Processor is a specialized file processor for a specific file type
type Processor func(*Source, Options) (Thumbnail, error)

// Matcher determines if a file matches an MIME type
type Matcher interface {
	// Takes up to the first 512 bytes of a file and returns, if the MIME type
	//  was matched.
	// If additional data is needed, it may use rs to read data beyond the first
	// 512 bytes.
	Match(start []byte, rs io.ReadSeeker) bool

	// Returns MIME type
	MIME() string

	// Returns canonical MIME type extension without leading dot
	Ext() string
}

type exactSig struct {
	ext, mime string
	sig       []byte
}

func (e *exactSig) Match(data []byte, _ io.ReadSeeker) bool {
	return bytes.HasPrefix(data, e.sig)
}

func (e *exactSig) MIME() string {
	return e.mime
}

func (e *exactSig) Ext() string {
	return e.ext
}

type maskedSig struct {
	exactSig
	mask []byte
}

func (m *maskedSig) Match(data []byte, _ io.ReadSeeker) bool {
	if len(data) < len(m.mask) {
		return false
	}
	for i, mask := range m.mask {
		db := data[i] & mask
		if db != m.sig[i] {
			return false
		}
	}
	return true
}

// Matches a MIME type with a provided function
type funcMatcher struct {
	exactSig
	match MatchFunc
}

func (f *funcMatcher) Match(start []byte, rs io.ReadSeeker) bool {
	return f.match(start, rs)
}

type MatchFunc func(start []byte, rs io.ReadSeeker) bool

// Constructs a Matcher from a a function
func NewFuncMatcher(mime, ext string, fn MatchFunc) Matcher {
	return &funcMatcher{
		exactSig{
			ext,
			mime,
			nil,
		},
		fn,
	}
}

func matchWebm(data []byte, _ io.ReadSeeker) bool {
	return matchWebmOrMkv(data, []byte("webm"))
}

func matchMkv(data []byte, _ io.ReadSeeker) bool {
	return matchWebmOrMkv(data, []byte("matroska"))
}

func matchWebmOrMkv(data []byte, contains []byte) bool {
	return len(data) > 8 &&
		bytes.HasPrefix(data, []byte("\x1A\x45\xDF\xA3")) &&
		bytes.Contains(data[4:], contains)
}

// MP3 is a retarded standard, that will not always even have a magic number.
// Need to detect with FFMPEG as a last resort.
func matchMp3(_ []byte, rs io.ReadSeeker) bool {
	buf, err := execCommand(
		rs, "ffprobe", "-",
		"-hide_banner",
		"-v", "fatal",
		"-of", "compact",
		"-show_entries", "format=format_name",
	)
	if err != nil {
		return false
	}
	defer PutBuffer(buf)

	return strings.TrimPrefix(buf.String(), "format|format_name=") == "mp3"
}

func matchMP4(data []byte, _ io.ReadSeeker) (matched bool) {
	if len(data) < 12 {
		return
	}

	boxSize := int(binary.BigEndian.Uint32(data[:4]))
	nope := boxSize%4 != 0 ||
		len(data) < boxSize ||
		!bytes.Equal(data[4:8], []byte("ftyp"))
	if nope {
		return
	}

	for st := 8; st < boxSize; st += 4 {
		if st == 12 {
			// minor version number
			continue
		}
		if bytes.Equal(data[st:st+3], []byte("mp4")) {
			return true
		}
	}
	return
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

// DetectMIME detects the MIME type of rs.
// accepted, if not nil, specifies MIME types to not reject with
// ErrUnsupportedMIME.
// Returns mime type, most common file extension and any error.
func DetectMIME(rs io.ReadSeeker, accepted map[string]bool) (
	mime string, ext string, err error,
) {
	_, err = rs.Seek(0, 0)
	if err != nil {
		return
	}

	// Read up to first 512 bytes
	buf := make([]byte, sniffSize)
	read, err := rs.Read(buf)
	switch err {
	case nil, io.EOF:
		// NOOP, if 512 bytes were read. Otherwise - shortens the buffer.
		buf = buf[:read]
	default:
		return
	}

	for _, m := range matchers {
		if (accepted == nil || accepted[m.MIME()]) && m.Match(buf, rs) {
			mime = m.MIME()
			ext = m.Ext()
			return
		}
	}

	err = ErrUnsupportedMIME
	return
}

func processFile(src *Source, opts Options) (thumb Thumbnail, err error) {
	src.Mime, src.Extension, err = DetectMIME(src.Data, opts.AcceptedMimeTypes)
	if err != nil {
		return
	}

	if override := overrideProcessors[src.Mime]; override != nil {
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
	case // Audio is treated like a video container with only an audio stream
		"audio/mpeg",
		"audio/aac",
		"audio/wave",
		"audio/x-flac",
		"audio/midi",
		"application/ogg",
		"video/webm",
		"video/x-matroska",
		"video/avi",
		"video/mp4",
		"video/quicktime",
		"video/x-ms-wmv",
		"video/x-flv":
		return processVideo(src, opts)
	default:
		err = ErrUnsupportedMIME
		return
	}
}
