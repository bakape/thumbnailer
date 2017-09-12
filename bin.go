package thumbnailer

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// Fetches a buffer from the internal pool.
// Use this, to reduce big reallocations on your side of the application.
func GetBuffer() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

// Returns a buffer to the pool to be reused or deallocated
func PutBuffer(b *bytes.Buffer) {
	b.Reset()
	bufPool.Put(b)
}

// Same as PutBuffer, but for []byte
func PutBytes(b []byte) {
	bufPool.Put(bytes.NewBuffer(b))
}

var binPaths = make(map[string]string, 4)

func init() {
	// For looking up binary absolute paths
	binPaths["which"] = "/usr/bin/which"
	if runtime.GOOS == "windows" {
		binPaths["which"] = os.Getenv("SYSTEMROOT") + `\where.exe`
	}

	for _, b := range [...]string{"ffmpeg", "ffprobe", "gm", "pngquant"} {
		findBin(b)
	}
}

// Find and store executable binary path, if any
func findBin(name string) {
	bin := name
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	path := filepath.Join("bin", bin)
	_, err := os.Stat(path) // Check for local copy
	if err != nil {
		// Scan system paths
		var buf []byte
		buf, err = command("which", bin).Output()
		if err != nil {
			log.Fatalf("executable not found: %s\n", bin)
		}
		path = strings.Split(string(buf), "\n")[0]
	}
	binPaths[name] = path
}

// Helper for piping output of one child process into another
type pipeLine []*exec.Cmd

// Execute pipeLine with passed input data. Returns the final buffered stdout
func (p pipeLine) Exec(rs io.ReadSeeker) (out []byte, err error) {
	_, err = rs.Seek(0, 0)
	if err != nil {
		return
	}

	var (
		stdIn  *bytes.Buffer
		stdOut = GetBuffer()
		stdErr bytes.Buffer
		first  = true
	)

	for _, c := range p {
		if first {
			c.Stdin = rs
		} else {
			c.Stdin = stdIn
		}
		c.Stdout = stdOut
		c.Stderr = &stdErr

		err = c.Run()
		if err != nil {
			formatFFMPEGError(&err, &stdErr)
			return
		}

		stdErr.Reset()
		if first {
			stdIn = stdOut
			stdOut = GetBuffer()
			defer PutBuffer(stdOut)
			first = false
		} else {
			stdIn.Reset()
			stdIn, stdOut = stdOut, stdIn // Reuse last buffer by swapping
		}
	}

	return stdIn.Bytes(), nil
}

func formatFFMPEGError(err *error, stdErr *bytes.Buffer) {
	*err = fmt.Errorf("%s: %s", *err, strings.TrimSpace(stdErr.String()))
}

// Helper for constructing child processes
func command(bin string, args ...string) *exec.Cmd {
	return &exec.Cmd{
		Path: binPaths[bin],
		Args: append([]string{bin}, args...),
	}
}

// Execute command with given data as stdin.
// Returns buffered output. Use PutBuffer() to return it back to the pool.
func execCommand(rs io.ReadSeeker, bin string, args ...string) (
	*bytes.Buffer, error,
) {
	if _, err := rs.Seek(0, 0); err != nil {
		return nil, err
	}

	var (
		stdOut = GetBuffer()
		stdErr bytes.Buffer
	)

	cmd := command(bin, args...)
	cmd.Stdin = rs
	cmd.Stderr = &stdErr
	cmd.Stdout = stdOut
	if err := cmd.Run(); err != nil {
		PutBuffer(stdOut)
		formatFFMPEGError(&err, &stdErr)
		return nil, err
	}
	return stdOut, nil
}
