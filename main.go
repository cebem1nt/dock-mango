package main

/*
#include <signal.h>
*/
import "C"

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/allan-simon/go-singleinstance"
	log "github.com/sirupsen/logrus"

	"github.com/diamondburned/gotk4-layer-shell/pkg/gtklayershell"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
)

const version = "0.4.8"

type WindowState int

const (
	WindowShow WindowState = iota
	WindowHide
)

var (
	activeClient                       *Client
	appDirs                            []string
	clients                            []Client
	configDirectory                    string
	dataHome                           string
	detectorEnteredAt                  int64
	mangoSocket                        string // $MANGO_INSTANCE_SIGNATURE
	imgSizeScaled                      int
	lastWinAddr                        string
	mainBox                            *gtk.Box
	oldClients                         []Client
	outerOrientation, innerOrientation gtk.Orientation
	pinned                             []string
	pinnedFile                         string
	src                                glib.SourceHandle
	widgetAnchor, menuAnchor           gdk.Gravity
	win                                *gtk.Window
	windowStateChannel                 chan WindowState = make(chan WindowState, 1)
	classesToIgnore                    []string
	mouseInsideDock                    bool
	mouseInsideHotspot                 bool
	locked                             bool
)

// Flags
var alignment = flag.String("a", "center", "Alignment in full width/height: \"start\", \"center\" or \"end\"")
var autohide = flag.Bool("d", false, "auto-hiDe: show dock when hotspot hovered, close when left or a button clicked")
var cssFileName = flag.String("s", "style.css", "Styling: css file name")
var debug = flag.Bool("debug", false, "turn on debug messages")
var displayVersion = flag.Bool("v", false, "display Version information")
var exclusive = flag.Bool("x", false, "set eXclusive zone: move other windows aside; overrides the \"-l\" argument")
var full = flag.Bool("f", false, "take Full screen width/height")
var ignoreClasses = flag.String("g", "", "quote-delimited, space-separated class list to iGnore in the dock")
var hotspotDelay = flag.Int64("hd", 20, "Hotspot Delay [ms]; the smaller, the faster mouse pointer needs to enter hotspot for the dock to appear; set 0 to disable")
var hotspotLayer = flag.String("hl", "overlay", "Hotspot Layer \"overlay\", \"top\" or \"bottom\"")
var ico = flag.String("ico", "", "alternative name or path for the launcher ICOn")
var imgSize = flag.Int("i", 48, "Icon size")
var launcherCmd = flag.String("c", "nwg-drawer", "Command assigned to the launcher button")
var launcherPos = flag.String("lp", "end", "Launcher button position, 'start' or 'end'")
var layer = flag.String("l", "overlay", "Layer \"overlay\", \"top\" or \"bottom\"")
var marginBottom = flag.Int("mb", 0, "Margin Bottom")
var marginLeft = flag.Int("ml", 0, "Margin Left")
var marginRight = flag.Int("mr", 0, "Margin Right")
var marginTop = flag.Int("mt", 0, "Margin Top")
var noLauncher = flag.Bool("nolauncher", false, "don't show the launcher button")
var numWS = flag.Int64("w", 10, "number of Workspaces you use")
var position = flag.String("p", "bottom", "Position: \"bottom\", \"top\" \"left\" or \"right\"")
var resident = flag.Bool("r", false, "Leave the program resident, but w/o hotspot")
var allowMultipleInstances = flag.Bool("m", false, "allow Multiple instances of the dock (skip lock file check)")

var vertical bool
var alignmentBox *gtk.Box

