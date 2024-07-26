package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/pancsta/sway-yasm/internal/daemon"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// TODO gen fzf.yml for user overrides

const (
	shellFzf = `
  fzf \
    --prompt 'Switcher: ' \
    --bind "load:pos(2)" \
    --bind "change:pos(1)" \
    --layout=reverse --info=hidden \
    --bind=space:accept,tab:offset-down,btab:offset-up
`
	shellFzfPickWin = `
  fzf \
    --prompt 'Move which window to this workspace?: ' \
    --layout=reverse --info=hidden \
    --bind=space:accept,tab:offset-down,btab:offset-up
`
	shellFzfClipboard = `
  fzf \
    --prompt 'Copy which one to the clipboard?: ' \
    --layout=reverse --info=hidden \
    --bind=space:accept,tab:offset-down,btab:offset-up
`
	shellFzfPickSpace = `
  fzf \
    --prompt 'Move which workspace to this output?: ' \
    --layout=reverse --info=hidden \
    --bind=space:accept,tab:offset-down,btab:offset-up
`
	shellFzfPath = `
  fzf \
    --prompt 'Run: ' \
    --layout=reverse --info=hidden \
    --bind=space:accept,tab:offset-down,btab:offset-up
`
	// junegunn/seoul256.vim (light)
	shellFzfLight = ` \
    --color=bg+:#D9D9D9,bg:#E1E1E1,border:#C8C8C8,spinner:#719899,hl:#719872,fg:#616161,header:#719872,info:#727100,pointer:#E12672,marker:#E17899,fg+:#616161,preview-bg:#D9D9D9,prompt:#0099BD,hl+:#719899
`
	shellSwitcher = `
    foot --title "sway-yasm" sway-yasm fzf switcher
`
	shellPickWin = `
    foot --title "sway-yasm" sway-yasm fzf pick-win
`
	shellPickSpace = `
    foot --title "sway-yasm" sway-yasm fzf pick-space
`
	shellPath = `
    foot --title "sway-yasm" sway-yasm fzf path
`
	shellClipboard = `
    foot --title "sway-yasm" sway-yasm fzf clipboard
`
)

// ///// ///// /////
// ///// FZF COMMANDS
// ///// ///// /////

func CmdFzfSwitcher(_ *cobra.Command, _ []string) {
	// req the daemon
	input, err := daemon.RemoteCall("Daemon.RemoteFZFList", daemon.RPCArgs{})
	if err != nil {
		log.Fatalf("rpc error: %s", err)
	}

	// run fzf
	result, err := runFZF(shellFzf, &input)
	if err != nil {
		log.Fatalf("fzf error: %s", err)
	}

	// match the window's ID at the end of the line
	winID, err := matchSuffixID(result)
	if err != nil {
		log.Fatalf("error: %s", err)
	}

	// focus the window
	_, err = daemon.RemoteCall("Daemon.RemoteFocusWinID", daemon.RPCArgs{WinID: winID})
	if err != nil {
		log.Fatalf("rpc error: %s", err)
	}
}

func CmdFzfPickWin(_ *cobra.Command, _ []string) {
	// req the daemon
	input, err := daemon.RemoteCall("Daemon.RemoteFZFListPickWin", daemon.RPCArgs{})
	if err != nil {
		log.Fatalf("rpc error: %s", err)
	}
	// run fzf
	result, err := runFZF(shellFzfPickWin, &input)
	if err != nil {
		log.Fatalf("fzf error: %s", err)
	}

	// match the window's ID at the end of the line
	winID, err := matchSuffixID(result)
	if err != nil {
		log.Fatalf("error: %s", err)
	}

	// move the window to the current workspace
	_, err = daemon.RemoteCall("Daemon.RemoteMoveWinToSpace", daemon.RPCArgs{WinID: winID})
	if err != nil {
		log.Fatalf("rpc error: %s", err)
	}
}

func CmdFzfPickSpace(_ *cobra.Command, _ []string) {
	// req the daemon
	list, err := daemon.RemoteCall("Daemon.RemoteFZFListPickSpace", daemon.RPCArgs{})
	if err != nil {
		log.Fatalf("rpc error: %s", err)
	}

	// run fzf to pick the workspace
	result, err := runFZF(shellFzfPickSpace, &list)
	if err != nil {
		log.Fatalf("fzf error: %s", err)
	}

	// move the workspace to the current output
	_, err = daemon.RemoteCall("Daemon.RemoteMoveSpaceToOutput", daemon.RPCArgs{
		Workspace: strings.Trim(result, " \n"),
	})
	if err != nil {
		log.Fatalf("rpc error: %s", err)
	}
}

func CmdFzfClipboard(_ *cobra.Command, _ []string) {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	cmdClipman := "clipman show-history"

	// get json
	histJSON, err := exec.Command(shell, "-c", cmdClipman).Output()
	if err != nil {
		log.Fatalf("clipman error: %s", err)
	}

	// parse json
	var hist []string
	err = json.Unmarshal(histJSON, &hist)
	slices.Reverse(hist)
	if err != nil {
		log.Fatalf("json error: %s", err)
	}

	// prep fzf input
	fzfInput := ""
	for i, h := range hist {
		clean := strings.Trim(clipboardSanitize.ReplaceAllString(h, " "), " ")
		if clean == "" {
			continue
		}

		fzfInput += fmt.Sprintf("(%d) %s\n", i, clean)
	}

	// run fzf
	result, err := runFZF(shellFzfClipboard, &fzfInput)
	if err != nil {
		log.Fatalf("fzf error: %s", err)
	}
	// match the entry's ID at the end of the line
	id, err := matchPrefixID(result)
	if err != nil {
		log.Fatalf("error: %s", err)
	}

	// set the clipboard
	_, err = daemon.RemoteCall("Daemon.RemoteCopy", daemon.RPCArgs{
		Clipboard: hist[id],
	})
	if err != nil {
		log.Fatalf("rpc error: %s", err)
	}
}

func CmdFzfPath(_ *cobra.Command, _ []string) {
	// req the daemon
	list, err := daemon.RemoteCall("Daemon.RemoteGetPathFiles", daemon.RPCArgs{})
	if err != nil {
		log.Fatalf("rpc error: %s", err)
	}

	// run fzf
	result, err := runFZF(shellFzfPath, &list)
	if err != nil {
		log.Fatalf("fzf error: %s", err)
	}

	// return the picked exe
	log.Printf("path: %s", result)
	result, err = daemon.RemoteCall("Daemon.RemoteExec", daemon.RPCArgs{
		ExePath: result,
	})
	if err != nil {
		log.Fatalf("error: cant run %s", result)
	}
}
