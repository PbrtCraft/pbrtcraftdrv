package mcwdrv

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MCWStatus record the status of MCWDriver
type MCWStatus int

const (
	// StatusIdle -> MCW is idle and can compile
	StatusIdle MCWStatus = iota

	// StatusReady -> MCW is config to ready for compiling
	StatusReady

	// StatusMc2pbrt -> MCW is running mc2pbrt
	StatusMc2pbrt

	// StatusPbrt -> MCW is running pbrt
	StatusPbrt
)

// MCWDriver manage mc2pbrt and pbrt to render a scene of minecraft
type MCWDriver struct {
	mutex      sync.Mutex
	status     MCWStatus
	cmdMc2pbrt *exec.Cmd
	cmdPbrt    *exec.Cmd

	lastCompile struct {
		err error
	}

	path struct {
		workdir     string
		mc2pbrtMain string
		pbrtBin     string
		logDir      string
	}
}

// Class is a type in Minecraft render config
type Class struct {
	Name   string      `json:"name"`
	Params interface{} `json:"params"`
}

// RenderConfig is Minecraft scene render config for mc2pbrt
// more info ref: https://github.com/PbrtCraft/mc2pbrt
type RenderConfig struct {
	World  string
	Player string
	Sample int
	Radius int

	Method      Class
	Camera      Class
	Phenomenons []Class
}

// ErrDriverNotIdel occur when try to compile when compiling
var ErrDriverNotIdel = fmt.Errorf("mcwdrv.Compile: Driver status not idle")

// NewMCWDriver return a minecraft world driver
func NewMCWDriver(workdir, mc2pbrtMain, pbrtBin, logDir string) (*MCWDriver, error) {
	ret := &MCWDriver{}
	var err error
	ret.path.mc2pbrtMain, err = filepath.Abs(mc2pbrtMain)
	if err != nil {
		return nil, fmt.Errorf("mcwdrv.NewMCWDriver: %s", err)
	}
	ret.path.pbrtBin, err = filepath.Abs(pbrtBin)
	if err != nil {
		return nil, fmt.Errorf("mcwdrv.NewMCWDriver: %s", err)
	}

	ret.path.workdir, err = filepath.Abs(workdir)
	if err != nil {
		return nil, fmt.Errorf("mcwdrv.NewMCWDriver: %s", err)
	}

	ret.path.logDir, err = filepath.Abs(logDir)
	if err != nil {
		return nil, fmt.Errorf("mcwdrv.NewMCWDriver: %s", err)
	}

	err = os.MkdirAll(ret.path.logDir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("mcwdrv.NewMCWDriver: %s", err)
	}

	return ret, nil
}

// Compile start a goroutine to generate pbrt file and render
func (drv *MCWDriver) Compile(rc RenderConfig) error {
	err := drv.writeRenderConfig(rc)
	if err != nil {
		return fmt.Errorf("mcwdrv.Compile: %s", err)
	}

	drv.mutex.Lock()
	if drv.status != StatusIdle {
		return ErrDriverNotIdel
	}
	drv.status = StatusReady
	drv.mutex.Unlock()

	go drv.compile()
	return nil
}

