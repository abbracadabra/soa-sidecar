package localInstance

var InstanceMap = make(map[string]*LocalInstance)

// func FindByPort(ip string, port int) *LocalInstance {
// 	return InstanceMap[ip+":"+strconv.Itoa(port)]
// }

// func AddInstance(ip string, port int) error {
// 	InstanceMap[ip+":"+strconv.Itoa(port)] = &LocalInstance{
// 		Pool: nil,
// 		// servName: servName,
// 		ip:   ip,
// 		port: port,
// 	}
// 	return nil
// }

type LocalInstance struct {
	ServName string
	Ip       string
	Port     int
	Pool     interface{}
}
