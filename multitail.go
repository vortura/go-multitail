package multitail

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/nxadm/tail"
)

type MultiTail struct {
	Lines chan *Line
	Config
}

type Config struct {
	ReadFromHead bool
}

type Line struct {
	Filename string    // The name of the file from which this line originates
	Text     string    // The contents of the file
	Num      int       // The line number
	SeekInfo SeekInfo  // SeekInfo
	Time     time.Time // Present time
	Err      error     // Error from tail
}

type SeekInfo struct {
	Offset int64
	Whence int
}

// OpenDirectory returns a Multitail for all files in the specified directory
// path
func OpenDirectory(path string, config Config) (*MultiTail, error) {
	isDir, err := isDirectory(path)
	if err != nil {
		return nil, err
	}
	if !isDir {
		return nil, errors.New("path is not a directory")
	}

	globPath := filepath.Join(path, "*")
	return OpenGlob(globPath, config)
}

// OpenGlob returns a Multitail for all files matching a specified glob pattern.
// See filepath.Glob for glob pattern syntax.
func OpenGlob(glob string, config Config) (*MultiTail, error) {
	multitail := newMultiTail(config)
	tailPaths, err := filepath.Glob(glob)
	if err != nil {
		return nil, err
	}
	if len(tailPaths) == 0 {
		return nil, fmt.Errorf("glob %s did not match any files", glob)
	}

	tails := make([]chan *tail.Line, len(tailPaths))
	for i, path := range tailPaths {
		t, err := tail.TailFile(path, getTailConfig(multitail))
		if err != nil {
			return nil, err
		}
		tails[i] = t.Lines
		go tailWorker(path, t.Lines, multitail.Lines)
	}

	return multitail, nil
}

func newMultiTail(config Config) *MultiTail {
	return &MultiTail{
		Config: config,
		Lines:  make(chan *Line),
	}
}

func getTailConfig(multitail *MultiTail) tail.Config {
	tailConfig := tail.Config{
		ReOpen: true,
		Follow: true,
	}

	// Read from end of file by default
	if !multitail.ReadFromHead {
		tailConfig.Location = &tail.SeekInfo{
			Whence: io.SeekEnd,
			Offset: 0,
		}
	}

	return tailConfig
}

func isDirectory(name string) (bool, error) {
	fileInfo, err := os.Stat(name)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), nil
}

// tailWorker consumes lines from input channel in, and writes them to output
// channel out
func tailWorker(filename string, in chan *tail.Line, out chan *Line) {
	for msg := range in {
		out <- &Line{
			Filename: filename,
			Text:     msg.Text,
			Num:      msg.Num,
			SeekInfo: SeekInfo(msg.SeekInfo),
			Time:     msg.Time,
			Err:      msg.Err,
		}
	}
}
