package mcwdrv

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
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

// PbrtStatus stores current pbrt status
type PbrtStatus struct {
	AllSec   float64 `json:"all_sec"`
	LeaveSec float64 `json:"leave_sec"`
}

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

	pbrtStatus *PbrtStatus
}

// Class is a type in Minecraft render config
type Class struct {
	Name   string      `json:"name"`
	Params interface{} `json:"params"`
}

// RenderConfig is Minecraft scene render config for mc2pbrt
// more info ref: https://github.com/PbrtCraft/mc2pbrt
type RenderConfig struct {
	World      string
	Player     string
	Sample     int
	Radius     int
	Resolution struct {
		Width  int
		Height int
	}

	Method      Class
	Camera      Class
	Phenomenons []Class
}

// ErrDriverNotIdel occur when try to compile when compiling
var ErrDriverNotIdel = fmt.Errorf("mcwdrv.Compile: Driver status not idle")

// Config for MCWDriver
type Config struct {
	Workdir     string `yaml:"workdir"`      // Path to workdir
	Mc2pbrtMain string `yaml:"mc2pbrt_main"` // Path to mc2pbrt/main.py
	PbrtBin     string `yaml:"pbrt_bin"`     // Path to pbrt binary
	LogDir      string `yaml:"log_dir"`      // mcwdrv log directory
}

// NewMCWDriver return a minecraft world driver
func NewMCWDriver(conf *Config) (*MCWDriver, error) {
	ret := &MCWDriver{}
	var err error
	ret.path.mc2pbrtMain, err = filepath.Abs(conf.Mc2pbrtMain)
	if err != nil {
		return nil, fmt.Errorf("mcwdrv.NewMCWDriver: %s", err)
	}
	ret.path.pbrtBin, err = filepath.Abs(conf.PbrtBin)
	if err != nil {
		return nil, fmt.Errorf("mcwdrv.NewMCWDriver: %s", err)
	}

	ret.path.workdir, err = filepath.Abs(conf.Workdir)
	if err != nil {
		return nil, fmt.Errorf("mcwdrv.NewMCWDriver: %s", err)
	}

	ret.path.logDir, err = filepath.Abs(conf.LogDir)
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
	drv.cmdPbrt.Stderr = logFile

	stdoutPipe, err := drv.cmdPbrt.StdoutPipe()
	if err != nil {
		log.Printf("pbrt: %s", err)
		drv.lastCompile.err = fmt.Errorf("pbrt: %s", err)
		return
	}
	go drv.pbrtReader(stdoutPipe)
	// drv.cmdPbrt.Stderr = logFile

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

// GetPbrtStatus return status of pbrt
func (drv *MCWDriver) GetPbrtStatus() *PbrtStatus {
	return drv.pbrtStatus
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

func (drv *MCWDriver) pbrtReader(reader io.Reader) {
	var out []byte
	buf := make([]byte, 12, 12)
	for {
		n, err := reader.Read(buf[:])
		out = append(out, buf[:n]...)
		if n > 0 {
			for {
				newLine := findLine(out)
				if newLine == -1 {
					break
				}
				log.Println("input", string(out[:newLine]))
				ps, err := parsePbrtStatus(string(out[:newLine]))
				if err == nil {
					drv.pbrtStatus = ps
					log.Println("get", *ps)
				}
				out = out[newLine+1:]
			}
		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			}
			break
		}
	}
}

func findLine(val []byte) int {
	nl := bytes.Index(val, []byte("\n"))
	rl := bytes.Index(val, []byte("\r"))
	if nl == -1 && rl == -1 {
		return -1
	} else if nl == -1 {
		return rl
	} else if rl == -1 {
		return nl
	}
	if nl > rl {
		return rl
	}
	return nl
}

// ErrPbrtStatusPatternNotMatch occur if status string not math pattern
var ErrPbrtStatusPatternNotMatch = errors.New("Pbrt status pattern not match")

// TODO: use more strict pattern
// Example: Rendering: [++++++              ]  (0.8s|1.1s)
var pbrtWorkingStatusPattern = regexp.MustCompile(`Rendering: \[\+* *\]  \(.*s\|.*s\)`)
var pbrtEndingStatusPattern = regexp.MustCompile(`Rendering: \[\+* *\]  \(.*s\)`)

func parsePbrtStatus(s string) (*PbrtStatus, error) {
	if pbrtWorkingStatusPattern.MatchString(s) {
		return parseWorkingPbrtStatus(s)
	} else if pbrtEndingStatusPattern.MatchString(s) {
		return parseEndingPbrtStatus(s)
	}
	return nil, fmt.Errorf("mcwdrv.parsePbrtStatus: %s", ErrPbrtStatusPatternNotMatch)
}

func parseWorkingPbrtStatus(s string) (*PbrtStatus, error) {
	leftIdx := strings.Index(s, "(")
	midIdx := strings.Index(s[leftIdx:], "s|") + leftIdx
	rightIdx := strings.Index(s[leftIdx:], "s)") + leftIdx

	allSecStr := s[leftIdx+1 : midIdx]
	leaveSecStr := s[midIdx+2 : rightIdx]
	allSec, err := strconv.ParseFloat(allSecStr, 64)
	if err != nil {
		return nil, fmt.Errorf("mcwdrv.parseWorkingPbrtStatus: %s", err)
	}
	leaveSec, err := strconv.ParseFloat(leaveSecStr, 64)
	if err != nil {
		return nil, fmt.Errorf("mcwdrv.parseWorkingPbrtStatus: %s", err)
	}
	return &PbrtStatus{
		AllSec:   allSec,
		LeaveSec: leaveSec,
	}, nil
}

func parseEndingPbrtStatus(s string) (*PbrtStatus, error) {
	leftIdx := strings.Index(s, "(")
	rightIdx := strings.Index(s[leftIdx:], "s)") + leftIdx

	allSecStr := s[leftIdx+1 : rightIdx]
	allSec, err := strconv.ParseFloat(allSecStr, 64)
	if err != nil {
		return nil, fmt.Errorf("mcwdrv.parseWorkingPbrtStatus: %s", err)
	}
	return &PbrtStatus{
		AllSec:   allSec,
		LeaveSec: 0,
	}, nil
}
