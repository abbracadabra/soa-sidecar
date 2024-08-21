package cluster

import (
	"math/rand"
	"sync"
	"test/utils/swimlane"
)

type RouteInfo struct {
	color string
	key   string
}

// 同机房  同泳道  路由hashkey
// 因路由hashkey需要每个cls有额外数据结构，需要闭包环境
func InstanceRouteByLaneCreator(cls *Cluster) InstanceRouter {
	stickyMap := sync.Map{}
	return func(routeInfo interface{}) *Instance {
		rf := routeInfo.(*RouteInfo)
		bind, ok := stickyMap.Load(rf.key)
		if ok {
			ins := bind.(*Instance)
			if ins.Status == REMOVED {
				stickyMap.Delete(rf.key)
			} else {
				return ins
			}
		}
		// lane
		filtered := swimlane.FilterLane(cls.GetInstances(), rf.color)
		// idc

		n := len(filtered)
		if n == 0 {
			return nil
		}
		i := rand.Intn(n)
		return filtered[i]
	}
}
