package usrCmds

import (
	"github.com/Difrex/gosway/ipc"
	"github.com/pancsta/sway-yasm/internal/types"
	"log"
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
	MouseToOutput(output string) error
	FocusWinID(id int) error
	WinMatchApp(win types.WindowData, match string) bool
	WinMatchTitle(win types.WindowData, match string) bool
}

type UserFunc func(DaemonAPI, map[string]string) (string, error)
type ListenerFunc func(DaemonAPI, types.WindowData)

var Registered map[string]UserFunc
var Listeners map[string][]ListenerFunc

func init() {
	if Listeners == nil {
		Listeners = make(map[string][]ListenerFunc)
	}
}

// register registers a new user command function.
func register(name string, fn UserFunc) {
	if Registered == nil {
		Registered = make(map[string]UserFunc)
	}
	Registered[name] = fn
}

// listener registers a new event listener.
func listener(event string, fn ListenerFunc) {
	if Listeners == nil {
		Listeners = make(map[string][]ListenerFunc)
	}
	Listeners[event] = append(Listeners[event], fn)
}

// register registers a new user command function.
func onClose(fn ListenerFunc) {
	listener("close", fn)
}

// register registers a new user command function.
func onFocus(fn ListenerFunc) {
	listener("focus", fn)
}

// register registers a new user command function.
func onNew(fn ListenerFunc) {
	listener("new", fn)
}

// inspect prints the value to the daemon's log.
func inspect(val any) {
	log.Printf("Inspect: %+v\n", val)
}

// inspect prints the value to the daemon's log.
func p(msg string, vals ...any) {
	log.Printf(msg, vals...)
}
