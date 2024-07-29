package usrCmds

import (
	"log"
)

func init() {
	register("arrange", ArrangeWindows)
}

// ArrangeWindows arranges the windows into desired workspaces.
func ArrangeWindows(d DaemonAPI, _ map[string]string) (string, error) {
	krusader := 0
	firefox := 0

	spaces := struct{ dev, blogic, read string }{
		"1:dev", "2:blogic", "3:read",
	}

	for _, win := range d.ListWindows() {
		log.Printf(`Arrage: #%d:%s "%s"`, win.ID, win.App, win.Title)
		var err error

		// multi space apps
		if d.WinMatchApp(win, "firefox") {
			// skip if already there
			if win.Workspace == spaces.dev || win.Workspace == spaces.blogic {
				continue
			}

			if firefox == 0 {
				err = d.MoveWinToSpace(win.ID, spaces.dev)
			} else {
				err = d.MoveWinToSpace(win.ID, spaces.blogic)
			}
			firefox++
		}

		if d.WinMatchApp(win, "krusader") {
			// skip if already there
			if win.Workspace == spaces.dev || win.Workspace == spaces.blogic {
				continue
			}

			if krusader == 0 {
				err = d.MoveWinToSpace(win.ID, spaces.dev)
			} else {
				err = d.MoveWinToSpace(win.ID, spaces.blogic)
			}
			krusader++
		}

		// 1:dev
		if d.WinMatchApp(win, "jetbrains") {
			err = d.MoveWinToSpace(win.ID, spaces.dev)
		}
		if d.WinMatchTitle(win, "jaeger") {
			err = d.MoveWinToSpace(win.ID, spaces.dev)
		}

		// 2:blogic
		if d.WinMatchApp(win, "obsidian") {
			err = d.MoveWinToSpace(win.ID, spaces.blogic)
		}
		if d.WinMatchTitle(win, "gmail") {
			err = d.MoveWinToSpace(win.ID, spaces.blogic)
		}

		// 3:read
		if d.WinMatchTitle(win, "pocket") {
			err = d.MoveWinToSpace(win.ID, spaces.read)
		}
		if d.WinMatchTitle(win, "inoreader") {
			err = d.MoveWinToSpace(win.ID, spaces.read)
		}
		if d.WinMatchApp(win, "thunderbird") {
			err = d.MoveWinToSpace(win.ID, spaces.read)
		}
		if d.WinMatchApp(win, "discord") {
			err = d.MoveWinToSpace(win.ID, spaces.read)
		}

		if err != nil {
			return "", err
		}
	}

	return "", nil
}
