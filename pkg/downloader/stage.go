package downloader

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type DownloaderStageSaver interface {
	Delete(name string) error
	Save(name, data string) error
	Read(name string) (string, error)
}

func NewLocalDirStageSaver(stageDir string) DownloaderStageSaver {
	return &LocalDirStageSaver{
		stageDir: stageDir,
	}
}

type LocalDirStageSaver struct {
	stageDir string
}

func (s *LocalDirStageSaver) Delete(name string) error {
	name = correctName(name)
	return os.Remove(filepath.Join(s.stageDir, name))
}

func (s *LocalDirStageSaver) Save(name, data string) (err error) {
	name = correctName(name)
	if err = os.MkdirAll(s.stageDir, 0755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(s.stageDir, name))

	if err != nil {
		return err

	}
	defer f.Close()
	_, err = f.WriteString(data)
	return
}

func (s *LocalDirStageSaver) Read(name string) (string, error) {
	name = correctName(name)
	f, err := os.Open(filepath.Join(s.stageDir, name))
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func correctName(name string) string {
	if !strings.HasPrefix(name, "hash:") {
		name = "hash:" + fmt.Sprintf("%x", md5.Sum([]byte(name)))
	}
	return name
}
