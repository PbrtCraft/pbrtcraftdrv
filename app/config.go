package main

import (
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

type appConfig struct {
	Path struct {
		Workdir     string `yaml:"workdir"`      // Path to workdir
		Mc2pbrtMain string `yaml:"mc2pbrt_main"` // Path to mc2pbrt/main.py
		PbrtBin     string `yaml:"pbrt_bin"`     // Path to pbrt binary
		LogDir      string `yaml:"log_dir"`      // mcwdrv log directory
	} `yaml:"path"`

	PythonFile typeFile `yaml:"python_file"`

	Minecraft struct {
		Directory string `yaml:"directory"`
	} `yaml:"minecraft"`
}

type srvConfig struct {
	Port string `json:"port"`
}

func getAppConfig(filename string) (*appConfig, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("app.getAppConfig: %s", err)
	}

	var c appConfig
	err = yaml.Unmarshal(bytes, &c)
	if err != nil {
		return nil, fmt.Errorf("app.getAppConfig: %s", err)
	}
	return &c, nil
}

func getSrvConfig(filename string) (*srvConfig, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("app.getSrvConfig: %s", err)
	}

	var c srvConfig
	err = yaml.Unmarshal(bytes, &c)
	if err != nil {
		return nil, fmt.Errorf("app.getSrvConfig: %s", err)
	}
	return &c, nil
}
