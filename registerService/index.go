package registerService

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func export(src string, proxyPort int, pushRegistry bool) {

}

type ReportServ struct {
	ip           string
	port         int
	proxyPort    int
	pushRegistry bool
}

func handler(w http.ResponseWriter, r *http.Request) {
	//parse  asemble
	data, err := io.ReadAll(r.Body)
	var result ReportServ
	err = json.Unmarshal(data, &result)

	//call exp
}

func start() {
	http.HandleFunc("/export", handler)

	fmt.Println("Starting server at port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
