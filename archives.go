package thumbnailer

import (
	"archive/zip"
	"bytes"
	"image"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

const (
	mimeZip  = "application/zip"
	mime7Zip = "application/x-7z-compressed"
	mimeRar  = "application/x-rar-compressed"
)

// Thumbnail the first image of a zip file
func processZip(rs io.ReadSeeker, src *Source, opts Options,
) (thumb image.Image, err error) {
	// Obtain io.ReaderAt and find out the size of the file
	var (
		size int64
		ra   io.ReaderAt
	)

	useFile := func(f *os.File) (err error) {
		info, err := f.Stat()
		if err != nil {
			return
		}
		size = info.Size()
		ra = f
		return
	}

	switch rs.(type) {
	case *os.File:
		err = useFile(rs.(*os.File))
		if err != nil {
			return
		}
	case *bytes.Reader:
		r := rs.(*bytes.Reader)
		ra = r
		size = r.Size()
	default:
		// Dump exotic io.ReadSeeker to file and use that
		var tmp *os.File
		tmp, err = ioutil.TempFile("", "")
		if err != nil {
			return
		}
		defer os.Remove(tmp.Name())
		defer tmp.Close()

		_, err = io.Copy(tmp, rs)
		if err != nil {
			return
		}
		err = useFile(tmp)
		if err != nil {
			return
		}
	}

	r, err := zip.NewReader(ra, size)
	if err != nil {
		return
	}

	imageCount := 0
	var firstImage *zip.File
	for _, f := range r.File {
		for _, ext := range [...]string{".png", ".jpg", ".jpeg", ".webp"} {
			if strings.HasSuffix(f.Name, ext) {
				if firstImage == nil {
					firstImage = f
				}
				imageCount++
				break
			}
		}
	}

	// If at least 90% of files in the archive root are images, this is a comic
	// archive
	if float32(imageCount)/float32(len(r.File)) >= 0.9 {
		src.Mime = "application/vnd.comicbook+zip"
		src.Extension = "cbz"
	}

	if firstImage == nil {
		err = ErrCantThumbnail
		return
	}

	err = func() (err error) {
		// Accept anything we can process
		opts := opts
		opts.AcceptedMimeTypes = nil

		f, err := firstImage.Open()
		if err != nil {
			return
		}
		defer f.Close()

		// zip.File does not provide seeking.
		// Temporary file to conserve RAM.
		tmp, err := ioutil.TempFile("", "")
		if err != nil {
			return
		}
		defer os.Remove(tmp.Name())
		defer tmp.Close()

		_, err = io.Copy(tmp, f)
		if err != nil {
			return
		}

		_, thumb, err = Process(tmp, opts)
		return
	}()
	if err != nil {
		err = ErrArchive{err}
	}
	return
}
