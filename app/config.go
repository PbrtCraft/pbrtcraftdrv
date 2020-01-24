package main

import (
	"fmt"
	"io/ioutil"

	"github.com/PbrtCraft/pbrtcraftdrv/mcwdrv"
	yaml "gopkg.in/yaml.v2"
)

type appConfig struct {
	MWCDriver *mcwdrv.Config `yaml:"mcw_driver"`

	PythonFile typeFile `yaml:"python_file"`

	Minecraft struct {
		Directory string `yaml:"directory"`
	} `yaml:"minecraft"`

	Srv struct {
		Port string `yaml:"port"`
	} `yaml:"srv"`
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
