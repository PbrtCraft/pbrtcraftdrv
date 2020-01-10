package mc

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
)

// World stores data of minecraft world
type World struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Icon    string   `json:"icon"`
	Players []string `json:"players"`
}

// NewWorld return new minecraft world info
func NewWorld(dir string) (*World, error) {
	players, err := listUsers(dir)
	if err != nil {
		return nil, fmt.Errorf("app.main.NewWorld: %s", err)
	}

	iconBase64 := ""
	bytes, err := ioutil.ReadFile(path.Join(dir, "icon.png"))
	if err != nil {
		iconBase64 = ""
	}
	iconBase64 = base64.StdEncoding.EncodeToString(bytes)

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("app.main.NewWorld: %s", err)
	}

	return &World{
		Players: players,
		Name:    filepath.Base(dir),
		Icon:    iconBase64,
		Path:    absDir,
	}, nil
}

func listUsers(mcDir string) ([]string, error) {
	files, err := ioutil.ReadDir(path.Join(mcDir, "playerdata"))
	if err != nil {
		return nil, fmt.Errorf("app.listUsers: %s", err)
	}

	users := []string{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fn := file.Name()
		playerName, err := uuidToName(fn[0 : len(fn)-4])
		if err != nil {
			log.Println(err)
			continue
		}
		users = append(users, playerName)
	}

	return users, nil
}
