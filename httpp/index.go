package httpp

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"test/cluster"
	"test/localInstance"
	"time"

	"golang.org/x/net/http2"
)

func ServeListenerIn(ln net.Listener, servName string, ins *localInstance.LocalInstance) {
	defer ln.Close()

	handler := &myHandler{outbound: false}
	handler.cycle = &inboundCycle{ins: ins, servName: servName}

	//http1
	var err error
	server := &http.Server{
		Handler: http.HandlerFunc(handler.handle),
	}
	//add http2 support to http1 server，这样能服务http1和2，http2.Server只支持http2
	err = http2.ConfigureServer(server, &http2.Server{})
	if err != nil {
		log.Fatalf("Failed to configure HTTP/2 server: %v", err)
	}

	err = server.Serve(ln)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func ServeListenerOut(ln net.Listener) {

	defer ln.Close()

	handler := &myHandler{outbound: true}
	handler.cycle = &outboundCycle{}

	//http1
	var err error
	server := &http.Server{
		Handler: http.HandlerFunc(handler.handle),
	}
	//add http2 support to http1 server，这样能服务http1和2，http2.Server只支持http2
	err = http2.ConfigureServer(server, &http2.Server{})
	if err != nil {
		log.Fatalf("Failed to configure HTTP/2 server: %v", err)
	}

	err = server.Serve(ln)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

var tr = &http.Transport{
	MaxIdleConns:        0,                // 最大空闲连接数
	MaxIdleConnsPerHost: 50,               // 每个主机的最大空闲连接数
	IdleConnTimeout:     60 * time.Second, // 空闲连接的超时时间
}

var client = &http.Client{
	Transport: tr,
}

type LifeCycle interface {
	toReq(*http.Request) (*http.Request, error)
}

type outboundCycle struct {
	LifeCycle
}

func (c *outboundCycle) toReq(r *http.Request) (*http.Request, error) {
	host := r.Header.Get("Host")
	cls := cluster.GetOrCreate(host) //集群
	ins := cls.Choose()              //实例  todo by 勇道

	if ins == nil {
		return nil, errors.New("no instance")
	}

	return http.NewRequest(r.Method, getFullURL(r, ins.IP+":"+strconv.Itoa(ins.Port)), r.Body)
}

type inboundCycle struct {
	LifeCycle
	servName string
	ins      *localInstance.LocalInstance
}

func (c *inboundCycle) toReq(r *http.Request) (*http.Request, error) {
	ins := c.ins
	return http.NewRequest(r.Method, getFullURL(r, ins.Ip+":"+strconv.Itoa(ins.Port)), r.Body)
}

type myHandler struct {
	cycle    LifeCycle
	outbound bool
}

func (mh *myHandler) handle(w http.ResponseWriter, r *http.Request) {
	fmt.Println("start servingggg  serv")
	defer r.Body.Close()

	req, err := mh.cycle.toReq(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	//开始转发流量，downstream upstream侧任何一方错误，那么两侧连接或stream不能用，无需再读取完Body，proxy两侧处理好超时即可
	var success bool
	defer func() {
		if !success {
			fmt.Println("转发异常")
		}
	}()
	resp, err := forwardReq(req, client)
	defer func() {
		if resp.Body != nil {
			resp.Body.Close()
		}
	}()
	if err != nil {
		return
	}
	err = forwardResp(resp, w)
	if err != nil {
		return
	}
	success = true
}

func forwardReq(r *http.Request, client *http.Client) (*http.Response, error) {
	return client.Do(r)
}
func forwardResp(resp *http.Response, w http.ResponseWriter) error {
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, err := io.Copy(w, resp.Body)
	return err
}

func getFullURL(req *http.Request, host string) string {
	u := &url.URL{
		Scheme:   "http",
		Host:     host,
		Path:     req.URL.Path,
		RawQuery: req.URL.RawQuery,
		Fragment: req.URL.Fragment,
	}

	return u.String()
}
