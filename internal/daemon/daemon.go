package daemon

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pancsta/sway-yast/pkg/watcher"

	"github.com/Difrex/gosway/ipc"
	"github.com/samber/lo"
)

// CONFIG
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

type WindowData struct {
	ID        int
	Output    string
	Workspace string
	Title     string
	App       string
}

type Daemon struct {
	conn               *ipc.SwayConnection
	MouseFollowsFocus  bool
	watcher            *watcher.PathWatcher
	ctx                context.Context
	winFocus           WindowFocus
	winData            map[string]WindowData
	openedByPID        int
	openedAt           time.Time
	Autoconfig         bool
	DefaultKeybindings bool
	Out 			  *log.Logger
}

///// DAEMON /////

func (d *Daemon) Start() {
	var err error
	d.ctx = context.Background()
	d.winData = make(map[string]WindowData)
	d.watcher, err = watcher.New(d.ctx)
	if err != nil {
		d.Out.Fatalf("error: %s", err)
	}
	// TODO reconnect backoff?
	conn, err := ipc.NewSwayConnection()
	if err != nil {
		d.Out.Fatal(err)
	}
	d.conn = conn

	// read the existing tree to fill out the MRU list
	tree, err := conn.GetTree()
	if err != nil {
		d.Out.Fatal("error:", err)
	}
	for _, output := range tree.Nodes {
		for _, workspace := range output.Nodes {
			for _, container := range workspace.Nodes {
				d.parseNode(&container, workspace.Name, output.Name)
			}
		}
	}

	// set up the window layout
	if d.Autoconfig {
		// TODO support --autoconfig=false
		//   unbind existing and bind our own alt+tab binding
		msgs := []string{
			`for_window [title="sway-yast"] floating enable`,
			`for_window [title="sway-yast"] border none`,
			`for_window [title="sway-yast"] sticky enable`,
		}
		err = d.SwayMsgs(msgs)
		if err != nil {
			d.Out.Fatal("error:", err)
		}
	}

	if d.DefaultKeybindings {
		msgs := []string{
			`bindsym alt+tab exec sway-yast switcher`,
			`bindsym mod4+o exec sway-yast pick-space`,
			`bindsym mod4+p exec sway-yast pick-win`,
			`bindsym mod4+d exec sway-yast path`,
		}
		err = d.SwayMsgs(msgs)
		if err != nil {
			d.Out.Fatal("error:", err)
		}
	}

	// TODO reconnect backoff?
	subCon, err := ipc.NewSwayConnection()
	if err != nil {
		d.Out.Fatal(err)
	}

	// Subscribe only to the window related events
	_, err = subCon.SendCommand(ipc.IPC_SUBSCRIBE, `["window"]`)
	if err != nil {
		d.Out.Fatal(err)
	}

	// Listen for the events
	s := subCon.Subscribe()
	defer s.Close()

	go rpcServer(d.Out, d)
	d.watcher.Start()
	log.Println("Listening for sway events...")

	for {
		select {
		case event := <-s.Events:
			if event.Change == "focus" {
				d.OnFocus(&event.Container)
			}
			// TODO test if event exist, needed when moving to another workspace, find the dest one
			//if event.Change == "blur" {
			//	d.OnBlur(&event.Container)
			//}
			if event.Change == "close" {
				d.OnClose(&event.Container)
			}
		case err := <-s.Errors:
			// TODO reconnect / backoff
			log.Println("Error:", err)
			break
		}
	}
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

func (d *Daemon) parseNode(con *ipc.Node, space, output string) {
	if con.Layout != "splith" && con.Layout != "splitv" && con.Layout != "tabbed" && con.Layout != "stacked" {
		id := strconv.Itoa(int(con.ID))
		data := WindowData{
			ID:        int(con.ID),
			Output:    output,
			Workspace: space,
			Title:     con.Name,
			// TODO AppID
			App: con.WindowProperties.Class,
		}
		d.winData[id] = data
		d.winFocus, _ = unshiftAndTrim(d.winFocus, id)
	}
	for _, node := range con.Nodes {
		d.parseNode(&node, space, output)
	}
}

func (d *Daemon) OnClose(c *ipc.Container) {
	id := strconv.Itoa(c.ID)
	// remove ID from winFocus
	d.winFocus = lo.Without(d.winFocus, id)
	// remove from winData
	delete(d.winData, id)
}

func (d *Daemon) OnFocus(con *ipc.Container) {
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
		ID:        con.ID,
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
		log.Printf("error: %s", err)
	}
}

func (d *Daemon) CurrentWin() WindowData {
	if len(d.winFocus) == 0 {
		return WindowData{}
	}
	id := d.winFocus[0]
	return d.winData[id]
}

func (d *Daemon) FocusWinID(id int) error {
	_, err := d.conn.RunSwayCommand(fmt.Sprintf(`[con_id="%d"] focus`, id))
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

func (d *Daemon) MouseToOutput(output string) error {
	_, err := d.conn.RunSwayCommand(fmt.Sprintf(
		`input 0:0:wlr_virtual_pointer_v1 map_to_output "%s"`, output))
	if err != nil {
		return err
	}
	return nil
}

///// UTILS /////

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
