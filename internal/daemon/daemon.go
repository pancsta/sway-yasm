package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pancsta/gosway/ipc"
	"github.com/samber/lo"

	"github.com/pancsta/sway-yasm/internal/types"
	"github.com/pancsta/sway-yasm/internal/watcher"
	usrCmds "github.com/pancsta/sway-yasm/pkg/usr-cmds"
)

// CONFIG TODO file
const (
	maxTracked = 100
	lenSpace   = 8
	lenDisplay = 3
	lenApp     = 15
	lenTitle   = 40
	rpcHost    = "localhost:7853"
	rpcHostDbg = "localhost:7854"
	// how long a PID can hold the switcher
	pidTimeout = time.Second * 3
)

// config end

type WindowFocus []string

type Daemon struct {
	conn               *ipc.SwayConnection
	MouseFollowsFocus  bool
	watcher            *watcher.PathWatcher
	ctx                context.Context
	winFocus           WindowFocus
	winData            map[string]types.WindowData
	openedByPID        int
	openedAt           time.Time
	Autoconfig         bool
	DefaultKeybindings bool
	Logger             *log.Logger
	// current mouse output
	mouseInOutput string
}

// API compat check
var _ usrCmds.DaemonAPI = &Daemon{}

// ///// ///// /////
// ///// DAEMON
// ///// ///// /////

func isClipmanRunning() bool {
	out, err := exec.Command("pgrep", "-a", "wl-paste").Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(out), "clipman")
}

func (d *Daemon) Start() {
	var err error
	d.ctx = context.Background()

	d.winData = make(map[string]types.WindowData)
	d.watcher, err = watcher.New(d.ctx, d.Logger)
	if err != nil {
		d.Logger.Fatalf("error: %s", err)
	}

	// connect
	conn, err := ipc.NewSwayConnection()
	if err != nil {
		d.Logger.Fatal(err)
	}
	d.conn = conn

	// read the existing tree to fill out the MRU list
	tree, err := conn.GetTree()
	if err != nil {
		d.Logger.Fatal("error:", err)
	}
	for _, output := range tree.Nodes {
		for _, workspace := range output.Nodes {
			for _, container := range workspace.Nodes {
				d.parseNode(&container, workspace.Name, output.Name)
			}
		}
	}

	if d.Autoconfig {
		err = d.autoconfig(err)
	}

	if d.DefaultKeybindings {
		err = d.defaultKeybinding(err)
	}

	// subscribe to events
	subCon, err := ipc.NewSwayConnection()
	if err != nil {
		d.Logger.Fatal(err)
	}

	// Subscribe only to the window related events
	_, err = subCon.SendCommand(ipc.IPC_SUBSCRIBE, `["window"]`)
	if err != nil {
		d.Logger.Fatal(err)
	}

	// Listen for the events
	s := subCon.Subscribe()
	defer s.Close()

	go rpcServer(d.Logger, d)
	d.watcher.Start()
	d.Logger.Printf("Listening for sway events...")

	for {
		select {

		case event := <-s.Events:
			if isLog() {
				d.Logger.Printf("Event: %s #%d", event.Change, event.Container.ID)
			}

			if event.Change == "focus" {
				d.onFocus("focus", &event.Container)
			}
			if event.Change == "new" {
				d.onFocus("new", &event.Container)
			}
			if event.Change == "close" {
				d.onClose(&event.Container)
			}

		case err := <-s.Errors:
			// TODO reconnect / backoff
			log.Println("Error:", err)
			os.Exit(1)
		}
	}
}

func (d *Daemon) defaultKeybinding(err error) error {
	var msgs []string
	if isDev() {
		msgs = []string{
			`bindsym alt+tab exec env YASM_DEBUG=1 sway-yasm switcher`,
			`bindsym mod4+o exec env YASM_DEBUG=1 sway-yasm pick-space`,
			`bindsym mod4+p exec env YASM_DEBUG=1 sway-yasm pick-win`,
			`bindsym mod4+d exec env YASM_DEBUG=1 sway-yasm path`,
			`bindsym alt+mod4+c exec env YASM_DEBUG=1 sway-yasm clipboard`,
		}
	} else {

		// TODO support $mod, read config
		msgs = []string{
			`bindsym alt+tab exec sway-yasm switcher`,
			`bindsym mod4+o exec sway-yasm pick-space`,
			`bindsym mod4+p exec sway-yasm pick-win`,
			`bindsym mod4+d exec sway-yasm path`,
			`bindsym alt+mod4+c exec env sway-yasm clipboard`,
		}
	}

	err = d.SwayMsgs(msgs)
	if err != nil {
		d.Logger.Fatal("error:", err)
	}

	return err
}

