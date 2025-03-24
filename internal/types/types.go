package types

import "github.com/pancsta/gosway/ipc"

type WindowData struct {
	ID int
	// eg HEADLESS-1
	Output    string
	Workspace string
	Title     string
	App       string
	Mark      string
	Rect      ipc.Rect
}
