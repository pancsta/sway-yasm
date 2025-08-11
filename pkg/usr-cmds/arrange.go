package usrCmds

// TODO declarative arrangement (optional)

import (
	"errors"
	"log"
	"strings"
)

func init() {
	register("arrange", ArrangeWindows)
}

var disableFloating = `[con_id=%d] floating disable`

// ArrangeWindows arranges the windows into desired workspaces.
func ArrangeWindows(d DaemonAPI, _ map[string]string) (string, error) {
	// firefox on space 1, 2, and 4
	firefox := 0
	// goland on space 1 and 4
	goland := 0

	spaces := struct{ dev, blogic, read, dev2, sidecar1, sidecar2 string }{
		"1:dev", "2:blogic", "3:read", "4:dev2", "5:sidecar1", "6:sidecar2",
	}

	for _, win := range d.ListWindows() {
		log.Printf(`Arrage: #%d:%s "%s"`, win.ID, win.App, win.Title)
		var err error

		// firefox on space 1, 2, and 4
		if d.WinMatchApp(win, "firefox") {
			if win.Workspace != spaces.dev && win.Workspace != spaces.dev2 &&
				win.Workspace != spaces.blogic {

				if firefox == 0 {
					err = d.MoveWinToSpace(win.ID, spaces.dev)
				} else if firefox == 1 {
					err = d.MoveWinToSpace(win.ID, spaces.dev2)
				} else {
					err = d.MoveWinToSpace(win.ID, spaces.blogic)
				}
			}
			firefox++
		}

		// goland on space 1 and 4
		if d.WinMatchApp(win, "jetbrains-go") && strings.HasSuffix(win.Title, "]") {

			if win.Workspace != spaces.dev && win.Workspace != spaces.dev2 {

				if goland == 0 {
					err = d.MoveWinToSpace(win.ID, spaces.dev)
				} else if goland == 1 {
					err = d.MoveWinToSpace(win.ID, spaces.dev2)
				}
			}
			goland++
		}

		// 1:dev
		space := spaces.dev
		// ...

		// 2:blogic
		space = spaces.blogic
		if d.WinMatchApp(win, "obsidian") ||
			d.WinMatchTitle(win, "gmail") {
			err = d.MoveWinToSpace(win.ID, space)

		} else if d.WinMatchMark(win, "chromium-default") ||
			d.WinMatchMark(win, "chromium-google") {
			err = d.MoveWinToSpace(win.ID, space)
		}

		// 3:read
		space = spaces.read
		// if d.WinMatchTitle(win, "slack") ||
		if d.WinMatchTitle(win, "element") ||
			d.WinMatchTitle(win, "telegram") ||
			d.WinMatchApp(win, "thunderbird") ||
			d.WinMatchApp(win, "discord") ||
			d.WinMatchMark(win, "chromium-read") {

			err = d.MoveWinToSpace(win.ID, space)
		}

		// 4:dev2
		space = spaces.dev2
		if d.WinMatchApp(win, "jetbrains-py") {
			err = d.MoveWinToSpace(win.ID, space)

		} else if d.WinMatchApp(win, "Cursor") {
			err = d.MoveWinToSpace(win.ID, space)
		}

		// 5:sidecar1
		space = spaces.sidecar1
		if d.WinMatchApp(win, "jetbrains-idea") ||
			d.WinMatchApp(win, "krusader") ||
			d.WinMatchTitle(win, "Gemini") ||
			d.WinMatchMark(win, "chromium-dev") {

			err = d.MoveWinToSpace(win.ID, space)

			// jetbrains
		} else if d.WinMatchApp(win, "jetbrains-go") {
			if d.WinMatchTitle(win, "Find") ||
				d.WinMatchTitle(win, "Services") ||
				d.WinMatchTitle(win, "Run") ||
				strings.HasPrefix(win.Title, "Debug") ||
				strings.HasSuffix(win.Title, "Database Sessions") ||
				strings.HasSuffix(win.Title, "AI Chat") {

				// move and un-float
				err = errors.Join(
					d.MoveWinToSpace(win.ID, space),
					d.SwayMsg(disableFloating, win.ID),
				)
			}
		}

		// err
		if err != nil {
			return "", err
		}
	}

	// ok
	return "", nil
}
