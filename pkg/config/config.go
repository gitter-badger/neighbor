package config

import (
	// stdlib
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	// external
	"github.com/golang/glog"
	// internal
)

// Contents contains the contents of the parsed config file.
type Contents struct {
	AccessToken string `json:"access_token"`
	SearchType  string `json:"search_type"`
	Query       string `json:"query"`

	ExternalCmdStr string `json:"external_command"`
}

// Config specifies information about the config file used for performing the experiment.
type Config struct {
	FilePath string
	Contents *Contents
}

// New is a constructor that returns a pointer to a Config object.
func New(fp string) *Config {
	return &Config{
		FilePath: fp,
	}
}

// Parse opens a file and calls a private `parse` method.
func (cfg *Config) Parse() {
	f, err := os.Open(cfg.FilePath)
	if err != nil {
		glog.Errorf("error opening config file %+v", err)
		return
	}
	defer f.Close()

	c := &Contents{}
	if err := parse(f, c); err != nil {
		glog.Errorf("error parsing config file %+v", err)
		return
	}

	cfg.Contents = c
	return
}

func parse(f io.Reader, d interface{}) error {
	b, err := ioutil.ReadAll(f)

	if err = json.Unmarshal(b, d); err != nil {
		return err
	}

	return nil
}
