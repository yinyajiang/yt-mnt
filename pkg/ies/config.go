package ies

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type IEKeys map[string]string

type ConfigSt struct {
	Tokens IEKeys `toml:"tokens" json:"tokens"`
}

var Cfg ConfigSt

func init() {
	var err error
	for _, file := range []string{"config.toml", "config.json", "conf.json"} {
		if !isExist(file) {
			continue
		}
		switch strings.ToLower(filepath.Ext(file)) {
		case ".toml":
			err = toml.Unmarshal(readAll(file), &Cfg)
		case ".json":
			err = json.Unmarshal(readAll(file), &Cfg)
		}
	}
	if err != nil {
		log.Fatal(err)
	}
}

func isExist(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

func readAll(file string) []byte {
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	by, err := io.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	return by
}
