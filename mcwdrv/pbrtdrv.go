package mcwdrv

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// PbrtStatus stores current pbrt status
type PbrtStatus struct {
	AllSec   float64 `json:"all_sec"`
	LeaveSec float64 `json:"leave_sec"`
}

// ErrPbrtStatusPatternNotMatch occur if status string not math pattern
var ErrPbrtStatusPatternNotMatch = errors.New("Pbrt status pattern not match")

// TODO: use more strict pattern
// Example: Rendering: [++++++              ]  (0.8s|1.1s)
var pbrtWorkingStatusPattern = regexp.MustCompile(`Rendering: \[\+* *\]  \(.*s\|.*s\)`)
var pbrtEndingStatusPattern = regexp.MustCompile(`Rendering: \[\+* *\]  \(.*s\)`)

type pbrtDrv struct {
	status  *PbrtStatus
	cmd     *exec.Cmd
	workdir string
	bin     string
}

func (pd *pbrtDrv) run(logFile io.Writer) error {
	targetPbrt := path.Join(pd.workdir, "scenes", "target.pbrt")
	pd.cmd = exec.Command(pd.bin, targetPbrt, "--outfile", "mc.png")
	pd.cmd.Dir = pd.workdir
	pd.cmd.Stderr = logFile

	stdoutPipe, err := pd.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("mc2pbrtdrv.run: %s", err)
	}
	go pd.reader(stdoutPipe)
	// drv.cmdPbrt.Stderr = logFile

	err = pd.cmd.Run()
	if err != nil {
		return fmt.Errorf("mc2pbrtdrv.run: %s", err)
	}
	return nil
}

func (pd *pbrtDrv) kill() {
	if pd.cmd != nil {
		pd.cmd.Process.Kill()
	}
}

func (pd *pbrtDrv) getStatus() *PbrtStatus {
	return pd.status
}

func (pd *pbrtDrv) reader(reader io.Reader) {
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
					pd.status = ps
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