func (drv *MCWDriver) compile() {
	defer func() {
		drv.setStatus(StatusIdle)
	}()

	var err error

	logFilename := strconv.FormatInt(time.Now().Unix(), 10) + ".log"
	logFilepath := path.Join(drv.path.logDir, logFilename)
	logFile, err := os.Create(logFilepath)
	if err != nil {
		drv.lastCompile.err = fmt.Errorf("open log file: %s", err)
		return
	}
	defer logFile.Close()

	log.Println("Start running mc2pbrt...")
	drv.setStatus(StatusMc2pbrt)
	mc2pbrtMain := drv.path.mc2pbrtMain
	if strings.HasSuffix(mc2pbrtMain, ".py") {
		// TODO: which python should call?
		drv.cmdMc2pbrt = exec.Command("python3", drv.path.mc2pbrtMain, "--filename", "config.json")
	} else {
		drv.cmdMc2pbrt = exec.Command(drv.path.mc2pbrtMain, "--filename", "config.json")
	}
	drv.cmdMc2pbrt.Dir = drv.path.workdir
	drv.cmdMc2pbrt.Stdout = logFile
	drv.cmdMc2pbrt.Stderr = logFile
	err = drv.cmdMc2pbrt.Run()
	if err != nil {
		log.Printf("mc2pbrt: %s", err)
		drv.lastCompile.err = fmt.Errorf("mc2pbrt: %s", err)
		return
	}
	log.Println("Start running mc2pbrt...ok")

	log.Println("Start running pbrt...")
	drv.setStatus(StatusPbrt)
	targetPbrt := path.Join(drv.path.workdir, "scenes", "target.pbrt")
	drv.cmdPbrt = exec.Command(drv.path.pbrtBin, targetPbrt, "--outfile", "mc.png")
	drv.cmdPbrt.Dir = drv.path.workdir
	drv.cmdPbrt.Stdout = logFile
	drv.cmdPbrt.Stderr = logFile
	err = drv.cmdPbrt.Run()
	if err != nil {
		log.Printf("pbrt: %s", err)
		drv.lastCompile.err = fmt.Errorf("pbrt: %s", err)
		return
	}
	log.Println("Start running pbrt...ok")

	drv.lastCompile.err = nil
}

// StopCompile stop mc2pbrt and pbrt process
func (drv *MCWDriver) StopCompile() error {
	if drv.cmdMc2pbrt != nil {
		drv.cmdMc2pbrt.Process.Kill()
	}
	if drv.cmdPbrt != nil {
		drv.cmdPbrt.Process.Kill()
	}
	return nil
}

func (drv *MCWDriver) setStatus(s MCWStatus) {
	drv.mutex.Lock()
	drv.status = s
	drv.mutex.Unlock()
}

// GetStatus return the status of driver
func (drv *MCWDriver) GetStatus() MCWStatus {
	return drv.status
}

// GetLastCompileResult return the last render err
func (drv *MCWDriver) GetLastCompileResult() error {
	return drv.lastCompile.err
}

// GetImageBase64 return the render result in base64
func (drv *MCWDriver) GetImageBase64() (string, error) {
	bytes, err := ioutil.ReadFile(path.Join(drv.path.workdir, "mc.png"))
	if err != nil {
		return "", fmt.Errorf("mcwdrv.GetImageBase64: %s", err)
	}
	imgBase64 := base64.StdEncoding.EncodeToString(bytes)
	return imgBase64, nil
}

// ListLogs return list of filename
func (drv *MCWDriver) ListLogs() ([]string, error) {
	files, err := ioutil.ReadDir(drv.path.logDir)
	if err != nil {
		return nil, fmt.Errorf("mcwdrv.ListLogs: %s", err)
	}
	ret := []string{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".log") {
			continue
		}
		ret = append(ret, file.Name())
	}

	// Reverse the order, let the most recent log be the first file
	l := len(ret)
	for i := 0; i < l-i-1; i++ {
		ret[i], ret[l-i-1] = ret[l-i-1], ret[i]
	}
	return ret, nil
}

// GetLog return log in string type
func (drv *MCWDriver) GetLog(filename string) (string, error) {
	logFilepath := path.Join(drv.path.logDir, filename)
	bs, err := ioutil.ReadFile(logFilepath)
	if err != nil {
		return "", fmt.Errorf("mcwdrv.GetLog: %s", err)
	}
	return string(bs), nil
}

// DeleteLog delete log file
func (drv *MCWDriver) DeleteLog(filename string) error {
	logFilepath := path.Join(drv.path.logDir, filename)
	err := os.Remove(logFilepath)
	if err != nil {
		return fmt.Errorf("mcwdrv.DeleteLog: %s", err)
	}
	return nil
}

func (drv *MCWDriver) writeRenderConfig(rc RenderConfig) error {
	bytes, err := json.MarshalIndent(rc, "", "  ")
	if err != nil {
		return fmt.Errorf("mcwdrv.writeRenderConfig: %s", err)
	}

	err = ioutil.WriteFile(path.Join(drv.path.workdir, "config.json"), bytes, 0644)
	if err != nil {
		return fmt.Errorf("mcwdrv.writeRenderConfig: %s", err)
	}
	return nil
}
