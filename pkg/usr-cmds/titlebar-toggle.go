package usrCmds

func init() {
	register("titlebar-toggle", TitlebarToggle)
}

// TitlebarToggle shows and hides the titlebar of the focused window.
func TitlebarToggle(d DaemonAPI, _ map[string]string) (string, error) {
	cw := d.FocusedWindow()
	path, err := d.GetWinTreePath(cw.ID)
	if err != nil {
		return "", err
	}

	win := path[len(path)-1]

	var border string
	if win.Border == "normal" {
		border = "none"
	} else {
		border = "normal"
	}

	return "", d.SwayMsg(`border %s`, border)
}
