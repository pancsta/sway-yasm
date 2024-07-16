package cmds

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/lithammer/dedent"
	"github.com/pancsta/sway-yasm/internal/daemon"
	"github.com/spf13/cobra"
	"runtime/debug"
)

var clipboardSanitize = regexp.MustCompile(`\s+`)

// ///// ///// /////
// ///// COBRAS
// ///// ///// /////

func mouseFollowsFocusFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("mouse-follows-focus", false,
		"Calls 'input ... map_to_output OUTPUT' on each focus")
}

func GetRootCmd(logger *log.Logger) *cobra.Command {

	cmdDaemon := &cobra.Command{
		Use:   "daemon",
		Short: "Start tracking focus in sway",
		Run:   cmdDaemon(logger),
	}

	mouseFollowsFocusFlag(cmdDaemon)
	cmdDaemon.Flags().Bool("autoconfig", true,
		"Automatically configure the layout and start clipman")
	cmdDaemon.Flags().Bool("default-keybindings", false,
		"Add default keybindings")

	cmdMRUList := &cobra.Command{
		Use:   "mru-list",
		Short: "Print a list of MRU window IDs",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(daemon.RemoteCall("Daemon.RemoteWinList", daemon.RPCArgs{}))
		},
	}

	cmdFzfSwitcher := &cobra.Command{
		Use:   "switcher",
		Short: "Run fzf with a list of windows",
		Run:   CmdFzfSwitcher,
	}

	cmdFzfPickWin := &cobra.Command{
		Use:   "pick-win",
		Short: "Run fzf with a list of windows to pick",
		Run:   CmdFzfPickWin,
	}

	cmdFzfPickSpace := &cobra.Command{
		Use:   "pick-space",
		Short: "Run fzf with a list of workspaces to pick",
		Run:   CmdFzfPickSpace,
	}

	cmdFzfPath := &cobra.Command{
		Use:   "path",
		Short: "Run fzf with a list of executable files from PATH",
		Long: "Run fzf with a list of executable files from PATH, with all the " +
				"dirs being watched for changes.",
		Run: CmdFzfPath,
	}

	cmdFzfPickClip := &cobra.Command{
		Use:   "clipboard",
		Short: "Run fzf with your clipboard history and copy the selection",
		Run:   CmdFzfClipboard,
	}

	cmdFzf := &cobra.Command{
		Use:   "fzf",
		Short: "Pure FZF versions of the switcher and pickers",
		Long: "Pure FZF versions of the switcher and pickers, which allows them " +
				"to be rendered directly in the terminal.",
	}

	cmdFzf.AddCommand(cmdFzfSwitcher, cmdFzfPickWin, cmdFzfPickSpace, cmdFzfPath, cmdFzfPickClip)

	cmdUserCmd := &cobra.Command{
		Use:     "usr-cmd",
		Short:   "Run a user command with a specific name and optional args",
		Example: "sway-yasm usr-cmd resize-toggle -- -f=1",
		Run:     CmdUsrCmd,
		Args:    cobra.ExactArgs(1),
	}

	cmdSwitcher := &cobra.Command{
		Use:   "switcher",
		Short: "Show the window switcher window using foot",
		Long: "Show the window switcher window using foot in the Most Recently " +
				"Used order. The list can be traversed by pressing Tab or arrows.",
		Run: CmdSwitcher,
	}

	cmdPickWin := &cobra.Command{
		Use:   "pick-win",
		Short: "Show the window picker using foot",
		Run:   CmdPickWin,
	}

	cmdPickSpace := &cobra.Command{
		Use:   "pick-space",
		Short: "Show the workspace picker using foot",
		Run:   CmdPickSpace,
	}

	cmdPath := &cobra.Command{
		Use:   "path",
		Short: "Show the +x files from PATH using foot",
		Long: "Show the +x files from PATH using foot, with all the dirs being " +
				"watched for changes.",
		Run: CmdPath,
	}

	cmdWinToSpace := &cobra.Command{
		Use:   "win-to-space",
		Short: "Move the current window to a specific workspace",
		Run:   CmdWinToSpace,
		Args:  cobra.ExactArgs(1),
	}

	cmdConfig := &cobra.Command{
		Use:   "config",
		Short: "Change the config of a running daemon process",
		Run:   CmdConfig,
	}
	mouseFollowsFocusFlag(cmdConfig)

	cmdClipboard := &cobra.Command{
		Use:   "clipboard",
		Short: "Set the clipboard contents from the history",
		Run:   CmdClipboard,
	}

	var rootCmd = &cobra.Command{
		Use: "sway-yasm",
		Run: CmdRoot,
	}
	rootCmd.AddCommand(cmdDaemon, cmdMRUList, cmdSwitcher, cmdPickWin, cmdConfig,
		cmdPickSpace, cmdPath, cmdUserCmd, cmdWinToSpace, cmdClipboard, cmdFzf)
	rootCmd.Flags().Bool("version", false,
		"Print version and exit")

	return rootCmd
}

