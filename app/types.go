package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/PbrtCraft/pbrtcraftdrv/parsepy"
)

type typeFile struct {
	Camera     string `yaml:"camera"`     // Path to camera.py
	Phenomenon string `yaml:"phenomenon"` // Path to phenomenon.py
	Method     string `yaml:"method"`     // Path to method.py
}

var (
	cameraTypeList     []*parsepy.Class
	phenomenonTypeList []*parsepy.Class
	methodTypeList     []*parsepy.Class
)

func initTypes(tf typeFile) error {
	var err error
	cameraTypeList, err = parsepy.GetClasses(tf.Camera)
	if err != nil {
		return fmt.Errorf("app.main.initTypes: %s", err)
	}

	phenomenonTypeList, err = parsepy.GetClasses(tf.Phenomenon)
	if err != nil {
		return fmt.Errorf("app.main.initTypes: %s", err)
	}

	methodTypeList, err = parsepy.GetClasses(tf.Method)
	if err != nil {
		return fmt.Errorf("app.main.initTypes: %s", err)
	}

	return nil
}

func typesHandler(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(map[string]interface{}{
		"camera":     cameraTypeList,
		"phenomenon": phenomenonTypeList,
		"method":     methodTypeList,
	}, "", " ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Fprintf(w, string(bytes))
}
