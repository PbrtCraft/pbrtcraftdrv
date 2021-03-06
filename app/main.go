package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"

	"github.com/PbrtCraft/pbrtcraftdrv/filetree"
	"github.com/PbrtCraft/pbrtcraftdrv/mcwdrv"
)

var mcwDriver *mcwdrv.MCWDriver
var appconf *appConfig
var srv http.Server

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
		Width       int            `json:"width,string"`
		Height      int            `json:"height,string"`
		Sample      int            `json:"sample,string"`
		Radius      int            `json:"radius,string"`
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
		Sample:      t.Sample,
		Radius:      t.Radius,
		Method:      t.Method,
		Camera:      t.Camera,
		Phenomenons: t.Phenomenons,
	}
	rc.Resolution.Width = t.Width
	rc.Resolution.Height = t.Height

	log.Println("PATH:", t.World)

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
	ft, err := filetree.GetFolder(path.Join(appconf.MWCDriver.Workdir, "scenes"))
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
	var tmp struct {
		DriverStatus string      `json:"driver_status"`
		Body         interface{} `json:"body"`
	}
	switch mcwDriver.GetStatus() {
	case mcwdrv.StatusIdle:
		tmp.DriverStatus = "idle"
	case mcwdrv.StatusReady:
		tmp.DriverStatus = "ready"
	case mcwdrv.StatusMc2pbrt:
		tmp.DriverStatus = "mc2pbrt"
	case mcwdrv.StatusPbrt:
		tmp.DriverStatus = "pbrt"
		tmp.Body = mcwDriver.GetPbrtStatus()
	}

	bs, err := json.Marshal(tmp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, string(bs))
}

func imgHandler(w http.ResponseWriter, r *http.Request) {
	imgBase64, err := mcwDriver.GetImageBase64()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, imgBase64)
}

func closeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Bye bye"))
	srv.Shutdown(context.TODO())
}

func main() {
	var err error
	appconfFilenamePtr := flag.String("appconf", "appconfig.yaml", "App Config filename")
	flag.Parse()

	log.Println("Start init app config...")
	appconf, err = getAppConfig(*appconfFilenamePtr)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Start init app config...DONE")

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
	mcwDriver, err = mcwdrv.NewMCWDriver(appconf.MWCDriver)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Start init mc driver...DONE")

	log.Println("Start init server...")

	mux := http.NewServeMux()
	mux.HandleFunc("/", mainHandler)
	mux.HandleFunc("/result", resultHandler)
	mux.HandleFunc("/files", filesHandler)

	mux.HandleFunc("/getstatus", statusHandler)
	mux.HandleFunc("/getimg", imgHandler)
	mux.HandleFunc("/gettype", typesHandler)
	mux.HandleFunc("/getworld", worldsHandler)
	mux.HandleFunc("/getfiles", getfilesHandler)
	mux.HandleFunc("/render", renderHandler)
	mux.HandleFunc("/stop", stopHandler)
	mux.HandleFunc("/close", closeHandler)

	mux.HandleFunc("/log", logHandler)
	mux.HandleFunc("/log/list", listLogHandler)
	mux.HandleFunc("/log/get", getLogHandler)
	mux.HandleFunc("/log/delete", deleteLogHandler)

	fsStatic := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fsStatic))

	fsScenes := http.FileServer(http.Dir(path.Join(appconf.MWCDriver.Workdir, "scenes")))
	mux.Handle("/scenes/", http.StripPrefix("/scenes/", fsScenes))
	log.Println("Start init server...DONE")

	port := appconf.Srv.Port
	log.Printf("Start listen at :%s...", port)
	srv = http.Server{Addr: ":" + port, Handler: mux}
	err = srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Println(err)
	}
}