func cmdDaemon(logger *log.Logger) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		mouseFollow, _ := cmd.Flags().GetBool("mouse-follows-focus")
		autoconfig, _ := cmd.Flags().GetBool("autoconfig")
		defaultKeybindings, _ := cmd.Flags().GetBool("default-keybindings")
		d := &daemon.Daemon{
			MouseFollowsFocus:  mouseFollow,
			Autoconfig:         autoconfig,
			DefaultKeybindings: defaultKeybindings,
			Logger:             logger,
		}
		if mouseFollow {
			d.Logger.Println("Mouse follows focus enabled")
		}
		d.Start()
	}
}

// ///// ///// /////
// ///// TERM WRAPPER COMMANDS
// ///// ///// /////

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

func CmdClipboard(_ *cobra.Command, _ []string) {
	if !shouldOpen() {
		log.Fatal("fzf error: already open")
	}

	_, err := run(shellClipboard)
	if err != nil {
		log.Fatalf("foot error: %s", err)
	}
}

// ///// ///// /////
// ///// OTHER CMDS
// ///// ///// /////

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
		fmt.Println(fmt.Sprintf(dedent.Dedent(strings.Trim(`
		sway-yasm: SWAY Yet Another Sway Manager
		
		Daemon for managing Sway WM windows, workspaces, outputs, clipboard and PATH
		using FZF, both as a floating window and in the terminal.
		
		Usage:
		
		$ sway-yasm daemon --autoconfig --default-keybindings
		$ sway-yasm switcher
		$ sway-yasm help`, " \n"))))
	}
}

func CmdUsrCmd(_ *cobra.Command, args []string) {
	usrArgs := ""
	if len(args) > 1 {
		usrArgs = strings.Join(args[1:], " ")
	}

	result, err := daemon.RemoteCall("Daemon.RemoteUsrCmd", daemon.RPCArgs{
		UsrCmd:  args[0],
		UsrArgs: usrArgs,
	})
	if err != nil {
		fmt.Printf("error: %s", err)
	}

	// TODO allow for fzf
	fmt.Printf(result)
}

func CmdWinToSpace(_ *cobra.Command, args []string) {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		log.Fatalf("error: %s", err)
	}

	_, err = daemon.RemoteCall("Daemon.RemoteWinToSpace", daemon.RPCArgs{
		SpaceNum: id,
	})
	if err != nil {
		log.Fatalf("rpc error: %s", err)
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
	fmt.Println("Config updated:")
	fmt.Printf("- mouse follows focus: %t\n", mouseFollow)
	// TODO print out the current config (as yaml)
}

// ///// ///// /////
// ///// HELPERS
// ///// ///// /////

func matchSuffixID(result string) (int, error) {
	re := regexp.MustCompile(`\((\d+)\)\s*$`)
	match := re.FindStringSubmatch(result)
	if len(match) == 0 {
		return 0, fmt.Errorf("no (ID) match")
	}

	return strconv.Atoi(match[1])
}

func matchPrefixID(result string) (int, error) {
	re := regexp.MustCompile(`^\s*\((\d+)\)`)
	match := re.FindStringSubmatch(result)
	if len(match) == 0 {
		return 0, fmt.Errorf("no (ID) match")
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

func runFZF(cmd string, input *string) (string, error) {
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
