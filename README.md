# dock-mango

A fork of nwg-dock-hyprland with different improvements and changes. Adapted for mangowm 

<img width="1920" height="1080" alt="image" src="https://github.com/user-attachments/assets/352f18ab-56d3-4339-a31d-187545d671eb" />


<img width="594" height="717" alt="image" src="https://github.com/user-attachments/assets/6fd6bd93-be5e-4ab8-867c-a834d7f8c494" />


## Differences from nwg-dock-hyprland:

- No per monitor support. You can't set hotspot / dock per specific monitor :(
- If using `-d` (auto hide) dock is revealed when left clicked, not hovered
- Support for `-hl bottom` hotspot is layered beneath any window
- Different default css and icons
- Misc changes, hover fixes, auto hide fixes, different window actions arrangement

## Installation

### Requirements

- `go`>=1.20 (just to build)
- `gtk3`
- `gtk-layer-shell`
- [nwg-drawer](https://github.com/nwg-piotr/nwg-drawer): optionally. You may use another launcher (see help),
or none at all. The launcher button won't show up, if so.

### Steps

1. Clone the repository, cd into it.
2. Install golang libraries with `make get`. First time it may take ages, be patient.
3. `make build`
4. `sudo make install`

## Running

Either start the dock permanently in `mango/config.conf`:

```conf
exec-once = dock-mango [arguments]
```

or assign the command to some key binding. Running the command again kills the existing program instance, so that
you could use the same key to open and close the dock.

## Running the dock residently

If you run the program with the `-d` or `-r` argument (preferably in autostart), it will be running residently.

```text
exec_always dock-mango -d
```

or

```text
exec_always dock-mango -r
```

### `-d` for autohiDe

Move the mouse pointer to expected dock location and left click for the dock to show up. It will be hidden a second after you leave the
window. Invisible hot spots will be created on all your outputs.

### `-r` for just Resident

No hotspot will be created. To show/hide the dock, bind the `exec dock-mango` command to some key or button.
How about the `Menu` key, which is usually useless?

Re-execution of the same command hides the dock. If a resident instance found, the `dock-mango` command w/o
arguments sends `SIGUSR1` to it. Actually `pkill -USR1 dock-mango` could be used instead. This also works in autohiDe
mode.

Re-execution of the command with the `-d` or `-r` argument won't kill the running instance. If the dock is
running residently, another instance will just exit with 0 code. In case you'd like to terminate it anyway, you need 
to `pkill -f dock-mango`.

*NOTE: you need to kill the running instance before reloading Hyprland, if you've just changed the arguments you
auto-start the dock with.*

```txt
$ dock-mango -h
Usage of dock-mango:
  -a string
        Alignment in full width/height: "start", "center" or "end" (default "center")
  -c string
        Command assigned to the launcher button (default "nwg-drawer")
  -d    auto-hiDe: show dock when hotspot hovered, close when left or a button clicked
  -debug
        turn on debug messages
  -f    take Full screen width/height
  -g string
        quote-delimited, space-separated class list to iGnore in the dock
  -hd int
        Hotspot Delay [ms]; the smaller, the faster mouse pointer needs to enter hotspot for the dock to appear; set 0 to disable (default 20)
  -hl string
        Hotspot Layer "overlay", "top" or "bottom" (default "overlay")
  -i int
        Icon size (default 48)
  -ico string
        alternative name or path for the launcher ICOn
  -l string
        Layer "overlay", "top" or "bottom" (default "overlay")
  -lp string
        Launcher button position, 'start' or 'end' (default "end")
  -m    allow Multiple instances of the dock (skip lock file check)
  -mb int
        Margin Bottom
  -ml int
        Margin Left
  -mr int
        Margin Right
  -mt int
        Margin Top
  -nolauncher
        don't show the launcher button
  -p string
        Position: "bottom", "top" "left" or "right" (default "bottom")
  -r    Leave the program resident, but w/o hotspot
  -s string
        Styling: css file name (default "style.css")
  -v    display Version information
  -w int
        number of Workspaces you use (default 10)
  -x    set eXclusive zone: move other windows aside; overrides the "-l" argument

Usage of signals:
 SIGRTMIN+1 (signal 35): toggle dock visibility (USR1 has been deprecated)
 SIGRTMIN+2 (signal 36): show the dock
 SIGRTMIN+3 (signal 37): hide the dock
```

## Styling

Edit `~/.config/nwg-dock-hyprland/style.css` to your taste. (Yes, styles are located in `~/.config/nwg-dock-hyprland/style.css`, not `~/.config/dock-mango`)

## Troubleshooting

### An application icon is not displayed

The only thing the dock knows about the app is it's class name.

```text
$ mmsg get all-clients
(...)
{
	"id": 36,
	"pid": 84624,
	"foreign_toplevel_id": "3ee55c1a16b458ef49a27f8352cf8d5d",
	"title": "Alacritty",
	"appid": "Alacritty",
	"monitor": "eDP-2",
	"tags": [
		3
	],
	"is_xwayland": false,
	"is_swallowing": false,
	"is_swallowedby": false,
	"is_group": false,
	"is_visible": false,
	"is_focused": false,
	"is_fullscreen": false,
	"is_floating": false,
	"is_maximized": false,
	"is_global": false,
	"is_unglobal": false,
	"is_overlay": false,
	"is_fakefullscreen": false,
	"is_minimized": false,
	"is_urgent": false,
	"is_scratchpad": false,
	"is_namedscratchpad": false,
	"x": 1915,
	"y": 35,
	"width": 952,
	"height": 1045,
	"scroller_proportion": 0.5
},
```

Now it'll look for an icon named 'alacritty'. If that fails, it'll look for a .desktop file named alacritty.desktop, which should contain the icon name or path. If this fails as well, no icon will be displayed. I've added workarounds for some most common exceptions, but it's impossible to predict every single application misbehaviour. This is either programmers fault (improper class name), or bad packaging (.desktop file name different from the application class name).

If some app has no icon in the dock:

1. check the app class name (`mmsg get all-clients`);
2. find the app's .desktop file;
3. copy it to ~/.local/share/applications/` and rename to <class_name>.desktop.

If the .desktop file contains proper icon definition (`Icon=`), it should work now.

## Credits

This program uses some great libraries:

- [gotk4](https://github.com/diamondburned/gotk4) by [diamondburned](https://github.com/diamondburned) released under [GNU Affero General Public License v3.0](https://github.com/diamondburned/gotk4/blob/4/LICENSE.md)
- [go-singleinstance](github.com/allan-simon/go-singleinstance) Copyright (c) 2015 Allan Simon
