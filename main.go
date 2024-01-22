package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Difrex/gosway/ipc"
	"github.com/spf13/cobra"
)

// CONFIG
const (
	maxTracked = 100
	lenSpace   = 6
	lenID      = 4
	lenDisplay = 3
	lenApp     = 15
	lenTitle   = 35
	rpcHost    = "localhost:7853"
	// how long a PID can hold the switcher
	pidTimeout = time.Second * 3
	cmdFzf     = `
  fzf \
    --prompt 'Switcher: ' \
    --bind "load:pos(2)" \
    --bind "change:pos(1)" \
    --layout=reverse --info=hidden \
    --bind=space:accept,tab:offset-down,btab:offset-up
`
	// junegunn/seoul256.vim (light)
	cmdFzfLight = ` \
    --color=bg+:#D9D9D9,bg:#E1E1E1,border:#C8C8C8,spinner:#719899,hl:#719872,fg:#616161,header:#719872,info:#727100,pointer:#E12672,marker:#E17899,fg+:#616161,preview-bg:#D9D9D9,prompt:#0099BD,hl+:#719899
`
	cmdSwitcher = `
    foot --title "sway-yast" sway-yast fzf
`
)

// config end

type WindowFocus []string

type WindowData struct {
	Output    string
	Workspace string
	Title     string
	App       string
}

var winFocus WindowFocus
var winData map[string]WindowData

type Daemon struct {
	conn *ipc.SwayConnection
}

func (d *Daemon) Start() {
	winData = make(map[string]WindowData)
	conn, err := ipc.NewSwayConnection()
	if err != nil {
		panic(err)
	}
	d.conn = conn

	// read the existing tree to fill out the MRU list
	tree, err := conn.GetTree()
	if err != nil {
		log.Fatal("error:", err)
	}
	for _, output := range tree.Nodes {
		for _, workspace := range output.Nodes {
			for _, container := range workspace.Nodes {
				d.parseNode(&container, workspace.Name, output.Name)
			}
		}
	}

	// set up the window layout
	_, err = conn.RunSwayCommand(`for_window [title="sway-yast"] floating enable border none`)
	if err != nil {
		log.Fatal("error:", err)
	}

	subCon, err := ipc.NewSwayConnection()
	if err != nil {
		panic(err)
	}

	// Subscribe only to the window related events
	_, err = subCon.SendCommand(ipc.IPC_SUBSCRIBE, `["window"]`)
	if err != nil {
		panic(err)
	}

	// Listen for the events
	s := subCon.Subscribe()
	defer s.Close()

	go rpcServer(d)
	log.Println("Listening for sway events...")

	for {
		select {
		case event := <-s.Events:
			if event.Change == "focus" {
				d.OnFocus(&event.Container)
			}
			if event.Change == "close" {
				d.OnClose(&event.Container)
			}
		case err := <-s.Errors:
			log.Println("Error:", err)
			break
		}
	}
}

func (d *Daemon) parseNode(con *ipc.Node, space, output string) {
	if con.Layout != "splith" && con.Layout != "splitv" && con.Layout != "tabbed" && con.Layout != "stacked" {
		id := strconv.Itoa(int(con.ID))
		data := WindowData{
			Output:    output,
			Workspace: space,
			Title:     con.Name,
			// TODO AppID
			App: con.WindowProperties.Class,
		}
		winData[id] = data
		winFocus, _ = unshiftAndTrim(winFocus, id)
	}
	for _, node := range con.Nodes {
		d.parseNode(&node, space, output)
	}
}

func (d *Daemon) OnClose(c *ipc.Container) {
	id := strconv.Itoa(c.ID)
	// remove ID from winFocus
	for i, v := range winFocus {
		if v == id {
			winFocus = append(winFocus[:i], winFocus[i+1:]...)
			break
		}
	}
	// remove from winData
	delete(winData, id)
}

func (d *Daemon) OnFocus(con *ipc.Container) {
	// TODO debug
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic:", r)
			fmt.Printf("Stack trace: %s\n", debug.Stack())
		}
	}()
	// skip self
	if con.Name == "sway-yast" {
		return
	}
	space, err := d.conn.GetFocusedWorkspace()
	if err != nil {
		log.Printf("error: %s", err)
	}
	id := strconv.Itoa(con.ID)
	data := WindowData{
		Output:    space.Output,
		Workspace: space.Name,
		Title:     con.Name,
	}
	if con.AppID != nil {
		data.App = con.AppID.(string)
	} else if con.WindowProperties.Class != "" {
		data.App = con.WindowProperties.Class
	} else {
		data.App = "unknown"
	}
	winData[id] = data
	var removed []string
	winFocus, removed = unshiftAndTrim(winFocus, id)
	for _, id := range removed {
		delete(winData, id)
	}
}

func main() {
	cmdDaemon := &cobra.Command{
		Use:   "daemon",
		Short: "Start tracking focus in sway",
		Run: func(_ *cobra.Command, _ []string) {
			d := &Daemon{}
			d.Start()
		},
	}

	cmdList := &cobra.Command{
		Use:   "mru-list",
		Short: "Returns a list of MRU window IDs",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(rpcCall("Daemon.WinList", RPCArgs{}))
		},
	}

	cmdFzf := &cobra.Command{
		Use:   "fzf",
		Short: "Render fzf with a list of windows",
		Run:   execFzf,
	}

	cmdSwitcher := &cobra.Command{
		Use:   "switcher",
		Short: "Show the switcher window using foot",
		Run:   execSwitcher,
	}

	var rootCmd = &cobra.Command{Use: "app"}
	rootCmd.AddCommand(cmdDaemon, cmdList, cmdFzf, cmdSwitcher)
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal("cobra error:", err)
	}
}

