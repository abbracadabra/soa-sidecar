package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"test/cmd/config"
	"test/nameService"
)

type reportServ struct {
	ip          string
	port        int
	transparent bool
	proxyIp     string
	proxyPort   int
	secure      bool
	protocol    string
	//注册中心参数
	export   bool
	servName string
	tags     map[string]string
}

type heartbeatMsg struct {
	proxyIp   string
	proxyPort int
	servName  string
	tags      map[string]string
}

func inboundExportHandler(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	var msg reportServ
	err = json.Unmarshal(data, &data)
	if err != nil {
		fmt.Println(err)
	}

	cfg := config.GetConfig()
	serveProtocolIn(msg.servName, msg.ip, msg.port, msg.proxyIp, msg.proxyPort, cfg.InboundTransparent, msg.secure, msg.protocol)

	//if msg.export {
	//	nameService.Heartbeat(msg.servName, msg.proxyIp, msg.proxyPort, msg.tags)
	//}
}

func heartbeatHandler(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	var msg heartbeatMsg
	err = json.Unmarshal(data, &msg)
	if err != nil {
		fmt.Println(err)
	}

	nameService.RegisterInstance(msg.servName, msg.proxyIp, msg.proxyPort, msg.tags)
}

func startConsoleServer(ip string, port int) {
	http.HandleFunc("/startProxy", inboundExportHandler)
	http.HandleFunc("/exportService", heartbeatHandler)

	fmt.Printf("Starting server at %s:%d\n", ip, port)
	if err := http.ListenAndServe(ip+":"+strconv.Itoa(port), nil); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
