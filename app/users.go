package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"

	"github.com/PbrtCraft/pbrtcraftdrv/mc"
)

var worlds []*mc.World

func initSingleWorld(mcDir string) error {
	world, err := mc.NewWorld(mcDir)
	if err != nil {
		return fmt.Errorf("app.main.initSingleWorld: %s", err)
	}
	worlds = []*mc.World{world}
	return nil
}

func initClientWorlds() error {
	mcWorldsDir, err := mc.FindMinecraft()
	if err != nil {
		return fmt.Errorf("app.main.initClientWorlds: %s", err)
	}

	worldFolders, err := listWorlds(mcWorldsDir)
	if err != nil {
		return fmt.Errorf("app.main.initClientWorlds: %s", err)
	}

	worlds = nil
	for _, folder := range worldFolders {
		mcDir := path.Join(mcWorldsDir, "saves", folder)
		world, err := mc.NewWorld(mcDir)
		if err != nil {
			log.Println("Read World:", folder, err)
			continue
		}
		worlds = append(worlds, world)
	}
	log.Println("Get", len(worlds), "world(s)")
	return nil
}

func worldsHandler(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.Marshal(worlds)
	if err != nil {
		log.Println(err)
		fmt.Fprint(w, "[]")
		return
	}

	fmt.Fprint(w, string(bytes))
}

func listWorlds(mcDir string) ([]string, error) {
	saveDir := path.Join(mcDir, "saves")
	folders, err := ioutil.ReadDir(saveDir)
	if err != nil {
		return nil, fmt.Errorf("app.listWorlds: %s", err)
	}
	ret := []string{}
	for _, folder := range folders {
		if folder.IsDir() {
			ret = append(ret, folder.Name())
		}
	}
	return ret, nil
}
