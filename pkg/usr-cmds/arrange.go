package usrCmds

import (
	"log"
)

func init() {
	register("arrange", ArrangeWindows)
}

// ArrangeWindows arranges the windows into desired workspaces.
// TODO mark windows
func ArrangeWindows(d DaemonAPI, _ map[string]string) (string, error) {
	firefox := 0
	chromium := 0

	spaces := struct{ dev, blogic, read, sidecar1, sidecar2 string }{
		"1:dev", "2:blogic", "3:read", "5:siecar1", "5:siecar2",
	}

	for _, win := range d.ListWindows() {
		log.Printf(`Arrage: #%d:%s "%s"`, win.ID, win.App, win.Title)
		var err error

		// firefox on space 1 and 2
		if d.WinMatchApp(win, "firefox") {
			if firefox == 0 {
				err = d.MoveWinToSpace(win.ID, spaces.dev)
			} else {
				err = d.MoveWinToSpace(win.ID, spaces.blogic)
			}
			firefox++
		}

		// one chromium per space
		if d.WinMatchApp(win, "chromium") && !d.WinMatchTitle(win, "gmail") {
			switch chromium {
			case 0:
				err = d.MoveWinToSpace(win.ID, spaces.sidecar1)
			case 1:
				err = d.MoveWinToSpace(win.ID, spaces.blogic)
			case 2:
				err = d.MoveWinToSpace(win.ID, spaces.read)
			}
			chromium++
		}

		// 1:dev
		if d.WinMatchApp(win, "jetbrains-go") {
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
		if d.WinMatchTitle(win, "slack") {
			err = d.MoveWinToSpace(win.ID, spaces.read)
		}
		if d.WinMatchTitle(win, "element") {
			err = d.MoveWinToSpace(win.ID, spaces.read)
		}
		if d.WinMatchTitle(win, "telegram") {
			err = d.MoveWinToSpace(win.ID, spaces.read)
		}
		if d.WinMatchApp(win, "thunderbird") {
			err = d.MoveWinToSpace(win.ID, spaces.read)
		}
		if d.WinMatchApp(win, "discord") {
			err = d.MoveWinToSpace(win.ID, spaces.read)
		}

		// 5:sidecar1
		if d.WinMatchApp(win, "jetbrains-id") {
			err = d.MoveWinToSpace(win.ID, spaces.sidecar1)
		}
		if d.WinMatchApp(win, "krusader") {
			err = d.MoveWinToSpace(win.ID, spaces.sidecar1)
		}

		// 6:sidecar2
		if d.WinMatchApp(win, "jetbrains-py") {
			err = d.MoveWinToSpace(win.ID, spaces.sidecar2)
		}

		if err != nil {
			return "", err
		}
	}

	return "", nil
}
