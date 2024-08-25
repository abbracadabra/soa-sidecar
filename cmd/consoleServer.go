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

type proxyReq struct {
	ip        string
	port      int
	proxyIp   string
	proxyPort int
	secure    bool
	protocol  string
	//注册中心参数
	export   bool
	servName string
	tags     map[string]string
}

type heartbeatMsg struct {
	ip       string `心跳上报只需提供自身ip端口`
	port     int
	servName string
	tags     map[string]string
}

var (
	exportedService = make(map[string]*proxyReq)
)

func startInboundProxyHandler(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	var msg proxyReq
	err = json.Unmarshal(data, &msg)
	if err != nil {
		fmt.Println(err)
	}

	cfg := config.GetConfig()
	err = serveProtocolIn(msg.servName, msg.ip, msg.port, msg.proxyIp, msg.proxyPort, cfg.InboundTransparent, msg.secure, msg.protocol)
	if err != nil {
		respond(w, 500, 1, err.Error())
		return
	}
	// 记录下代理ip和端口
	exportedService[fmt.Sprintf("%s:%d", msg.ip, msg.port)] = &msg
	respond(w, 200, 0, "ok")
}

func exportServiceHandler(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	var msg heartbeatMsg
	err = json.Unmarshal(data, &msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//心跳上报只需提供自身ip端口
	proxyReq := exportedService[fmt.Sprintf("%s:%d", msg.ip, msg.port)]
	err = nameService.RegisterInstance(msg.servName, proxyReq.proxyIp, proxyReq.proxyPort, msg.tags)
	if err != nil {
		respond(w, 500, 1, err.Error())
		return
	}
	respond(w, 200, 0, "ok")
}

func respond(w http.ResponseWriter, status int, code int, msg string) error {
	body := map[string]any{"code": code, "msg": msg}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(body)
	if err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		return err
	}
	return nil
}

func startConsoleServer(ip string, port int) {
	http.HandleFunc("/startInboundProxy", startInboundProxyHandler)
	http.HandleFunc("/exportService", exportServiceHandler)

	fmt.Printf("Starting server at %s:%d\n", ip, port)
	if err := http.ListenAndServe(ip+":"+strconv.Itoa(port), nil); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
