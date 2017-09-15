package thumbnailer

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"time"
)

var (
	// ErrNoStreams denotes no decodeable audio or video streams were found in
	// a media container
	ErrNoStreams = errors.New("no decodeable video or audio streams found")

	// ErrNoThumb denotes a thumbnail can not be generated for this file.
	// Example: audio file with no cover art.
	ErrNoThumb = errors.New("no thumbnail can be generated")
)

type mediaInfo struct {
	// Container information
	Format struct {
		FormatName string   `json:"format_name"`
		Duration   duration `json:"duration"`
		Meta       struct {
			Title  string `json:"title"`
			Artist string `json:"artist"`
		} `json:"tags"`
	}

	// Streams detected in the container
	Streams []struct {
		CodecName string `json:"codec_name"`
		CodecType string `json:"codec_type"`
		Width     uint64 `json:"width"`
		Height    uint64 `json:"height"`
	} `json:"streams"`
}

// Parses FFMPEG duration stings
type duration time.Duration

func (d *duration) UnmarshalJSON(data []byte) error {
	if len(data) < 3 {
		return nil
	}
	data = data[1 : len(data)-1] // Strip quotes

	f, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return err
	}

	*d = duration(time.Duration(f * float64(time.Second)))
	return nil
}

// Returns media file information
func getMediaInfo(src *Source) (err error) {
	var (
		info mediaInfo
		args = make([]string, 0, 16)
	)

	// OGG does not support duration detection without seeking
	if src.Mime == "application/ogg" {
		var tmp string
		tmp, err = dumpToTemp(src.data)
		if err != nil {
			return
		}
		defer os.Remove(tmp)
		args = append(args, tmp)
	} else {
		args = append(args, "-")
	}

	args = append(
		args,
		"-hide_banner",
		"-of", "json=c=1",
		"-show_entries", "format=format_name,duration:stream=codec_name,codec_type,width,height",
	)

	buf, err := execCommand(src.data, "ffprobe", args...)
	if err != nil {
		return
	}
	defer PutBuffer(buf)

	err = json.Unmarshal(buf.Bytes(), &info)
	if err != nil {
		return
	}

	src.Length = time.Duration(info.Format.Duration)

	// Detect any audio stream, video stream and/or cover art
	for _, s := range info.Streams {
		switch s.CodecType {
		case "audio":
			src.HasAudio = true
		case "video":
			// Detect dimensions to skip furher checks by the thumbnailer
			if src.Width == 0 && s.Width != 0 {
				src.Width = s.Width
			}
			if src.Height == 0 && s.Height != 0 {
				src.Height = s.Height
			}

			switch s.CodecName {
			// Cover art counts as a video stream
			case "png", "jpeg", "gif":
				src.HasCoverArt = true
			default:
				src.HasVideo = true
			}
		}
	}

	return
}

func processVideo(src *Source, opts Options) (thumb Thumbnail, err error) {
	err = getMediaInfo(src)
	if err != nil {
		return
	}
	err = handleDims(src, &thumb, opts)
	if err != nil {
		return
	}

	// MP4 and its offspiring need input seeking and thus can not be piped in.
	// Write a temp file to disk.
	var tmp string
	switch src.Mime {
	case "video/mp4", "video/quicktime":
		tmp, err = dumpToTemp(src.data)
		if err != nil {
			return
		}
		defer os.Remove(tmp)
	}

	// TODO
	// c.ExtractMeta(&src)

	args := append(make([]string, 0, 16), "-i")
	if tmp != "" {
		args = append(args, tmp)
	} else {
		args = append(args, "-")
	}
	args = append(
		args,
		"-hide_banner",
		"-an", "-sn",
		"-frames:v", "1",
		"-f", "image2",
	)
	switch {
	case src.HasCoverArt:
	case src.HasVideo:
		args = append(args, "-vf", "thumbnail")
	default:
		// As of writing ffmpeg does not support cover art in neither MP4-like
		// containers or OGG, so consider these unthumbnailable
		if src.HasAudio {
			err = ErrNoThumb
		} else {
			err = ErrNoStreams
		}
		return
	}
	args = append(args, "-")

	pipe := pipeLine{
		command("ffmpeg", args...),
		genThumb(src, &thumb, opts)[0],
	}
	thumb.Data, err = pipe.Exec(src.data)
	return
}

// Dump data to a temp file on disk.
// The caller is responsible for removing the file.
func dumpToTemp(rs io.ReadSeeker) (name string, err error) {
	_, err = rs.Seek(0, 0)
	if err != nil {
		return
	}

	tmp, err := ioutil.TempFile("", "thumbnailer-")
	if err != nil {
		return
	}
	defer tmp.Close()

	_, err = io.Copy(tmp, rs)
	if err != nil {
		return
	}

	return tmp.Name(), nil
}
