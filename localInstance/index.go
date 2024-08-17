package localInstance

import "test/cluster"

var InstanceMap = make(map[string]*cluster.Instance)

func FindByPort(port int) *cluster.Instance {
	return InstanceMap[name]
}