func buildMainBox() {
	if mainBox != nil {
		mainBox.Destroy()
	}

	mainBox = gtk.NewBox(innerOrientation, 0)

	if *alignment == "start" {
		alignmentBox.PackStart(mainBox, false, true, 0)
	} else if *alignment == "end" {
		alignmentBox.PackEnd(mainBox, false, true, 0)
	} else {
		alignmentBox.PackStart(mainBox, true, false, 0)
	}

	var err error
	pinned, err = loadTextFile(pinnedFile)
	if err != nil {
		pinned = nil
	}

	var allItems []string
	for _, cntPin := range pinned {
		if !isIn(allItems, cntPin) {
			allItems = append(allItems, cntPin)
		}
	}

	for _, cntTask := range clients {
		if !isIn(allItems, cntTask.AppId) && !strings.Contains(*launcherCmd, cntTask.AppId) && cntTask.AppId != "" {
			allItems = append(allItems, cntTask.AppId)
		}
	}

	divider := 1
	if len(allItems) > 0 {
		divider = len(allItems)
	}

	// scale icons down when their number increases
	if *imgSize*6/(divider) < *imgSize {
		overflow := (len(allItems) - 6) / 3
		imgSizeScaled = *imgSize * 6 / (6 + overflow)
	} else {
		imgSizeScaled = *imgSize
	}

	if *launcherPos == "start" {
		button := launcherButton(position)
		if button != nil {
			mainBox.PackStart(button, false, false, 0)
		}
	}

	var alreadyAdded []string

	for _, pin := range pinned {
		if isIn(classesToIgnore, pin) {
			log.Debugf("Ignoring pin '%s'", pin)
			continue
		}

		if inTasks(pin) {
			instances := taskInstances(pin)
			c := instances[0]

			if isIn(classesToIgnore, c.AppId) || isIn(alreadyAdded, c.AppId) {
				continue
			}

			button := taskButton(c, instances, position)
			mainBox.PackStart(button, false, false, 0)

			if c.AppId == activeClient.AppId && !*autohide {
				button.SetObjectProperty("name", "active")
			} else {
				button.SetObjectProperty("name", "")
			}

			clientMenu(c.AppId, instances)
			alreadyAdded = append(alreadyAdded, c.AppId)
		} else {
			button := pinnedButton(pin, position)
			mainBox.PackStart(button, false, false, 0)
		}
	}

	for _, t := range clients {
		if isIn(classesToIgnore, t.AppId) {
			log.Debugf("Ignoring '%s'", t.AppId)
			continue
		}

		instances := taskInstances(t.AppId)

		if !isIn(alreadyAdded, t.AppId) {
			button := taskButton(t, instances, position)
			mainBox.PackStart(button, false, false, 0)

			if t.AppId == activeClient.AppId && !*autohide {
				button.SetObjectProperty("name", "active")
			} else {
				button.SetObjectProperty("name", "")
			}

			clientMenu(t.AppId, instances)
			alreadyAdded = append(alreadyAdded, t.AppId)
		}
	}

	if *launcherPos == "end" {
		button := launcherButton(position)

		if button != nil {
			mainBox.PackStart(button, false, false, 0)
		}
	}

	mainBox.ShowAll()
}

func setupHotSpot(dockWindow *gtk.Window) gtk.Window {
	w, h := dockWindow.Size()
	win := gtk.NewWindow(gtk.WindowToplevel)

	gtklayershell.InitForWindow(win)
	gtklayershell.SetNamespace(win, "hotspot")

	var box *gtk.Box
	if *position == "bottom" || *position == "top" {
		box = gtk.NewBox(gtk.OrientationVertical, 0)
	} else {
		box = gtk.NewBox(gtk.OrientationHorizontal, 0)
	}
	win.Add(box)

	detectorBox := gtk.NewEventBox()
	detectorBox.SetObjectProperty("name", "detector-box")

	if *position == "bottom" || *position == "right" {
		box.PackStart(detectorBox, false, false, 0)
	} else {
		box.PackEnd(detectorBox, false, false, 0)
	}

	// Toggle dock on click only
	detectorBox.Connect("button-press-event", func(_ *gtk.EventBox, e *gdk.Event) {
		btnEvent := e.AsButton()
		if btnEvent.Button() == 1 {
			if locked {
				return
			}

			locked = true

			if !dockWindow.IsVisible() {
				dockWindow.Show()
			} else {
				dockWindow.Hide()
			}

			go func() {
				glib.TimeoutAdd(300, func() {
					locked = false
				})
			}()
		}
	})

	if *position == "bottom" || *position == "top" {
		detectorBox.SetSizeRequest(w, h/3)

		if *position == "bottom" {
			gtklayershell.SetAnchor(win, gtklayershell.LayerShellEdgeBottom, true)
		} else {
			gtklayershell.SetAnchor(win, gtklayershell.LayerShellEdgeTop, true)
		}

		gtklayershell.SetAnchor(win, gtklayershell.LayerShellEdgeLeft, *full)
		gtklayershell.SetAnchor(win, gtklayershell.LayerShellEdgeRight, *full)
	}

	if *position == "left" || *position == "right" {
		detectorBox.SetSizeRequest(w/3, h)

		if *position == "left" {
			gtklayershell.SetAnchor(win, gtklayershell.LayerShellEdgeLeft, true)
		} else {
			gtklayershell.SetAnchor(win, gtklayershell.LayerShellEdgeRight, true)
		}

		gtklayershell.SetAnchor(win, gtklayershell.LayerShellEdgeTop, *full)
		gtklayershell.SetAnchor(win, gtklayershell.LayerShellEdgeBottom, *full)
	}

	if *hotspotLayer == "top" {
		gtklayershell.SetLayer(win, gtklayershell.LayerShellLayerTop)
	} else if *hotspotLayer == "bottom" {
		gtklayershell.SetLayer(win, gtklayershell.LayerShellLayerBottom)
	} else {
		gtklayershell.SetLayer(win, gtklayershell.LayerShellLayerOverlay)
	}

	gtklayershell.SetExclusiveZone(win, -1)

	return *win
}

