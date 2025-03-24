package usrCmds

import (
	"errors"
	"log"
	"strings"
)

func init() {
	register("arrange", ArrangeWindows)
}

// ArrangeWindows arranges the windows into desired workspaces.
// TODO mark windows
func ArrangeWindows(d DaemonAPI, _ map[string]string) (string, error) {
	firefox := 0

	spaces := struct{ dev, blogic, read, sidecar1, sidecar2 string }{
		"1:dev", "2:blogic", "3:read", "5:sidecar1", "6:sidecar2",
	}

	for _, win := range d.ListWindows() {
		log.Printf(`Arrage: #%d:%s "%s"`, win.ID, win.App, win.Title)
		var err error

		// firefox on space 1 and 2
		if d.WinMatchApp(win, "firefox") {
			if win.Workspace != spaces.dev && win.Workspace != spaces.blogic {
				if firefox == 0 {
					err = d.MoveWinToSpace(win.ID, spaces.dev)
				} else {
					err = d.MoveWinToSpace(win.ID, spaces.blogic)
				}
			}
			firefox++
		}

		// 1:dev

		if d.WinMatchApp(win, "jetbrains-go") && strings.HasSuffix(win.Title, "]") {
			err = d.MoveWinToSpace(win.ID, spaces.dev)
		}

		// 2:blogic

		if d.WinMatchApp(win, "obsidian") ||
			d.WinMatchTitle(win, "gmail") {
			err = d.MoveWinToSpace(win.ID, spaces.blogic)
		}
		if d.WinMatchMark(win, "chromium-default") ||
			d.WinMatchMark(win, "chromium-google") {
			err = d.MoveWinToSpace(win.ID, spaces.dev)
		}

		// 3:read

		if d.WinMatchTitle(win, "slack") ||
			d.WinMatchTitle(win, "element") ||
			d.WinMatchTitle(win, "telegram") ||
			d.WinMatchApp(win, "thunderbird") ||
			d.WinMatchApp(win, "discord") ||
			d.WinMatchMark(win, "chromium-read") {

			err = d.MoveWinToSpace(win.ID, spaces.read)
		}

		// 5:sidecar1

		if d.WinMatchApp(win, "jetbrains-id") ||
			d.WinMatchApp(win, "krusader") ||
			d.WinMatchMark(win, "chromium-dev") {

			err = d.MoveWinToSpace(win.ID, spaces.sidecar1)
		}
		if d.WinMatchApp(win, "jetbrains-go") {

			if d.WinMatchTitle(win, "AIAssistant") ||
				d.WinMatchTitle(win, "Git") ||
				d.WinMatchTitle(win, "Find") ||
				d.WinMatchTitle(win, "Debug") {

				err = errors.Join(
					d.MoveWinToSpace(win.ID, spaces.sidecar1),
					d.SwayMsg(`[con_id=%d] floating disable`, win.ID),
				)
			}
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
