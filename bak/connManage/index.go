package connManage

import "net"

type Req struct {
	Finished chan bool
	Resp     interface{}
	Err      error
}

var activeReqs = make(map[net.Conn]map[string]*Req)

func AddReqFuture(conn net.Conn, id string) *Req {
	r := &Req{
		Finished: make(chan bool),
	}
	activeReqs[conn] = make(map[string]*Req)
	activeReqs[conn][id] = r
	return r
}

func RemoveActiveReq(conn net.Conn, id string, resp interface{}) {
	r := activeReqs[conn][id]
	r.Finished <- true
	r.Resp = resp
	delete(activeReqs[conn], id)
}
