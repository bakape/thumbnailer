package thumbnailer

import "testing"

func TestMetadataExtraction(t *testing.T) {
	t.Parallel()

	f := openSample(t, "title.mp3")
	defer f.Close()

	src, _, err := Process(f, Options{})
	if err != nil && err != ErrCantThumbnail {
		t.Fatal(err)
	}
	if src.Artist != "Test Artist" {
		t.Errorf("unexpected artist: Test Artist : %s", src.Artist)
	}
	if src.Title != "Test Title" {
		t.Errorf("unexpected title: Test Title: %s", src.Title)
	}
}
