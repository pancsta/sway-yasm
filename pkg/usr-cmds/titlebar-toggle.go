package usrCmds

func init() {
	register("titlebar-toggle", TitlebarToggle)
}

// TitlebarToggle is a template for creating new user commands, with some API examples.
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
