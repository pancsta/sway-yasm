package usrCmds

func init() {
	register("template", Template)
	// onFocus(func(api DaemonAPI, win types.WindowData) {
	// 	fmt.Println("template.focus")
	// })
	// onClose(func(api DaemonAPI, win types.WindowData) {
	// 	fmt.Println("template.close")
	// })
	// onNew(func(api DaemonAPI, win types.WindowData) {
	// 	fmt.Println("template.new")
	// })
	// onCopy(func(api DaemonAPI, text string) string {
	// 	fmt.Println("template.copy")
	// })
}

// Template is a template for creating new user commands, with some API examples.
func Template(d DaemonAPI, args map[string]string) (string, error) {
	win := d.FocusedWindow()
	path, err := d.GetWinTreePath(win.ID)
	if err != nil {
		return "", err
	}

	p("Focused window: %d", win.Title)
	p("Focused workspace: %d", path[0].Name)
	inspect(args)

	return "cli output", d.SwayMsg(`exec echo %d`, win.ID)
}
