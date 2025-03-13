package usrCmds

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// TODO config
var skipOutputs = []string{}

func init() {
	register("next-vnc", NextVnc)
}

func getOutputNum(name string) string {
	output := strings.Split(name, "-")
	if len(output) < 2 {
		return ""
	}

	return output[1]
}

func NextVnc(d DaemonAPI, args map[string]string) (string, error) {
	// current output
	win := d.FocusedWindow()

	// other visible outputs for this vnc session
	visibleOutputs := []string{}
	for _, name := range d.Outputs() {
		if vncOutputVisible(getOutputNum(name)) &&
			!slices.Contains(skipOutputs, name) {
			visibleOutputs = append(visibleOutputs, name)
		}
	}
	var nextOutput string

	if len(visibleOutputs) == 0 {

		return "", fmt.Errorf("no visible vnc outputs")

	} else if len(visibleOutputs) <= 1 {
		if os.Getenv("YASM_LOG") != "" {
			log.Printf("only 1 output vnc visible (%s)", visibleOutputs[0])
		}
		nextOutput = visibleOutputs[0]

	} else {
		log.Printf("visible outputs: %s", visibleOutputs)

		idx := slices.Index(visibleOutputs, win.Output)
		if idx == -1 {
			// current output not vnc visible, pick 1st visible one
			idx = 0
		}
		if _, ok := args["--back"]; ok {
			idx = (idx - 1)
			if idx < 0 {
				idx = len(visibleOutputs) - 1
			}
		} else {
			// fwd
			idx = (idx + 1) % len(visibleOutputs)
		}
		nextOutput = visibleOutputs[idx]
	}

	// find the MRU window on next output to get its workspace
	for _, winId := range d.MruList() {
		win := d.WindowById(winId)
		if win.Output != nextOutput {
			continue
		}

		err := d.FocusSpace(win.Workspace, win.Output)

		// end
		return nextOutput + " - " + win.Title + " (" + win.Workspace + ")\n", err
	}

	return "", fmt.Errorf("no windows found for output %s", nextOutput)
}

func vncOutputVisible(outputNum string) bool {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	cmd := `wayvncctl -S /tmp/wayvnc-` + outputNum + ` client-list`
	// TODO d.Log
	if os.Getenv("YASM_LOG") != "" {
		log.Println(cmd)
	}
	out, err := exec.Command(shell, "-c", cmd).Output()
	if err != nil || len(out) == 0 {
		return false
	}

	return true
}