func execSwitcher(_ *cobra.Command, _ []string) {
	pid := os.Getpid()
	if rpcCall("Daemon.ShouldOpen", RPCArgs{PID: pid}) != "true" {
		log.Fatal("fzf error: already open")
	}
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	err := exec.Command(shell, "-c", cmdSwitcher).Run()
	if err != nil {
		log.Fatal("foot error: " + err.Error())
	}
}

func execFzf(_ *cobra.Command, _ []string) {
	// run fzf
	fzfInput := rpcCall("Daemon.FZFList", RPCArgs{})
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	cmd := cmdFzf
	if isLightMode() {
		cmd = strings.TrimRight(cmd, " \n") + cmdFzfLight
	}
	fzf := exec.Command(shell, "-c", cmd)
	fzf.Stdin = bytes.NewBuffer([]byte(fzfInput))
	// bind the UI
	fzf.Stderr = os.Stderr
	// read the result
	result, err := fzf.Output()
	if err != nil {
		log.Fatal("fzf error: " + err.Error())
	}
	re := regexp.MustCompile(`\((\d+)\)`)
	match := re.FindStringSubmatch(string(result))
	if len(match) == 0 {
		log.Fatal("fzf error: no match")
	}
	// focus the window
	winID, err := strconv.Atoi(match[1])
	if err != nil {
		log.Fatal("fzf error: " + err.Error())
	}
	rpcCall("Daemon.FocusWinID", RPCArgs{WinID: winID})
}

// RPC

type RPCArgs struct {
	WinID int
	PID   int
}

// RPC method
func (d *Daemon) WinList(_ RPCArgs, reply *string) error {
	ids := ""
	for _, id := range winFocus {
		ids += fmt.Sprintf("%s ", id)
	}
	*reply = ids
	return nil
}

// RPC method
func (d *Daemon) FZFList(_ RPCArgs, reply *string) error {
	ret := ""
	for _, id := range winFocus {
		data := winData[id]
		display := strings.Replace(data.Output, "HEADLESS-", "H-", 1)
		ret += fmt.Sprintf("%-*s (%s) %-*s | %-*s | %-*s | %-*s \n",
			lenSpace, maxLen(data.Workspace, lenSpace),
			id, lenID-len(id), " ",
			lenDisplay, maxLen(display, lenDisplay),
			lenApp, maxLen(data.App, lenApp),
			lenTitle, maxLen(data.Title, lenTitle),
		)
	}
	*reply = ret
	return nil
}

var openedByPID int
var openedAt time.Time

// RPC method
func (d *Daemon) ShouldOpen(args RPCArgs, reply *string) error {
	if openedByPID == 0 {
		*reply = "true"
		openedByPID = args.PID
		openedAt = time.Now()
		return nil
	}
	// check if the holding process is alive
	proc, _ := os.FindProcess(openedByPID)
	timeoutOut := time.Now().Sub(openedAt) > pidTimeout
	if dead := proc.Signal(syscall.Signal(0)); dead != nil || timeoutOut {
		openedByPID = args.PID
		openedAt = time.Now()
		*reply = "true"
	} else {
		*reply = "false"
	}
	return nil
}

// RPC method
func (d *Daemon) FocusWinID(args RPCArgs, _ *string) error {
	log.Printf("focusing %d...", args.WinID)
	_, err := d.conn.RunSwayCommand(fmt.Sprintf("[con_id=\"%d\"] focus", args.WinID))
	if err != nil {
		return err
	}
	return nil
}

var client *rpc.Client

func rpcCall(method string, args RPCArgs) string {
	var err error
	if client == nil {
		client, err = rpc.Dial("tcp", rpcHost)
		if err != nil {
			fmt.Println("rpc connection error, is the daemon running?")
			os.Exit(1)
		}
	}
	var reply string
	err = client.Call(method, args, &reply)
	if err != nil {
		log.Fatal("rpc error:", err)
	}
	return reply
}

func rpcServer(server any) {
	err := rpc.Register(server)
	if err != nil {
		log.Fatal("register error:", err)
	}
	l, err := net.Listen("tcp", rpcHost)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	rpc.Accept(l)
}

// Utils

func unshiftAndTrim(slice []string, id string) ([]string, []string) {
	for i, v := range slice {
		if v == id {
			slice = append(slice[:i], slice[i+1:]...)
			break
		}
	}
	ret := append([]string{id}, slice...)
	var removed []string
	if len(ret) > maxTracked {
		removed = ret[maxTracked:]
		ret = ret[:maxTracked]
	}
	return ret, removed
}

func maxLen(str string, maxLength int) string {
	if len(str) > maxLength {
		if len(str) > 4 {
			return str[:maxLength-3] + "..."
		} else {
			return str[:maxLength]
		}
	}
	return str
}

func isLightMode() bool {
	cmd := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "light")
}
