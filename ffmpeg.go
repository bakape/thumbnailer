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

	*d = duration(time.Duration(f) * time.Second)
	return nil
}

// Returns media file information
func getMediaInfo(rs io.ReadSeeker) (info mediaInfo, err error) {
	buf, err := execCommand(
		rs,
		"ffprobe",
		"-",
		"-hide_banner",
		"-of", "json=c=1",
		"-show_entries", "format=format_name,duration:stream=codec_name,codec_type,width,height",
	)
	if err != nil {
		return
	}
	defer PutBuffer(buf)

	err = json.Unmarshal(buf.Bytes(), &info)
	return
}

func processVideo(src *Source, opts Options) (thumb Thumbnail, err error) {
	info, err := getMediaInfo(src.data)
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

	err = handleDims(src, &thumb, opts)
	if err != nil {
		return
	}

	// MP4 and its offspiring need input seeking and thus can not be piped in.
	// Write a temp file to disk
	var (
		tmp   *os.File
		isMP4 bool
	)
	switch src.Mime {
	case "video/mp4", "video/quicktime":
		isMP4 = true

		_, err = src.data.Seek(0, 0)
		if err != nil {
			return
		}

		tmp, err = ioutil.TempFile("", "thumbnailer-")
		if err != nil {
			return
		}
		defer tmp.Close()
		defer os.Remove(tmp.Name())

		_, err = io.Copy(tmp, src.data)
		if err != nil {
			return
		}
	}

	// TODO
	// c.ExtractMeta(&src)

	args := append(make([]string, 0, 16), "-i")
	if isMP4 {
		args = append(args, tmp.Name())
	} else {
		args = append(args, "-")
	}
	args = append(
		args,
		"-hide_banner",
		"-an", "-sn",
		"-frames:v", "1",
		"-f", "image2", // May have transparency, so always output PNG
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
