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
type ListenerFuncs struct {
	WinListenerFunc  WinListenerFunc
	ClipListenerFunc ClipListenerFunc
}
type WinListenerFunc func(DaemonAPI, types.WindowData)
type ClipListenerFunc func(DaemonAPI, string) string

var Registered map[string]UserFunc
var Listeners map[string][]*ListenerFuncs

// register registers a new user command function.
func register(name string, fn UserFunc) {
	if Registered == nil {
		Registered = make(map[string]UserFunc)
	}
	Registered[name] = fn
}

// listener registers a new event listener.
func listener(event string, fn *ListenerFuncs) {
	if Listeners == nil {
		Listeners = make(map[string][]*ListenerFuncs)
	}
	Listeners[event] = append(Listeners[event], fn)
}

// onClose registers a window "close" event listener.
func onClose(fn WinListenerFunc) {
	listener("close", &ListenerFuncs{WinListenerFunc: fn})
}

// onFocus registers a window "focus" event listener.
func onFocus(fn WinListenerFunc) {
	listener("focus", &ListenerFuncs{WinListenerFunc: fn})
}

// onNew registers a window "new" event listener.
func onNew(fn WinListenerFunc) {
	listener("new", &ListenerFuncs{WinListenerFunc: fn})
}

// onCopy registers a clipboard event listener, triggered when a history item is
// copied.
func onCopy(fn ClipListenerFunc) {
	listener("copy", &ListenerFuncs{ClipListenerFunc: fn})
}

// inspect prints the value to the daemon's log.
func inspect(val any) {
	log.Printf("Inspect: %+v\n", val)
}

// inspect prints the value to the daemon's log.
func p(msg string, vals ...any) {
	log.Printf(msg, vals...)
}
