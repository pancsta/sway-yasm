# sway-yasm

Sway's **y**et **a**nother **s**way **m**anager is a daemon for managing [Sway WM's](https://github.com/swaywm/sway) windows, workspaces, outputs, clipboard and PATH using [FZF](https://github.com/junegunn/fzf) - both as floating foot windows and in the terminal.

It tries to deliver all these features in one command, without any configuration, and with a single binary, so it can be deployed easily:

>`sway-yasm daemon --default-keybindings`

<table>

  <tr>
    <td align="center">switcher</td>
    <td align="center">path</td>
  </tr>
  <tr>
    <td align="center">
        <img src="assets/switcher.dark.png#gh-dark-mode-only" alt="switcher"/>
        <img src="assets/switcher.light.png#gh-light-mode-only" alt="switcher"/>
    </td>
    <td align="center">
        <img src="assets/path.dark.png#gh-dark-mode-only" alt="path"/>
        <img src="assets/path.light.png#gh-light-mode-only" alt="path"/>
    </td>
  </tr>

  <tr>
    <td align="center">pick-space</td>
    <td align="center">pick-win</td>
  </tr>
  <tr>
    <td align="center">
        <img src="assets/pick-space.dark.png#gh-dark-mode-only" alt="pick-space"/>
        <img src="assets/pick-space.light.png#gh-light-mode-only" alt="pick-space"/>
    </td>
    <td align="center">
        <img src="assets/pick-win.dark.png#gh-dark-mode-only" alt="pick-win"/>
        <img src="assets/pick-win.light.png#gh-light-mode-only" alt="pick-win"/>
    </td>
  </tr>

  <tr>
    <td align="center">clipboard</td>
    <td align="center"></td>
  </tr>
  <tr>
    <td align="center">
        <img src="assets/clipboard.dark.png#gh-dark-mode-only" alt="clipboard"/>
        <img src="assets/clipboard.light.png#gh-light-mode-only" alt="clipboard"/>
    </td>
    <td align="center">
    </td>
  </tr>
</table>

## install

Install using one of the following ways:

- binary from [the releases page](https://github.com/pancsta/sway-yasm/releases/latest)
- `go install github.com/pancsta/sway-yasm@latest`
- `git clone; cd; go mod tidy; ./scripts/build`

## features

- window / workspace management
  - alt+tab / MRU order for windows
  - move a workspace to the current output
  - move a window to the current workspace
- miscellaneous management
  - run anything in your `PATH`
  - copy from clipboard history using `clipman` and `wl-clipboard`
- [user command files](#user-command-files) (scripts)
  - resize-toggle
  - arrange
  - titlebar-toggle
- daemon (IPC & RPC) architecture, filesystem-free
- uses `fzf`, so renders in the terminal
- shows a floating window using `foot`
- dark mode support<br />
  checks `gsettings get org.gnome.desktop.interface color-scheme`
- 1-hand keystrokes for window switching
- [mouse follows focus](#mouse-follows-focus) mode (optional)
- plain MRU list via `mru-list` for integrations

## usage

1. Start the daemon<br />
   `sway-yasm daemon --default-keystrokes`
2. press: `alt+tab`
3. term: `sway-yasm fzf switcher`
4. term: `sway-yasm --help`
5. see: [default keystrokes](#default-keystrokes)

## help

```text
$ sway-yasm --help

Usage:
  sway-yasm [flags]
  sway-yasm [command]

Available Commands:
  completion     Generate the autocompletion script for the specified shell
  config         Change the config of a running daemon process
  daemon         Start tracking focus in sway
  fzf            Pure FZF versions of the switcher and pickers
  help           Help about any command
  mru-list       Print a list of MRU window IDs
  path           Show the +x files from PATH using foot
  pick-clipboard Set the clipboard contents from the history
  pick-space     Show the workspace picker using foot
  pick-win       Show the window picker using foot
  switcher       Show the switcher window using foot
  usr-cmd        Run a user command with a specific name and optional args
  win-to-space   Move the current window to a specific workspace

Flags:
  -h, --help      help for sway-yasm
      --version   Print version and exit

Use "sway-yasm [command] --help" for more information about a command.
```

```text
$ sway-yasm daemon --help

Start tracking focus in sway

Usage:
  sway-yasm daemon [flags]

Flags:
      --autoconfig            Automatically configure the layout and start clipman (default true)
      --default-keybindings   Add default keybindings
  -h, --help                  help for daemon
      --mouse-follows-focus   Calls 'input ... map_to_output OUTPUT' on each focus
```

## keystrokes

### window switcher

Normal mode:

- `alt+tab` show the switcher, preselect the previous window, enter `Switcher` mode

Switcher mode:

- `space` focus the selected window, close the switcher
- `enter` focus the selected window, close the switcher
- `tab` select the next window in the list
- `down` select the next window in the list
- `shift+tab` select the previous window in the list
- `up` select the previous window in the list
- `esc` close the switcher
- `ctrl+c` close the switcher
- `a-z`, `0-9` fuzzy search

Example - switch to the 3rd MRU window:

- `alt+tab`
- `tab`
- `space`

Example - switch to Krusader by name:

- `alt+tab`
- `k`, `r`, `u`
- `enter`

### default keystrokes

Various ways to get the default keybindings.

```bash
$ sway-yasm daemon --default-keybindings
```

```bash
# shell
swaymsg bindsym alt+tab exec sway-yasm switcher
swaymsg bindsym mod4+o exec sway-yasm pick-space
swaymsg bindsym mod4+p exec sway-yasm pick-win
swaymsg bindsym mod4+d exec sway-yasm path
swaymsg bbindsym mod4+alt+c exec sway-yasm clipboard
```

```text
# config
bindsym alt+tab exec sway-yasm switcher
bindsym $mod+o exec sway-yasm pick-space
bindsym $mod+p exec sway-yasm pick-win
bindsym $mod+d exec sway-yasm path
bindsym $mod+alt+c exec sway-yasm clipboard
```

### simulate blur events

```text
# pass `container move to workspace number` via sway-yasm
# as IPC doesnt offer the blur event

bindsym $mod+Control+1 exec sway-yasm win-to-space 1
bindsym $mod+Control+2 exec sway-yasm win-to-space 2
bindsym $mod+Control+3 exec sway-yasm win-to-space 3
bindsym $mod+Control+4 exec sway-yasm win-to-space 4
bindsym $mod+Control+5 exec sway-yasm win-to-space 5
bindsym $mod+Control+6 exec sway-yasm win-to-space 6
bindsym $mod+Control+7 exec sway-yasm win-to-space 7
bindsym $mod+Control+8 exec sway-yasm win-to-space 8
bindsym $mod+Control+9 exec sway-yasm win-to-space 9
bindsym $mod+Control+0 exec sway-yasm win-to-space 10
```

## mouse follows focus

```bash
$ sway-yasm daemon --mouse-follows-focus
```

Using `input map_to_output`, the daemon traps the relative cursor inside the currently focused output. Changing focus moves the cursor between outputs (thus the name). Useful for VNC screens on separate machines. When combined with [waycorner](https://github.com/AndreasBackx/waycorner), it creates a synergy-like effect.

Turning on/off:

```bash
$ sway-yasm config --mouse-follows-focus=false
$ sway-yasm config --mouse-follows-focus=true
```

### waycorner config example

```toml
# HEADLESS-1 (right screen)
[pro5-left]
enter_command = [ "sway-pointer-output", "2" ]
locations = ["left"]
[pro5-left.output]
description = ".*output 1.*"

# HEADLESS-2 (left screen)
[mini6-right]
enter_command = [ "sway-pointer-output", "1" ]
locations = ["right"]
[mini6-right.output]
description = ".*output 2.*"
```

## configuration

[No yaml yet](#todo), but check headings of these files:

- [daemon.go](internal/daemon/daemon.go)
- [fzf.go](internal/cmds/fzf.go)

## troubleshooting

`env YASM_LOG=1 sway-yasm`

## development

- `./scripts/build.sh`
- `env YASM_LOG=1 YASM_DEBUG=1 ./sway-yasm deamon`
- `env YASM_LOG=1 YASM_DEBUG=1 ./sway-yasm switcher`

## user command files

User command files provide a simple way to **script sway using Go within the daemon**, and can be fairly easily exchanged with others.

- [resize-toggle](pkg/usr-cmds/resize-toggle.go)
  - `sway-yasm usr-cmd resize-toggle`
  - resizes a split to 10/50/90%
- [arrange](pkg/usr-cmds/arrange.go)
  - `sway-yasm usr-cmd arrange`
  - arranges windows to workspaces
- [titlebar-toggle](pkg/usr-cmds/titlebar-toggle.go)
  - `sway-yasm usr-cmd titlebar-toggle`
  - shows/hides window's titlebar

Installing a user command file:

```shell
cp my-cmd.go pkg/usr-cmds
./scripts/build.sh
# run
./sway-yasm deamon
./sway-yasm usr-cmd my-cmd 123 -- -a --b=c
```

Modifying a user command file:

See [pkg/usr-cmds/api.go](pkg/usr-cmds/api.go) for the API and [pkg/usr-cmds/template.go](pkg/usr-cmds/template.go) for a sample usage.

```shell
nano pkg/usr-cmds/arrange.go
./scripts/build.sh
# run
./sway-yasm deamon
./sway-yasm usr-cmd arrange
```

## todo

- yaml config file
- user scripts in wasm
- underscore windows from the current workspace
- show on all screens (via wayland)
- pick grouping containers with `pick-container`
- `switcher --current-output-only`
- `switcher --current-space-only`
- `switcher --group-by-output`
- tests (wink wink)
- themes
- reconnect logic

## changelog

See [CHANGELOG.md](CHANGELOG.md).

## kudos

- [applist.py](https://github.com/davxy/dotfiles/blob/main/_old/sway/applist.py)
- [sway-fzfify](https://github.com/ldelossa/sway-fzfify)
- [Difrex/gosway](https://github.com/Difrex/gosway)
- [fzf](https://github.com/junegunn/fzf)
- [contexts](https://contexts.co/)
]()