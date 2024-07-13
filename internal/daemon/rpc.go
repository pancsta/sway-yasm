package daemon

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strings"
	"syscall"
	"time"

	ss "github.com/pancsta/sway-yast/internal/watcher/states"
	usrCmds "github.com/pancsta/sway-yast/pkg/usr-cmds"
)

// RPC

var client *rpc.Client

type RPCArgs struct {
	WinID             int
	SpaceNum          int
	Workspace         string
	PID               int
	MouseFollowsFocus bool
	ExePath           string
	UsrCmd            string
	UsrArgs           string
}

// RemoteWinList is an RPC method
func (d *Daemon) RemoteWinList(_ RPCArgs, reply *string) error {
	ids := ""
	for _, id := range d.winFocus {
		ids += fmt.Sprintf("%s ", id)
	}
	*reply = ids
	return nil
}

// RemoteFZFList is an RPC method
func (d *Daemon) RemoteFZFList(_ RPCArgs, reply *string) error {
	ret := ""
	// TODO extract
	for _, id := range d.winFocus {
		data := d.winData[id]
		display := strings.Replace(data.Output, "HEADLESS-", "H-", 1)
		// ret += fmt.Sprintf("%-*s (%s) %s| %-*s | %-*s | %-*s \n",
		ret += fmt.Sprintf("%-*s | %-*s | %-*s | %-*s (%s) \n",
			lenDisplay, maxLen(display, lenDisplay),
			lenSpace, maxLen(data.Workspace, lenSpace),
			lenApp, maxLen(data.App, lenApp),
			lenTitle, maxLen(data.Title, lenTitle),
			id,
		)
	}
	*reply = ret
	return nil
}

// RemoteFZFListPickWin is an RPC method
func (d *Daemon) RemoteFZFListPickWin(_ RPCArgs, reply *string) error {
	space := d.winData[d.winFocus[0]].Workspace
	ret := ""
	// TODO extract
	for _, id := range d.winFocus {
		data := d.winData[id]
		// skip same workspace
		if data.Workspace == space {
			continue
		}
		display := strings.Replace(data.Output, "HEADLESS-", "H-", 1)
		// ret += fmt.Sprintf("%-*s (%s) %s| %-*s | %-*s | %-*s \n",
		ret += fmt.Sprintf("%-*s | %-*s | %-*s | %-*s (%s) \n",
			lenDisplay, maxLen(display, lenDisplay),
			lenSpace, maxLen(data.Workspace, lenSpace),
			lenApp, maxLen(data.App, lenApp),
			lenTitle, maxLen(data.Title, lenTitle),
			id,
		)
	}
	*reply = ret
	return nil
}

// RemoteFZFListPickSpace is an RPC method
func (d *Daemon) RemoteFZFListPickSpace(_ RPCArgs, reply *string) error {
	currWin := d.FocusedWindow()
	spaces, err := d.ListSpaces([]string{currWin.Output})
	if err != nil {
		log.Printf("error: %s", err)
		return err
	}
	*reply = strings.Join(spaces, "\n")
	return nil
}

// RemoteShouldOpen is an RPC method
func (d *Daemon) RemoteShouldOpen(args RPCArgs, reply *string) error {
	if d.openedByPID == 0 {
		*reply = "true"
		d.openedByPID = args.PID
		d.openedAt = time.Now()
		return nil
	}
	// check if the holding process is alive
	proc, _ := os.FindProcess(d.openedByPID)
	timeoutOut := time.Now().Sub(d.openedAt) > pidTimeout
	if dead := proc.Signal(syscall.Signal(0)); dead != nil || timeoutOut {
		d.openedByPID = args.PID
		d.openedAt = time.Now()
		*reply = "true"
	} else {
		*reply = "false"
	}
	return nil
}

// RemoteFocusWinID is an RPC method
func (d *Daemon) RemoteFocusWinID(args RPCArgs, _ *string) error {
	log.Printf("focusing %d...", args.WinID)
	err := d.FocusWinID(args.WinID)
	if err != nil {
		log.Printf("error: %s", err)
		return err
	}
	return nil
}

// RemoteMoveSpaceToOutput is an RPC method
func (d *Daemon) RemoteMoveSpaceToOutput(args RPCArgs, _ *string) error {
	currentWin := d.FocusedWindow()
	if currentWin.Output == "" {
		err := errors.New("no focused window / output")
		log.Printf("error: %s", err)

		return err
	}

	err2 := d.MoveSpaceToOutput(args.Workspace, currentWin.Output, currentWin)
	if err2 != nil {
		return err2
	}

	return nil
}

