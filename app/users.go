package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	uuidNameMap map[string]string
)

func init() {
	if err := readUUIDCache(); err != nil {
		uuidNameMap = map[string]string{}
	}
}

func readUUIDCache() error {
	bs, err := ioutil.ReadFile("usercache.json")
	if err != nil {
		return fmt.Errorf("app.main.readUUIDCache: %s", err)
	}
	err = json.Unmarshal(bs, &uuidNameMap)
	if err != nil {
		return fmt.Errorf("app.main.readUUIDCache: %s", err)
	}
	return nil
}

func writeUUIDCache() error {
	bs, err := json.Marshal(uuidNameMap)
	if err != nil {
		return fmt.Errorf("app.main.writeUUIDCache: %s", err)
	}
	err = ioutil.WriteFile("usercache.json", bs, 0666)
	if err != nil {
		return fmt.Errorf("app.main.writeUUIDCache: %s", err)
	}
	return nil
}

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

var worlds []*World

func initSingleWorld(mcDir string) error {
	world, err := NewWorld(mcDir)
	if err != nil {
		return fmt.Errorf("app.main.initSingleWorld: %s", err)
	}
	worlds = []*World{world}
	return nil
}

func initClientWorlds() error {
	mcWorldsDir, err := findMinecraft()
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
		world, err := NewWorld(mcDir)
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

// ErrMinecraftClientNotFound returned when minecraft client folder not found
var ErrMinecraftClientNotFound = errors.New("Minecraft client folder not found")

func findMinecraft() (string, error) {
	var mcDir string
	if runtime.GOOS == "windows" {
		mcDir = path.Join(os.Getenv("APPDATA"), ".minecraft")
	} else if runtime.GOOS == "linux" {
		mcDir = path.Join(os.Getenv("HOME"), ".minecraft")
	} else if runtime.GOOS == "darwin" {
		mcDir = path.Join(os.Getenv("HOME"), "Library", "Application Support", "minecraft")
	}

	if mcDir == "" {
		return "", ErrMinecraftClientNotFound
	}

	if _, err := os.Stat(mcDir); os.IsNotExist(err) {
		return "", ErrMinecraftClientNotFound
	}
	return mcDir, nil
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

func uuidToName(uuid string) (string, error) {
	if name, exist := uuidNameMap[uuid]; exist {
		return name, nil
	}

	url := fmt.Sprintf("https://api.mojang.com/user/profiles/%s/names",
		strings.Replace(uuid, "-", "", -1))
	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("app.uuidToName: %s", err)
	}
	req.Header.Set("User-Agent", "Get Name By UUID")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("app.uuidToName: %s", err)
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("app.uuidToName: %s", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("app.uuidToName: Request StatusCode = %d, %s",
			resp.StatusCode, string(bytes))
	}

	var tmp []struct {
		Name string `json:"name"`
	}
	err = json.Unmarshal(bytes, &tmp)
	if err != nil {
		return "", fmt.Errorf("app.uuidToName: %s", err)
	}
	if len(tmp) == 0 {
		return "", fmt.Errorf("app.uuidToName: cannot found player by uuid %s", uuid)
	}

	uuidNameMap[uuid] = tmp[0].Name

	err = writeUUIDCache()
	if err != nil {
		log.Println(err)
	}

	return tmp[0].Name, nil
}
