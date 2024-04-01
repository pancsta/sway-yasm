package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/pancsta/sway-yast/internal/daemon"
	"github.com/spf13/cobra"
)

// TODO put these in a config and make it customizable
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
    foot --title "sway-yast" sway-yast fzf
`
	shellPickWin = `
    foot --title "sway-yast" sway-yast fzf-pick-win
`
	shellPickSpace = `
    foot --title "sway-yast" sway-yast fzf-pick-space
`
	shellPath = `
    foot --title "sway-yast" sway-yast fzf-path
`
)

///// TERM WRAPPER COMMANDS /////

// TODO open on all visible outputs, as screen session clients
// use https://github.com/rajveermalviya/go-wayland
func CmdSwitcher(_ *cobra.Command, _ []string) {
	if !shouldOpen() {
		log.Fatal("fzf error: already open")
	}
	_, err := run(shellSwitcher)
	if err != nil {
		log.Fatalf("foot error: %s", err)
	}
}

func CmdPickWin(_ *cobra.Command, _ []string) {
	if !shouldOpen() {
		log.Fatal("fzf error: already open")
	}
	_, err := run(shellPickWin)
	if err != nil {
		log.Fatal("foot error: " + err.Error())
	}
}

func CmdPickSpace(_ *cobra.Command, _ []string) {
	if !shouldOpen() {
		log.Fatal("fzf error: already open")
	}
	_, err := run(shellPickSpace)
	if err != nil {
		log.Fatalf("foot error: %s", err)
	}
}

func CmdPath(_ *cobra.Command, _ []string) {
	if !shouldOpen() {
		log.Fatal("fzf error: already open")
	}
	_, err := run(shellPath)
	if err != nil {
		log.Fatalf("foot error: %s", err)
	}
}

func CmdConfig(cmd *cobra.Command, _ []string) {
	mouseFollow, _ := cmd.Flags().GetBool("mouse-follows-focus")
	result, err := daemon.RemoteCall("Daemon.RemoteSetConfig", daemon.RPCArgs{
		MouseFollowsFocus: mouseFollow,
	})
	if err != nil {
		log.Fatalf("rpc error: %s", err)
	}
	if result != "" {
		log.Fatal("config error")
	}
	fmt.Println("Config updated")
	// TODO print the current config
}

func CmdRoot(cmd *cobra.Command, _ []string) {
	version, _ := cmd.Flags().GetBool("version")
	if version {
		build, ok := debug.ReadBuildInfo()
		if !ok {
			panic("No build info available")
		}
		fmt.Println(build.Main.Version)
		os.Exit(0)
	} else {
		fmt.Println("Yet Another Sway Tab\n\nUsage:\n$ sway-yast daemon\n$ sway-yast --help")
	}
}

///// FZF COMMANDS /////

func CmdFzf(_ *cobra.Command, _ []string) {
	// req the daemon
	input, err := daemon.RemoteCall("Daemon.RemoteFZFList", daemon.RPCArgs{})
	if err != nil {
		log.Fatalf("rpc error: %s", err)
	}

	// run fzf
	result, err := fzf(shellFzf, &input)
	if err != nil {
		log.Fatalf("fzf error: %s", err)
	}

	// match the window's ID at the end of the line
	winID, err := matchWinID(result)
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
	result, err := fzf(shellFzfPickWin, &input)
	if err != nil {
		log.Fatalf("fzf error: %s", err)
	}

	// match the window's ID at the end of the line
	winID, err := matchWinID(result)
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
	result, err := fzf(shellFzfPickSpace, &list)
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

func CmdFzfPath(_ *cobra.Command, _ []string) {
	// req the daemon
	list, err := daemon.RemoteCall("Daemon.RemoteGetPathFiles", daemon.RPCArgs{})
	if err != nil {
		log.Fatalf("rpc error: %s", err)
	}

	// run fzf
	result, err := fzf(shellFzfPath, &list)
	if err != nil {
		log.Fatalf("fzf error: %s", err)
	}

	// return the picked exe
	log.Printf("path: %s", result)
	_, err = daemon.RemoteCall("Daemon.RemoteExec", daemon.RPCArgs{
		ExePath: result,
	})
	if err != nil {
		log.Fatalf("error: cant run %s", result)
	}
}

///// HELPERS /////

func matchWinID(result string) (int, error) {
	re := regexp.MustCompile(`\((\d+)\)\s*$`)
	match := re.FindStringSubmatch(result)
	if len(match) == 0 {
		return 0, fmt.Errorf("no winID match")
	}
	return strconv.Atoi(match[1])
}

func shouldOpen() bool {
	pid := os.Getpid()
	shouldOpen, err := daemon.RemoteCall("Daemon.RemoteShouldOpen", daemon.RPCArgs{PID: pid})
	if err != nil {
		log.Printf("rpc error: %s", err)
		return false
	}
	return shouldOpen == "true"
}

func fzf(cmd string, input *string) (string, error) {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	if daemon.IsLightMode() {
		cmd = strings.TrimRight(cmd, " \n") + shellFzfLight
	}
	fzf := exec.Command(shell, "-c", cmd)
	fzf.Stdin = bytes.NewBuffer([]byte(*input))
	// bind the UI
	fzf.Stderr = os.Stderr
	// read the result
	result, err := fzf.Output()
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func run(cmd string) (string, error) {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	out, err := exec.Command(shell, "-c", cmd).Output()
	return string(out), err
}
