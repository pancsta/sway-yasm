package usrCmds

import (
	"github.com/Difrex/gosway/ipc"
	"github.com/pancsta/sway-yast/internal/types"
)

type DaemonAPI interface {
	FocusedWindow() types.WindowData
	ListSpaces(skipOutputs []string) ([]string, error)
	GetWinTreePath(id int) ([]*ipc.Node, error)
	PrevWindow() types.WindowData
	SwayMsgs(msgs []string) error
	SwayMsg(msg string, args ...any) error
	MoveWinToSpaceNum(winID, spaceNum int) error
	MoveWinToSpace(winID int, space string) error
	MoveSpaceToOutput(space, output string, focusedWinData types.WindowData) error
	ListWindows() map[string]types.WindowData
	WinMatch(win types.WindowData, match string, matchApp, matchTitle bool) bool
	MouseToOutput(output string) error
	FocusWinID(id int) error
}

type UserFunc func(DaemonAPI, map[string]string) (string, error)

var (
	Registered map[string]UserFunc
)

// register registers a user command
func register(name string, fn UserFunc) {
	if Registered == nil {
		Registered = make(map[string]UserFunc)
	}
	Registered[name] = fn
}
