package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
)

type Monitor struct {
	Name string `json:"name"`
}

type Client struct {
	Id    int    `json:"id"`
	Pid   int    `json:"pid"`
	Title string `json:"title"`
	AppId string `json:"appid"`
	Tags  []int  `json:"tags"`
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

func updateClients() error {
	reply, err := mmsg("get all-clients")

	if err != nil {
		return err
	}

	var resp struct {
		Clients []Client `json:"clients"`
	}

	if err := json.Unmarshal(reply, &resp); err != nil {
		return err
	}

	clients = resp.Clients

	activeClient, _ = getActiveWindow()
	return nil
}

func getActiveWindow() (*Client, error) {
	var activeWindow Client
	reply, err := mmsg("get focusing-client")

	err = json.Unmarshal([]byte(reply), &activeWindow)
	if err == nil {
		return &activeWindow, nil
	}

	return nil, err
}

func focusWindow(window Client) {
	cmd := fmt.Sprintf("dispatch focusid, client, %d", window.Id)
	reply, _ := mmsg(cmd)

	log.Debugf("%s -> %s", cmd, reply)
}

func closeWindow(window Client) {
	cmd := fmt.Sprintf("dispatch killclient, client, %d", window.Id)
	reply, _ := mmsg(cmd)

	log.Debugf("%s -> %s", cmd, reply)
}

func floatWindow(window Client) {
	cmd := fmt.Sprintf("dispatch togglefloating, client, %d", window.Id)
	reply, _ := mmsg(cmd)

	log.Debugf("%s -> %s", cmd, reply)
}

func fullscreenWindow(window Client) {
	cmd := fmt.Sprintf("dispatch togglefullscreen, client, %d", window.Id)
	reply, _ := mmsg(cmd)

	log.Debugf("%s -> %s", cmd, reply)
}

func tagWindow(window Client, workspace int) {
	cmd := fmt.Sprintf("dispatch tag, client, %d, %d, 0", window.Id, workspace)
	reply, _ := mmsg(cmd)

	log.Debugf("%s -> %s", cmd, reply)
}
