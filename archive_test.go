package thumbnailer

import (
	"bytes"
	"io"
	"io/ioutil"
	"sync"
	"testing"
)

type customReadSeeker struct {
	r bytes.Reader
}

func (c *customReadSeeker) Read(p []byte) (n int, err error) {
	return c.r.Read(p)
}

func (c *customReadSeeker) Seek(offset int64, whence int) (int64, error) {
	return c.r.Seek(offset, whence)
}

func TestArchiveReadSeekerTypes(t *testing.T) {
	var wg sync.WaitGroup

	file := openSample(t, "sample.zip")

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.Seek(0, 0)
	if err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name string
		rs   io.ReadSeeker
	}{
		{"file", file},
		{"bytes.Reader", bytes.NewReader(buf)},
		{"custom io.ReadSeeker", &customReadSeeker{*bytes.NewReader(buf)}},
	}

	for i := range cases {
		c := cases[i]
		wg.Add(1)
		t.Run(c.name, func(t *testing.T) {
			// t.Parallel()
			defer wg.Done()

			_, err := processZip(c.rs, &Source{}, Options{})
			if err != nil {
				t.Fatal(err)
			}
		})
	}

	go func() {
		wg.Wait()
		file.Close()
	}()
}
