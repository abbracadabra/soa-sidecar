package registerService

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"test/cluster"
	"test/nameService"
	"test/serverr"
)

func export(src string, proxyPort int, pushRegistry bool) {

	// cluster.FindByName()

}

type ReportServ struct {
	ip           string
	port         int
	proxyPort    int
	pushRegistry bool
	servName     string
	tags         map[string]string
}

func handler(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	var result ReportServ
	err = json.Unmarshal(data, &result)
	if err != nil {
		fmt.Println(err)
	}
	cluster.FindByName(result.servName, false).Add(result.ip, result.port, result.tags)

	if result.pushRegistry {
		nameService.Register(result.servName, result.ip, result.port)
	}

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(result.proxyPort))
	if err != nil {
		log.Fatalf("Error creating listener: %s", err)
	}
	defer listener.Close()

	log.Println("Server is listening on :8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %s", err)
			continue
		}
		serverr.Route()
	}

	//看看nacos的register怎么搞的

	//call exp
}

func start() {
	http.HandleFunc("/export", handler)

	fmt.Println("Starting server at port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
