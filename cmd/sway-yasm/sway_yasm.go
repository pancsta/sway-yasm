package main

import (
	"io"
	"log"
	"os"

	"github.com/pancsta/sway-yasm/internal/cmds"
)

func main() {
	// TODO --status (PID, config, windows count)
	// TODO slog
	logger := log.New(os.Stdout, "", 0)
	if os.Getenv("YASM_LOG") == "" {
		logger.SetOutput(io.Discard)
	}

	// start the root command
	err := cmds.GetRootCmd(logger).Execute()
	if err != nil {
		logger.Fatal("cobra error:", err)
	}
}
