package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"test/nameService"
)

type ReportServ struct {
	ip           string
	port         int
	transparent  bool
	proxyPort    int
	secure       bool
	protocol     string
	pushRegistry bool
	//注册中心参数
	servName string
	tags     map[string]string
}

/*
if trans

	addServRule

else

	open port

add local instance mapping, init instance conn pool
if export2Reg

	invoke nacos
*/
func inboundExportHandler(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	var result ReportServ
	err = json.Unmarshal(data, &result)
	if err != nil {
		fmt.Println(err)
	}

	serveProtocolIn(result.servName, result.transparent, result.ip, result.port, result.secure, result.protocol)

	//TODO
	if result.pushRegistry {
		nameService.Register(result.servName, result.ip, result.port)
	}
}

func startConsoleServer(ip string, port int) {
	http.HandleFunc("/export", inboundExportHandler)

	fmt.Println("Starting server at port 8080")
	if err := http.ListenAndServe(ip+":"+strconv.Itoa(port), nil); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