// RemoteMoveWinToSpace is an RPC method
func (d *Daemon) RemoteMoveWinToSpace(args RPCArgs, _ *string) error {
	space := d.FocusedWindow().Workspace
	if space == "" {
		err := errors.New("no focused window / space")
		log.Printf("error: %s", err)
		return err
	}
	log.Printf("moving win %d to %s", args.WinID, space)
	_, err := d.conn.RunSwayCommand(fmt.Sprintf(`[con_id="%d"] move workspace %s`, args.WinID, space))
	if err != nil {
		log.Printf("error: %s", err)
		return err
	}
	err = d.FocusWinID(args.WinID)
	if err != nil {
		log.Printf("error: %s", err)
		return err
	}
	return nil
}

// RemoteSetConfig is an RPC method
func (d *Daemon) RemoteSetConfig(args RPCArgs, _ *string) error {
	log.Printf("RemoteSetConfig %v...", args.MouseFollowsFocus)
	d.MouseFollowsFocus = args.MouseFollowsFocus
	if !d.MouseFollowsFocus {
		// set the pointer to all the outputs
		_, err := d.conn.RunSwayCommand(`input 0:0:wlr_virtual_pointer_v1 map_to_output "*"`)
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoteGetPathFiles is an RPC method
func (d *Daemon) RemoteGetPathFiles(args RPCArgs, ret *string) error {
	log.Printf("RemoteGetPathFiles...")
	<-d.watcher.Mach.When1(ss.AllRefreshed, nil)
	log.Printf("AllRefreshed...")
	d.watcher.ResultsLock.Lock()
	defer d.watcher.ResultsLock.Unlock()
	*ret += strings.Join(d.watcher.Results, "\n")

	return nil
}

// RemoteExec is an RPC method
func (d *Daemon) RemoteExec(args RPCArgs, ret *string) error {
	log.Printf("RemoteExec...")
	path := args.ExePath
	_, err := d.conn.RunSwayCommand("exec " + path)
	if err != nil {
		return err
	}

	return nil
}

// RemoteWinToSpace is an RPC method
func (d *Daemon) RemoteWinToSpace(args RPCArgs, ret *string) error {
	log.Printf("RemoteWinToSpace...")

	cw := d.FocusedWindow()
	err2 := d.MoveWinToSpaceNum(cw.ID, args.SpaceNum)
	if err2 != nil {
		return err2
	}

	return nil
}

func RemoteCall(method string, args RPCArgs) (string, error) {
	// TODO timeout
	log.Printf("rpcCall %s...", method)

	var err error
	url := rpcHost
	if os.Getenv("YAST_DEBUG") != "" {
		url = rpcHostDbg
	}

	if client == nil {
		client, err = rpc.Dial("tcp", url)
		if err != nil {
			fmt.Println("rpc connection error, is the daemon running?")
			os.Exit(1)
		}
	}
	var reply string
	err = client.Call(method, args, &reply)
	if err != nil {
		return "", err
	}

	return reply, nil
}

// RemoteUsrCmd is an RPC method
func (d *Daemon) RemoteUsrCmd(rpcArgs RPCArgs, rpcRet *string) error {
	log.Printf("RemoteUsrCmd... " + rpcArgs.UsrCmd)

	// init
	var (
		cmdRet string
		err    error
		args   = parseFlags(strings.Trim(rpcArgs.UsrArgs, " \n"))
	)

	// run
	for name, fn := range usrCmds.Registered {
		if name != rpcArgs.UsrCmd {
			continue
		}
		cmdRet, err = fn(d, args)
		break
	}

	// err
	if err != nil {
		log.Printf("error: %s", err)
		return err
	}

	// ret
	log.Printf("cmdRet: %s", cmdRet)
	*rpcRet = cmdRet
	return nil
}

// SERVER

func rpcServer(out *log.Logger, server any) {
	err := rpc.Register(server)

	if err != nil {
		out.Fatal("register error:", err)
	}
	url := rpcHost
	if os.Getenv("YAST_DEBUG") != "" {
		url = rpcHostDbg
	}

	l, err := net.Listen("tcp", url)
	if err != nil {
		out.Fatal("listen error:", err)
	}
	rpc.Accept(l)
}
