package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
)

type tag struct {
	Index       int    `json:"index"`
	IsActive    bool   `json:"is_active"`
	IsUrgent    bool   `json:"is_urgent"`
	Layout      string `json:"layout"`
	ClientCount int    `json:"client_count"`
}

type monitor struct {
	Name            string `json:"name"`
	Active          bool   `json:"active"`
	IsHDR           bool   `json:"is_hdr"`
	IsVRR           bool   `json:"is_vrr"`
	X               int    `json:"x"`
	Y               int    `json:"y"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	LayoutIndex     int    `json:"layout_index"`
	LayoutSymbol    string `json:"layout_symbol"`
	LastOpenSurface string `json:"last_open_surface"`
	TagNum          int    `json:"tag_num"`
	HideClients     int    `json:"hide_clients"`
}

type client struct {
	Id                 int     `json:"id"`
	Pid                int     `json:"pid"`
	ForeignToplevelId  string  `json:"foreign_toplevel_id"`
	Title              string  `json:"title"`
	AppId              string  `json:"appid"`
	Monitor            string  `json:"monitor"`
	Tags               []int   `json:"tags"`
	IsXWayland         bool    `json:"is_xwayland"`
	IsSwallowing       bool    `json:"is_swallowing"`
	IsSwallowedBy      bool    `json:"is_swallowedby"`
	IsGroup            bool    `json:"is_group"`
	IsVisible          bool    `json:"is_visible"`
	IsFocused          bool    `json:"is_focused"`
	IsFullscreen       bool    `json:"is_fullscreen"`
	IsFloating         bool    `json:"is_floating"`
	IsMaximized        bool    `json:"is_maximized"`
	IsGlobal           bool    `json:"is_global"`
	IsUnglobal         bool    `json:"is_unglobal"`
	IsOverlay          bool    `json:"is_overlay"`
	IsFakeFullscreen   bool    `json:"is_fakefullscreen"`
	IsMinimized        bool    `json:"is_minimized"`
	IsUrgent           bool    `json:"is_urgent"`
	IsScratchpad       bool    `json:"is_scratchpad"`
	IsNamedScratchpad  bool    `json:"is_namedscratchpad"`
	X                  int     `json:"x"`
	Y                  int     `json:"y"`
	Width              int     `json:"width"`
	Height             int     `json:"height"`
	ScrollerProportion float64 `json:"scroller_proportion"`
}

func mmsg(cmd string) ([]byte, error) {
	conn, err := net.Dial("unix", mangoSocket)
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	if _, err := conn.Write([]byte(cmd + "\n")); err != nil {
		return nil, err
	}

	var out bytes.Buffer
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		out.Write(scanner.Bytes())
		out.WriteByte('\n')
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func listMonitors() error {
	reply, err := mmsg("get all-monitors")
	if err != nil {
		return err
	}

	var resp struct {
		Monitors []monitor `json:"monitors"`
	}

	if err := json.Unmarshal(reply, &resp); err != nil {
		return err
	}

	monitors = resp.Monitors
	return nil
}

func listClients() error {
	reply, err := mmsg("get all-clients")

	if err != nil {
		return err
	}

	var resp struct {
		Clients []client `json:"monitors"`
	}

	if err := json.Unmarshal(reply, &resp); err != nil {
		return err
	}

	clients = resp.Clients

	activeClient, _ = getActiveWindow()
	return nil
}

func getActiveWindow() (*client, error) {
	var activeWindow client
	reply, err := mmsg("get focusing-client")

	err = json.Unmarshal([]byte(reply), &activeWindow)
	if err == nil {
		return &activeWindow, nil
	}

	return nil, err
}

func focusWindow(window client) {
	cmd := fmt.Sprintf("dispatch focusid, %s", window.Id)
	reply, _ := mmsg(cmd)

	log.Debugf("%s -> %s", cmd, reply)
}

func closeWindow(window client) {
	// TODO, there is no closeid
}

func floatWindow(window client) {
	// TODO, no floatid
}

func fullscreenWindow(window client) {
	// TODO, no fullscreen by id
}

func moveWindowToWorkspace(window client, workspace int) {
	// TODO cant tag by id
}
