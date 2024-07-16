package main

import (
	"io"
	"log"
	"os"

	"github.com/pancsta/sway-yasm/internal/cmds"
)

func main() {
	// TODO --status (PID, config, windows count)
	if os.Getenv("YASM_LOG") == "" {
		log.SetOutput(io.Discard)
	}
	// TODO slog
	logger := log.New(os.Stdout, "", 0)

	err := cmds.GetRootCmd(logger).Execute()
	if err != nil {
		logger.Fatal("cobra error:", err)
	}
}