func (d *Daemon) autoconfig(err error) error {
	msgs := []string{
		`for_window [title="sway-yasm"] floating enable`,
		`for_window [title="sway-yasm"] border none`,
		`for_window [title="sway-yasm"] sticky enable`,
	}
	err = d.SwayMsgs(msgs)
	if err != nil {
		d.Logger.Fatal("error:", err)
	}

	if !isClipmanRunning() {
		d.Logger.Printf("clipman not running, starting...")

		err = d.SwayMsg("exec wl-paste -t text --watch clipman store " +
			"--no-persist --max-items=200")
		if err != nil {
			d.Logger.Fatal("error:", err)
		}
	}

	return err
}

// ListSpaces returns names of the current workspaces.
func (d *Daemon) ListSpaces(skipOutputs []string) ([]string, error) {
	tree, err := d.conn.GetTree()
	if err != nil {
		return nil, err
	}
	var ret = []string{}
	for _, output := range tree.Nodes {
		if lo.Contains(skipOutputs, output.Name) {
			continue
		}
		for _, workspace := range output.Nodes {
			if workspace.Name == "__i3_scratch" {
				continue
			}
			ret = append(ret, workspace.Name)
		}
	}
	return ret, nil
}

// GetWinTreePath returns the nodes between tree root and the passed window ID,
// starting with the workspace
func (d *Daemon) GetWinTreePath(id int) ([]*ipc.Node, error) {
	tree, err := d.conn.GetTree()
	if err != nil {
		return nil, err
	}

	for i := range tree.Nodes {
		for ii := range tree.Nodes[i].Nodes {
			workspace := &tree.Nodes[i].Nodes[ii]

			if workspace.Name == "__i3_scratch" {
				continue
			}

			_, path := findPathToRoot(workspace, int64(id), nil)
			if path != nil {
				slices.Reverse(path)
				return path, nil
			}
		}
	}

	return nil, nil
}

// findPathToRoot searches for the target node and returns the path from the target to the root
func findPathToRoot(node *ipc.Node, targetID int64, path []*ipc.Node) (bool, []*ipc.Node) {
	// Add the current node to the path
	path = append([]*ipc.Node{node}, path...)

	// Check if the current node is the target
	if node.ID == targetID {
		return true, path
	}

	// Recursively search in the children
	for i := range node.Nodes {
		found, p := findPathToRoot(&node.Nodes[i], targetID, path)
		if found {
			return true, p
		}
	}

	// Target not found in this subtree
	return false, nil
}

func (d *Daemon) parseNode(con *ipc.Node, space, output string) {
	isWin := con.Layout != "splith" && con.Layout != "splitv" &&
		con.Layout != "tabbed" && con.Layout != "stacked"

	if isWin {
		id := strconv.Itoa(int(con.ID))
		data := types.WindowData{
			ID:        int(con.ID),
			Output:    output,
			Workspace: space,
			Title:     con.Name,
			App:       con.WindowProperties.Class,
			Rect:      con.Rect,
		}
		if con.AppID != nil {
			data.App = con.AppID.(string)
		}

		d.winData[id] = data
		d.winFocus, _ = unshiftAndTrim(d.winFocus, id)
	}

	for _, node := range con.Nodes {
		d.parseNode(&node, space, output)
	}
}

func (d *Daemon) onClose(c *ipc.Container) {
	id := strconv.Itoa(c.ID)

	// remove ID from winFocus
	d.winFocus = lo.Without(d.winFocus, id)

	// remove from winData
	data := d.winData[id]
	delete(d.winData, id)

	// run user scripts
	for _, l := range usrCmds.Listeners["close"] {
		l.WinListenerFunc(d, data)
	}
}

func (d *Daemon) onFocus(event string, con *ipc.Container) {
	// TODO event enum

	// skip self
	if con.Name == "sway-yasm" {
		return
	}

	space, err := d.conn.GetFocusedWorkspace()
	if err != nil {
		d.Logger.Printf("error: %s", err)
	}

	// update win data
	id := strconv.Itoa(con.ID)
	data := types.WindowData{
		ID:        con.ID,
		Output:    space.Output,
		Workspace: space.Name,
		Title:     con.Name,
		Rect:      con.Rect,
		App:       con.WindowProperties.Class,
	}
	if con.AppID != nil {
		data.App = con.AppID.(string)
	}
	d.winData[id] = data

	var removed []string
	d.winFocus, removed = unshiftAndTrim(d.winFocus, id)
	for _, id := range removed {
		delete(d.winData, id)
	}

	// move the pointer
	if !d.MouseFollowsFocus {
		return
	}
	err = d.MouseToOutput(data.Output)
	if err != nil {
		d.Logger.Printf("error: %s", err)
	}

	// run user scripts
	for _, l := range usrCmds.Listeners[event] {
		l.WinListenerFunc(d, data)
	}
}

func (d *Daemon) FocusedWindow() types.WindowData {
	if len(d.winFocus) < 1 {
		return types.WindowData{}
	}
	id := d.winFocus[0]

	return d.winData[id]
}

func (d *Daemon) PrevWindow() types.WindowData {
	if len(d.winFocus) < 2 {
		return types.WindowData{}
	}
	id := d.winFocus[1]

	return d.winData[id]
}

