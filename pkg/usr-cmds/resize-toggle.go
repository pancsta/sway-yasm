package usrCmds

func init() {
	register("resize-toggle", ResizeToggle)
}

// ResizeToggle toggles the width of the top-most split parent of the focused
// window, between 10-50-90% of the screen width.
func ResizeToggle(d DaemonAPI, _ map[string]string) (string, error) {
	path, err := d.GetWinTreePath(d.FocusedWindow().ID)
	if err != nil {
		return "", err
	}

	space := path[0]
	split := path[1]
	// skip non-split splits
	for i := 2; split.Rect.Width == space.Rect.Width && i < len(path); i++ {
		split = path[i]
	}

	halfWidth := space.Rect.Width / 2
	maxWidth := space.Rect.Width
	targetWidth := int(0.9 * float32(maxWidth))

	var ppt string
	if split.Rect.Width == halfWidth {
		ppt = "90ppt"
	} else if split.Rect.Width < targetWidth {
		ppt = "50ppt"
	} else {
		ppt = "10ppt"
	}

	return ppt, d.SwayMsg(`[con_id=%d] resize set width %s`, split.ID, ppt)
}
