package mcwdrv

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sync"
)

type MCWStatus int

const (
	StatusIdle MCWStatus = iota
	StatusReady
	StatusMc2pbrt
	StatusPbrt
)

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
	}
}

type Class struct {
	Name   string      `json:"name"`
	Params interface{} `json:"params"`
}

type RenderConfig struct {
	World  string
	Player string
	Sample int
	Radius int

	Method      Class
	Camera      Class
	Phenomenons []Class
}

var ErrDriverNotIdel = fmt.Errorf("mcwdrv.Compile: Driver status not idle")

func NewMCWDriver(workdir, mc2pbrtMain, pbrtBin string) (*MCWDriver, error) {
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

	return ret, nil
}

func (drv *MCWDriver) Compile(rc RenderConfig) error {
	drv.mutex.Lock()
	if drv.status != StatusIdle {
		return ErrDriverNotIdel
	}
	drv.status = StatusReady
	drv.mutex.Unlock()

	err := drv.writeRenderConfig(rc)
	if err != nil {
		return fmt.Errorf("mcwdrv.Compile: %s", err)
	}
	go drv.compile()
	return nil
}

func (drv *MCWDriver) compile() {
	defer func() {
		drv.setStatus(StatusIdle)
	}()

	var err error

	drv.setStatus(StatusMc2pbrt)
	drv.cmdMc2pbrt = exec.Command("python3", drv.path.mc2pbrtMain, "--filename", "config.json")
	drv.cmdMc2pbrt.Dir = drv.path.workdir
	drv.cmdMc2pbrt.Stdout = os.Stdout
	drv.cmdMc2pbrt.Stderr = os.Stderr
	err = drv.cmdMc2pbrt.Run()
	if err != nil {
		drv.lastCompile.err = fmt.Errorf("mc2pbrt: %s", err)
		return
	}

	drv.setStatus(StatusPbrt)
	targetPbrt := path.Join(drv.path.workdir, "scenes", "target.pbrt")
	drv.cmdPbrt = exec.Command(drv.path.pbrtBin, targetPbrt, "--outfile", "mc.png")
	drv.cmdPbrt.Dir = drv.path.workdir
	drv.cmdPbrt.Stdout = os.Stdout
	drv.cmdPbrt.Stderr = os.Stderr
	err = drv.cmdPbrt.Run()
	if err != nil {
		drv.lastCompile.err = fmt.Errorf("pbrt: %s", err)
		return
	}

	drv.lastCompile.err = nil
}

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

func (drv *MCWDriver) GetStatus() MCWStatus {
	return drv.status
}

func (drv *MCWDriver) GetLastCompileResult() error {
	return drv.lastCompile.err
}

func (drv *MCWDriver) GetImageBase64() (string, error) {
	bytes, err := ioutil.ReadFile(path.Join(drv.path.workdir, "mc.png"))
	if err != nil {
		return "", fmt.Errorf("mcwdrv.GetImageBase64: %s", err)
	}
	imgBase64 := base64.StdEncoding.EncodeToString(bytes)
	return imgBase64, nil
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
