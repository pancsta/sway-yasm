package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/pancsta/sway-yast/internal/cmd"
	"github.com/pancsta/sway-yast/internal/daemon"
	"github.com/spf13/cobra"
	"os/signal"
	"syscall"
)

func main() {
	// TODO --status (PID, config, windows count)
	// TODO readme, screenshots
	// TODO auto bind default shortcuts via --bind-default-keys
	//   alt+tab, cmd+o, cmd+p
	// TODO include desktop shortcuts in "path" (Name, Exe)
	//  - ~/.local/share/applications
	//  - /usr/share/applications
	if os.Getenv("YAST_LOG") == "" {
		log.SetOutput(io.Discard)
	}
	out := log.New(os.Stdout, "", 0)

	cmdDaemon := &cobra.Command{
		Use:   "daemon",
		Short: "Start tracking focus in sway",
		Run: func(cmd *cobra.Command, args []string) {
			mouseFollow, _ := cmd.Flags().GetBool("mouse-follows-focus")
			autoconfig, _ := cmd.Flags().GetBool("autoconfig")
			defaultKeybindings, _ := cmd.Flags().GetBool("default-keybindings")
			d := &daemon.Daemon{
				MouseFollowsFocus:  mouseFollow,
				Autoconfig:         autoconfig,
				DefaultKeybindings: defaultKeybindings,
				Out:                out,
			}
			if mouseFollow {
				d.Out.Println("Mouse follows focus enabled")
			}
			d.Start()
		},
	}
	// TODO extract
	cmdDaemon.Flags().Bool("mouse-follows-focus", false,
		"Calls 'input ... map_to_output OUTPUT' on each focus")
	cmdDaemon.Flags().Bool("autoconfig", true,
		"Automatic configuration of layout")
	cmdDaemon.Flags().Bool("default-keybindings", false,
		"Add default keybindings")

	cmdList := &cobra.Command{
		Use:   "mru-list",
		Short: "Print a list of MRU window IDs",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(daemon.RemoteCall("Daemon.RemoteWinList", daemon.RPCArgs{}))
		},
	}

	cmdFzf := &cobra.Command{
		Use:   "fzf",
		Short: "Run fzf with a list of windows",
		Run:   cmd.CmdFzf,
	}

	cmdFzfPickWin := &cobra.Command{
		Use:   "fzf-pick-win",
		Short: "Run fzf with a list of windows to pick",
		Run:   cmd.CmdFzfPickWin,
	}

	cmdFzfPickSpace := &cobra.Command{
		Use:   "fzf-pick-space",
		Short: "Run fzf with a list of workspaces to pick",
		Run:   cmd.CmdFzfPickSpace,
	}

	cmdFzfPath := &cobra.Command{
		Use:   "fzf-path",
		Short: "Run fzf with a list of executable files from PATH",
		Run:   cmd.CmdFzfPath,
	}

	cmdSwitcher := &cobra.Command{
		Use:   "switcher",
		Short: "Show the switcher window using foot",
		Run:   cmd.CmdSwitcher,
	}

	cmdPickWin := &cobra.Command{
		Use:   "pick-win",
		Short: "Show the window picker using foot",
		Run:   cmd.CmdPickWin,
	}

	cmdPickSpace := &cobra.Command{
		Use:   "pick-space",
		Short: "Show the workspace picker using foot",
		Run:   cmd.CmdPickSpace,
	}

	cmdPath := &cobra.Command{
		Use:   "path",
		Short: "Show the +x files from PATH using foot",
		Run:   cmd.CmdPath,
	}

	cmdConfig := &cobra.Command{
		Use:   "config",
		Short: "Change the config of a running daemon process",
		Run:   cmd.CmdConfig,
	}
	// TODO extract
	cmdConfig.Flags().Bool("mouse-follows-focus", false,
		"Calls 'input ... map_to_output OUTPUT' on each focus")

	var rootCmd = &cobra.Command{
		Use: "sway-yast",
		Run: cmd.CmdRoot,
	}
	rootCmd.AddCommand(cmdDaemon, cmdList, cmdFzf, cmdSwitcher, cmdFzfPickWin, cmdPickWin, cmdConfig, cmdFzfPickSpace,
		cmdPickSpace, cmdPath, cmdFzfPath)
	rootCmd.Flags().Bool("version", false,
		"Print version and exit")

	err := rootCmd.Execute()
	if err != nil {
		out.Fatal("cobra error:", err)
	}
}

func waitForExit() {
	// Create a channel to receive OS signals.
	sigs := make(chan os.Signal, 1)

	// `signal.Notify` makes `os.Signal` send OS signals to the channel.
	// If no signals are provided, all incoming signals will be relayed to the channel.
	// Otherwise, just the provided signals will.
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// The program will wait here until it gets an OS signal, such as SIGINT or SIGTERM,
	// and then it will exit.
	<-sigs
}