func (d *Daemon) FocusWinID(id int) error {
	err := d.SwayMsg(`[con_id=%d] focus`, id)
	if err != nil {
		return err
	}

	return nil
}

func (d *Daemon) SwayMsgs(msgs []string) error {
	for _, msg := range msgs {
		_, err := d.conn.RunSwayCommand(msg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Daemon) SwayMsg(msg string, args ...any) error {
	cmd := fmt.Sprintf(msg, args...)

	if isLog() {
		d.Logger.Printf("swaymsg %s", cmd)
	}
	_, err := d.conn.RunSwayCommand(cmd)
	if err != nil {
		return err
	}

	return nil
}

func (d *Daemon) MouseToOutput(output string) error {
	if d.mouseInOutput == output {
		return nil
	}

	_, err := d.conn.RunSwayCommand(fmt.Sprintf(
		`input 0:0:wlr_virtual_pointer_v1 map_to_output "%s"`, output))
	if err != nil {
		return err
	}
	d.mouseInOutput = output

	return nil
}

func (d *Daemon) ListWindows() map[string]types.WindowData {
	return d.winData
}

func (d *Daemon) MoveWinToSpaceNum(winID, spaceNum int) error {
	name, err := d.spaceNameFromID(spaceNum)
	if err != nil {
		return err
	}

	return d.MoveWinToSpace(winID, name)
}

func (d *Daemon) MoveWinToSpace(winID int, space string) error {
	winIDStr := strconv.Itoa(winID)
	win := d.winData[winIDStr]

	if win.Workspace == space {
		// skip already there
		return nil
	}

	err := d.SwayMsg("[con_id=%d] move to workspace %s", winID, space)
	if err != nil {
		return err
	}

	// set the new workspace
	win.Workspace = space
	d.winData[winIDStr] = win

	return nil
}

func (d *Daemon) spaceNameFromID(spaceID int) (string, error) {
	spaceIDStr := strconv.Itoa(spaceID)
	names, err := d.ListSpaces([]string{})
	if err != nil {
		return "", nil
	}

	for _, name := range names {
		if strings.HasPrefix(name, spaceIDStr+":") {
			return name, nil
		}

	}

	return "", errors.New("ws not found")
}

func (d *Daemon) MoveSpaceToOutput(space, output string, focusedWinData types.WindowData) error {
	d.Logger.Printf("moving space %s to %s", space, output)
	msgs := []string{
		fmt.Sprintf("workspace %s", space),
		fmt.Sprintf("move workspace to output %s", output),
	}
	err := d.SwayMsgs(msgs)
	if err != nil {
		d.Logger.Printf("error: %s", err)
		return err
	}

	time.Sleep(100 * time.Millisecond)

	// focus the original window back
	err = d.FocusWinID(focusedWinData.ID)
	if err != nil {
		d.Logger.Printf("error: %s", err)
		return err
	}
	if d.MouseFollowsFocus {
		err = d.MouseToOutput(focusedWinData.Output)
		if err != nil {
			d.Logger.Printf("error: %s", err)
			return err
		}
	}

	return nil
}

func (d *Daemon) WinMatchApp(win types.WindowData, match string) bool {
	return strings.Contains(strings.ToLower(win.App), strings.ToLower(match))
}

func (d *Daemon) WinMatchTitle(win types.WindowData, match string) bool {
	return strings.Contains(strings.ToLower(win.Title), strings.ToLower(match))
}

func (d *Daemon) HandlerOnFocus(func(types.WindowData)) {
	// TODO
}

func (d *Daemon) HandlerOnClose(func(types.WindowData)) {
	// TODO
}

func (d *Daemon) HandlerOnNew(func(types.WindowData)) {
	// TODO
}

// ///// ///// /////
// ///// UTILS
// ///// ///// /////

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

func IsLightMode() bool {
	cmd := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "light")
}

func isLog() bool {
	return os.Getenv("YASM_LOG") != ""
}

func isDev() bool {
	return os.Getenv("YASM_DEBUG") != ""
}

// parseFlags parses a string of flags into a map
// input: 23 -a --b=4 foo=2 -bar=1
// output: map[123: a: b:4 bar:1 foo:2]
func parseFlags(input string) map[string]string {
	flags := strings.Split(input, " ")
	flagMap := make(map[string]string)

	for _, flag := range flags {
		prefix1 := strings.HasPrefix(flag, "--")
		prefix2 := strings.HasPrefix(flag, "-")
		equals := strings.Index(flag, "=") > 0

		if prefix1 || prefix2 || equals {
			parts := strings.SplitN(flag, "=", 2)
			if len(parts) == 2 {
				flagMap[parts[0]] = parts[1]
			} else {
				flagMap[parts[0]] = ""
			}
		} else {
			// index flag
			flagMap[flag] = ""
		}
	}

	return flagMap
}
