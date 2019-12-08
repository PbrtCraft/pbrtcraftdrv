package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
	"strconv"

	"github.com/PbrtCraft/pbrtcraftdrv/filetree"
	"github.com/PbrtCraft/pbrtcraftdrv/mcwdrv"
)

var mcwDriver *mcwdrv.MCWDriver
var appconf *appConfig
var srvconf *srvConfig

func mainHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("").Delims("[[", "]]").
		ParseFiles("template/basic.html", "template/index.html")
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "basic", nil)
}

func resultHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("").Delims("[[", "]]").
		ParseFiles("template/basic.html", "template/result.html")

	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	imgBase64, err := mcwDriver.GetImageBase64()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err != nil {
		log.Println(err)
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
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "basic", nil)
}

func renderHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var t struct {
		World       string         `json:"world"`
		Player      string         `json:"player"`
		Sample      string         `json:"sample"`
		Radius      string         `json:"radius"`
		Method      mcwdrv.Class   `json:"method"`
		Camera      mcwdrv.Class   `json:"camera"`
		Phenomenons []mcwdrv.Class `json:"phenomenons"`
	}
	err := decoder.Decode(&t)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rc := mcwdrv.RenderConfig{
		World:       t.World,
		Player:      t.Player,
		Method:      t.Method,
		Camera:      t.Camera,
		Phenomenons: t.Phenomenons,
	}

	log.Println("PATH:", t.World)

	rc.Sample, err = strconv.Atoi(t.Sample)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rc.Radius, err = strconv.Atoi(t.Radius)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = mcwDriver.Compile(rc)
	if err != nil {
		log.Println(err)
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
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(ft)
	if err != nil {
		log.Println(err)
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
	appconfFilenamePtr := flag.String("appconf", "appconfig.yaml", "App Config filename")
	srvconfFilenamePtr := flag.String("srvconf", "srvconfig.yaml", "Server Config filename")
	flag.Parse()

	log.Println("Start init app config...")
	appconf, err = getAppConfig(*appconfFilenamePtr)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Start init app config...DONE")

	log.Println("Start init srv config...")
	srvconf, err = getSrvConfig(*srvconfFilenamePtr)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Start init srv config...DONE")

	log.Println("Start init srv worlds....")
	if appconf.Minecraft.Directory == "" {
		log.Println("init client worlds...")
		err = initClientWorlds()
	} else {
		log.Println("init single world...")
		err = initSingleWorld(appconf.Minecraft.Directory)
	}
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Start init srv worlds...DONE")

	log.Println("Start reading python types...")
	err = initTypes(appconf.PythonFile)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Start reading python types...DONE")

	log.Println("Start init mc driver...")
	mcwDriver, err = mcwdrv.NewMCWDriver(
		appconf.Path.Workdir,
		appconf.Path.Mc2pbrtMain,
		appconf.Path.PbrtBin,
		appconf.Path.LogDir,
	)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Start init mc driver...DONE")

	log.Println("Start init server...")
	http.HandleFunc("/", mainHandler)
	http.HandleFunc("/result", resultHandler)
	http.HandleFunc("/files", filesHandler)

	http.HandleFunc("/getstatus", statusHandler)
	http.HandleFunc("/getimg", imgHandler)
	http.HandleFunc("/gettype", typesHandler)
	http.HandleFunc("/getworld", worldsHandler)
	http.HandleFunc("/getfiles", getfilesHandler)
	http.HandleFunc("/render", renderHandler)
	http.HandleFunc("/stop", stopHandler)

	http.HandleFunc("/log", logHandler)
	http.HandleFunc("/log/list", listLogHandler)
	http.HandleFunc("/log/get", getLogHandler)
	http.HandleFunc("/log/delete", deleteLogHandler)

	fsStatic := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fsStatic))

	fsScenes := http.FileServer(http.Dir(path.Join(appconf.Path.Workdir, "scenes")))
	http.Handle("/scenes/", http.StripPrefix("/scenes/", fsScenes))
	log.Println("Start init server...DONE")

	log.Printf("Start listen at :%s...", srvconf.Port)
	http.ListenAndServe(":"+srvconf.Port, nil)
}
