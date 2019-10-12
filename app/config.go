package main

import (
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

type config struct {
	Path struct {
		Workdir     string `yaml:"workdir"`      // Path to workdir
		Mc2pbrtMain string `yaml:"mc2pbrt_main"` // Path to mc2pbrt/main.py
		PbrtBin     string `yaml:"pbrt_bin"`     // Path to pbrt binary
	} `yaml:"path"`

	PythonFile typeFile `yaml:"python_file"`

	Minecraft struct {
		Directory string `yaml:"directory"`
	} `yaml:"minecraft"`
}

func getConfig(filename string) (*config, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("app.getConfig: %s", err)
	}

	var c config
	// Default values
	c.Path.Workdir = "../../workdir/"
	c.Path.Mc2pbrtMain = "../../mc2pbrt/mc2pbrt/main.py"
	c.Path.PbrtBin = "../../pbrt-v3-minecraft/build/pbrt"
	err = yaml.Unmarshal(bytes, &c)
	if err != nil {
		return nil, fmt.Errorf("app.getConfig: %s", err)
	}
	return &c, nil
}
