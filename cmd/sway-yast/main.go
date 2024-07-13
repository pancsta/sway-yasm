package main

import (
	"io"
	"log"
	"os"

	"github.com/pancsta/sway-yast/internal/cmds"
	"github.com/spf13/cobra"
	"os/signal"
	"syscall"
)

func main() {
	// TODO --status (PID, config, windows count)
	// TODO readme, screenshots
	// TODO auto bind default shortcuts via --bind-default-keys
	//   alt+tab, cmds+o, cmds+p
	// TODO include desktop shortcuts in "path" (Name, Exe)
	//  - ~/.local/share/applications
	//  - /usr/share/applications
	if os.Getenv("YAST_LOG") == "" {
		log.SetOutput(io.Discard)
	}
	out := log.New(os.Stdout, "", 0)

	cmdList := cmds.GetCmds(out)

	var rootCmd = &cobra.Command{
		Use: "sway-yast",
		Run: cmds.CmdRoot,
	}
	rootCmd.AddCommand(cmdList...)
	rootCmd.Flags().Bool("version", false,
		"Print version and exit")

	err := rootCmd.Execute()
	if err != nil {
		out.Fatal("cobra error:", err)
	}
}

// TODO
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
