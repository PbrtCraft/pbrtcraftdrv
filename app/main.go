package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/PbrtCraft/pbrtcraftdrv/filetree"
	"github.com/PbrtCraft/pbrtcraftdrv/mcwdrv"
)

var mcwDriver *mcwdrv.MCWDriver
var appconf *config

func mainHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("").Delims("[[", "]]").
		ParseFiles("template/basic.html", "template/index.html")
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "basic", nil)
}

func resultHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("").Delims("[[", "]]").
		ParseFiles("template/basic.html", "template/result.html")

	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	imgBase64, err := mcwDriver.GetImageBase64()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data := struct {
		Pano   bool
		ImgSrc template.URL
	}{
		Pano:   false,
		ImgSrc: template.URL("data:image/jpeg;base64," + imgBase64),
	}

	tmpl.ExecuteTemplate(w, "basic", data)
}

func filesHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("").Delims("[[", "]]").
		ParseFiles("template/basic.html", "template/files.html")
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "basic", nil)
}

func renderHandler(w http.ResponseWriter, r *http.Request) {
	rc := mcwdrv.RenderConfig{
		World:  "/home/mukyu99/Minecraft/world",
		Player: "Mudream",
	}
	var err error
	decoder := json.NewDecoder(r.Body)
	var t struct {
		Sample      string         `json:"sample"`
		Radius      string         `json:"radius"`
		Method      mcwdrv.Class   `json:"method"`
		Camera      mcwdrv.Class   `json:"camera"`
		Phenomenons []mcwdrv.Class `json:"phenomenons"`
	}
	err = decoder.Decode(&t)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	rc.Sample, err = strconv.Atoi(t.Sample)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rc.Radius, err = strconv.Atoi(t.Radius)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	rc.Method = t.Method
	rc.Camera = t.Camera
	rc.Phenomenons = t.Phenomenons

	err = mcwDriver.Compile(rc)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	mcwDriver.StopCompile()
}

func getfilesHandler(w http.ResponseWriter, r *http.Request) {
	ft, err := filetree.GetFolder(path.Join(appconf.Path.Workdir, "scenes"))
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(ft)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, string(bytes))
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	str := ""
	switch mcwDriver.GetStatus() {
	case mcwdrv.StatusIdle:
		str = "idle"
	case mcwdrv.StatusReady:
		str = "ready"
	case mcwdrv.StatusMc2pbrt:
		str = "mc2pbrt"
	case mcwdrv.StatusPbrt:
		str = "pbrt"
	}
	fmt.Fprintf(w, str)
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := listUsers()
	if err != nil {
		log.Println(err)
		fmt.Fprint(w, "[]")
		return
	}

	bytes, err := json.Marshal(users)
	if err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "[]")
		return
	}

	fmt.Fprint(w, string(bytes))
}

func listUsers() ([]string, error) {
	files, err := ioutil.ReadDir(path.Join(appconf.Minecraft.Directory, "world", "playerdata"))
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
			fmt.Println(err)
			continue
		}
		users = append(users, playerName)
	}

	return users, nil
}

func uuidToName(uuid string) (string, error) {
	url := fmt.Sprintf("https://api.mojang.com/user/profiles/%s/names",
		strings.Replace(uuid, "-", "", -1))
	client := http.Client{
		Timeout: time.Second * 2,
	}
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
	return tmp[0].Name, nil
}

func imgHandler(w http.ResponseWriter, r *http.Request) {
	imgBase64, err := mcwDriver.GetImageBase64()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, imgBase64)
}

func main() {
	var err error
	appconfFilenamePtr := flag.String("appconf", "appconfig.yaml", "Config filename")
	flag.Parse()

	appconf, err = getConfig(*appconfFilenamePtr)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = initTypes(appconf.PythonFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	mcwDriver, err = mcwdrv.NewMCWDriver(
		appconf.Path.Workdir,
		appconf.Path.Mc2pbrtMain,
		appconf.Path.PbrtBin,
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	http.HandleFunc("/", mainHandler)
	http.HandleFunc("/result", resultHandler)
	http.HandleFunc("/files", filesHandler)

	http.HandleFunc("/getstatus", statusHandler)
	http.HandleFunc("/getimg", imgHandler)
	http.HandleFunc("/gettype", typesHandler)
	http.HandleFunc("/getuser", usersHandler)
	http.HandleFunc("/getfiles", getfilesHandler)
	http.HandleFunc("/render", renderHandler)
	http.HandleFunc("/stop", stopHandler)

	fsStatic := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fsStatic))

	fsScenes := http.FileServer(http.Dir(path.Join(appconf.Path.Workdir, "scenes")))
	http.Handle("/scenes/", http.StripPrefix("/scenes/", fsScenes))

	http.ListenAndServe(":8080", nil)
}