func main() {
	sigRtmin := syscall.Signal(C.SIGRTMIN)
	sigToggle := sigRtmin + 1
	sigShow := sigRtmin + 2
	sigHide := sigRtmin + 3

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", flag.CommandLine.Name())
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nUsage of signals:\n")
		fmt.Fprintf(flag.CommandLine.Output(), " SIGRTMIN+1 (%s): toggle dock visibility (USR1 has been deprecated)\n", sigToggle)
		fmt.Fprintf(flag.CommandLine.Output(), " SIGRTMIN+2 (%s): show the dock\n", sigShow)
		fmt.Fprintf(flag.CommandLine.Output(), " SIGRTMIN+3 (%s): hide the dock\n", sigHide)
	}

	flag.Parse()
	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	if *autohide && *resident {
		log.Warn("autohiDe and Resident arguments are mutually exclusive, ignoring -d!")
		*autohide = false
	}

	if *displayVersion {
		fmt.Printf("nwg-dock-hyprland version %s\n", version)
		os.Exit(0)
	}

	if os.Getenv("MANGO_INSTANCE_SIGNATURE") != "" {
		mangoSocket = os.Getenv("MANGO_INSTANCE_SIGNATURE")
	} else {
		log.Fatal("MANGO_INSTANCE_SIGNATURE is not set. Are you running mango?")
	}

	if *autohide {
		log.Info("Starting in autohiDe mode")
	}

	if *resident {
		log.Info("Starting in resident mode")
	}

	if *ignoreClasses != "" {
		log.Infof("Ignoring classes: '%s'", *ignoreClasses)
		classesToIgnore = strings.Split(*ignoreClasses, " ")
	}

	// Gentle SIGTERM handler thanks to reiki4040 https://gist.github.com/reiki4040/be3705f307d3cd136e85
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGUSR1, sigToggle, sigShow, sigHide)

	go func() {
		for {
			s := <-signalChan
			switch s {
			case syscall.SIGTERM:
				log.Info("SIGTERM received, bye bye!")
				gtk.MainQuit()
			case sigToggle:
				if *resident || *autohide {
					if !win.IsVisible() {
						log.Debug("sigToggle received, showing the window")
						windowStateChannel <- WindowShow
					} else {
						log.Debug("sigToggle received, hiding the window")
						windowStateChannel <- WindowHide
					}
				} else {
					log.Debug("sigToggle received, but I'm not resident, ignoring")
				}
			case sigShow:
				if *resident || *autohide {
					if !win.IsVisible() {
						log.Debug("sigShow received, showing the window")
						windowStateChannel <- WindowShow
					} else {
						log.Debug("sigShow received, but window already visible, ignoring")
					}
				} else {
					log.Debug("sigToggle received, but I'm not resident, ignoring")
				}
			case sigHide:
				if *resident || *autohide {
					if !win.IsVisible() {
						log.Debug("sigHide received, but window already hidden, ignoring")
					} else {
						log.Debug("sigHide received, hiding the window")
						windowStateChannel <- WindowHide
					}
				} else {
					log.Debug("sigHide received, but I'm not resident, ignoring")
				}
			default:
				log.Warn("Unknown signal")
			}
		}
	}()

	var err error

	if !*allowMultipleInstances {
		log.Debug("Allowing only one instance of nwg-dock-hyprland")
		// If running instance found, send sigToggle to it.
		// If it's running with `-r` or `-d` flag, it'll show/hide the window.
		// Otherwise, it'll ignore the signal.

		// Use md5-hashed $USER name to create unique lock files for multiple users
		lockFilePath := fmt.Sprintf("%s/nwg-dock-%s.lock", tempDir(), md5Hash(os.Getenv("USER")))
		lockFile, e := singleinstance.CreateLockFile(lockFilePath)

		if e != nil {
			content, err := readTextFile(lockFilePath)
			if err == nil {
				pid, err := strconv.Atoi(content)
				if err == nil {
					if *autohide || *resident {
						log.Info("Running instance found, terminating...")
					} else {
						_ = syscall.Kill(pid, sigToggle)
						log.Info("Sending sigToggle to running instance and bye, bye!")
					}
				}
			} else {
				log.Warnf("Error reading lock file: %s at %s", err, lockFilePath)
			}
			os.Exit(0)
		}

		defer lockFile.Close()
	}

	if !*noLauncher && *launcherCmd == "" {
		if isCommand("nwg-drawer") {
			*launcherCmd = "nwg-drawer"
		} else if isCommand("nwggrid") {
			*launcherCmd = "nwggrid -p"
		}

		if *launcherCmd != "" {
			log.Infof("Using auto-detected launcher command: '%s'", *launcherCmd)
		} else {
			log.Info("Neither 'nwg-drawer' nor 'nwggrid' command found, and no other launcher specified; hiding the launcher button.")
		}
	}

	dataHome, err = getDataHome()
	if err != nil {
		log.Fatal("Error getting data directory:", err)
	}
	configDirectory = configDir()
	// if it doesn't exist:
	createDir(configDirectory)

	if !pathExists(fmt.Sprintf("%s/style.css", configDirectory)) {
		err := copyFile(filepath.Join(dataHome, "nwg-dock-hyprland/style.css"), fmt.Sprintf("%s/style.css", configDirectory))
		if err != nil {
			log.Warnf("Error copying file: %s", err)
		}
	}

	cacheDirectory := cacheDir()
	if cacheDirectory == "" {
		log.Panic("Couldn't determine cache directory location")
	}
	pinnedFile = filepath.Join(cacheDirectory, "nwg-dock-pinned")
	cssFile := filepath.Join(configDirectory, *cssFileName)

	appDirs = getAppDirs()

	gtk.Init()

	cssProvider := gtk.NewCSSProvider()

	err = cssProvider.LoadFromPath(cssFile)
	if err != nil {
		log.Warnf("%s file not found, using GTK styling\n", cssFile)
	} else {
		log.Printf("Using style: %s\n", cssFile)
		screen := gdk.ScreenGetDefault()
		gtk.StyleContextAddProviderForScreen(screen, cssProvider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
	}

	win = gtk.NewWindow(gtk.WindowToplevel)
	if err != nil {
		log.Fatal("Unable to create window:", err)
	}

	gtklayershell.InitForWindow(win)
	gtklayershell.SetNamespace(win, "nwg-dock")

	if *exclusive {
		gtklayershell.AutoExclusiveZoneEnable(win)
		*layer = "top"
	}

	if *position == "bottom" || *position == "top" {
		if *position == "bottom" {
			gtklayershell.SetAnchor(win, gtklayershell.LayerShellEdgeBottom, true)

			widgetAnchor = gdk.GravityNorth
			menuAnchor = gdk.GravitySouth
		} else {
			gtklayershell.SetAnchor(win, gtklayershell.LayerShellEdgeTop, true)

			widgetAnchor = gdk.GravitySouth
			menuAnchor = gdk.GravityNorth
		}

		outerOrientation = gtk.OrientationVertical
		innerOrientation = gtk.OrientationHorizontal

		gtklayershell.SetAnchor(win, gtklayershell.LayerShellEdgeLeft, *full)
		gtklayershell.SetAnchor(win, gtklayershell.LayerShellEdgeRight, *full)
	}

	if *position == "left" || *position == "right" {
		if *position == "left" {
			gtklayershell.SetAnchor(win, gtklayershell.LayerShellEdgeLeft, true)
		} else {
			gtklayershell.SetAnchor(win, gtklayershell.LayerShellEdgeRight, true)
		}

		gtklayershell.SetAnchor(win, gtklayershell.LayerShellEdgeTop, *full)
		gtklayershell.SetAnchor(win, gtklayershell.LayerShellEdgeBottom, *full)

		outerOrientation = gtk.OrientationHorizontal
		innerOrientation = gtk.OrientationVertical

		widgetAnchor = gdk.GravityEast
		menuAnchor = gdk.GravityWest
	}

	if *layer == "top" {
		gtklayershell.SetLayer(win, gtklayershell.LayerShellLayerTop)
	} else if *layer == "bottom" {
		gtklayershell.SetLayer(win, gtklayershell.LayerShellLayerBottom)
	} else {
		gtklayershell.SetLayer(win, gtklayershell.LayerShellLayerOverlay)
	}

	gtklayershell.SetMargin(win, gtklayershell.LayerShellEdgeTop, *marginTop)
	gtklayershell.SetMargin(win, gtklayershell.LayerShellEdgeLeft, *marginLeft)
	gtklayershell.SetMargin(win, gtklayershell.LayerShellEdgeRight, *marginRight)
	gtklayershell.SetMargin(win, gtklayershell.LayerShellEdgeBottom, *marginBottom)

	win.Connect("destroy", func() {
		gtk.MainQuit()
	})

	// Close the window on leave, but not immediately, to avoid accidental closes
	if *autohide {
		win.Connect("leave-notify-event", func() {
			src = glib.TimeoutAdd(1000, func() bool {
				mouseInsideDock = false
				win.Hide()
				src = 0
				return false
			})
		})
	}

	win.Connect("enter-notify-event", func() {
		mouseInsideDock = true
		cancelClose()
	})

	outerBox := gtk.NewBox(outerOrientation, 0)
	outerBox.SetObjectProperty("name", "box")
	win.Add(outerBox)

	alignmentBox = gtk.NewBox(innerOrientation, 0)
	outerBox.PackStart(alignmentBox, true, true, 0)

	mainBox = gtk.NewBox(innerOrientation, 0)
	var hotspot gtk.Window
	// We'll pack mainBox later, in buildMainBox

	err = updateClients()
	if err != nil {
		log.Fatalf("Couldn't list clients: %s", err)
	}

	if *autohide {
		glib.TimeoutAdd(500, win.Hide)

		mRefProvider := gtk.NewCSSProvider()
		css := "window { all: unset; }"
		hotspotCssFile := filepath.Join(configDirectory, "hotspot.css")
		if !pathExists(hotspotCssFile) {
			_ = mRefProvider.LoadFromData(css)
			log.Infof("Optional '%s' file not found, using internal definition", hotspotCssFile)
		} else {
			err := mRefProvider.LoadFromPath(hotspotCssFile)
			if err == nil {
				log.Infof("Hotspot css loaded from %s", hotspotCssFile)
			} else {
				log.Warnf("Error loading hotspot css from %s", hotspotCssFile)
			}
		}

		if err != nil {
			log.Warn(err)
		}

		hotspot = setupHotSpot(win)

		ctx := hotspot.StyleContext()
		ctx.AddProvider(mRefProvider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)

		hotspot.ShowAll()
	}

	buildMainBox()
	win.ShowAll()

	go func() {
		for {
			windowState := <-windowStateChannel

			glib.TimeoutAdd(0, func() bool {
				if windowState == WindowShow && win != nil && !win.IsVisible() {
					win.ShowAll()
				}
				if windowState == WindowHide && win != nil && win.IsVisible() {
					win.Hide()
				}

				return false
			})
		}
	}()

	addr := &net.UnixAddr{
		Name: mangoSocket,
		Net:  "unix",
	}

	oldClients = clients

	refreshMainBox := func() {
		if len(clients) != len(oldClients) {
			glib.TimeoutAdd(0, func() bool {
				buildMainBox()
				oldClients = clients
				return false
			})
		}
	}

	go func() {
		conn, err := net.DialUnix("unix", nil, addr)

		if err != nil {
			log.Fatalf("Error connecting to the socket: %s", err)
		}

		defer conn.Close()
		conn.Write([]byte("watch all-clients\n"))

		r := bufio.NewReader(conn)

		for {
			if _, err = r.ReadString('\n'); err != nil {
				log.Fatalf("Error reading: %s", err)
			}

			err = updateClients()

			if err != nil {
				log.Fatalf("Couldn't update clients: %s", err)
			}

			glib.IdleAdd(func() bool {
				refreshMainBox()
				return false
			})
		}
	}()

	gtk.Main()
}
