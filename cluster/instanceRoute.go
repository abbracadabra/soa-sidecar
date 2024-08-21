package cluster

import (
	"math/rand"
	"sync"
	"test/utils/swimlane"
)

type RouteInfo struct {
	Color string
	Key   string
}

// 同机房  同泳道  路由hashkey
// 因路由hashkey需要每个cls有额外数据结构，需要闭包环境
func instanceRouteByLaneCreator(cls *Cluster) InstanceRouter {
	stickyMap := sync.Map{}
	return func(routeInfo interface{}) *Instance {
		rf := routeInfo.(*RouteInfo)
		bind, sticky := stickyMap.Load(rf.Key)
		if sticky {
			ins := bind.(*Instance)
			if ins.Status == REMOVED {
				stickyMap.Delete(rf.Key)
			} else {
				return ins
			}
		}
		// lane
		filtered := swimlane.FilterLane(cls.GetInstances(), rf.Color)
		// idc todo

		n := len(filtered)
		if n == 0 {
			return nil
		}
		i := rand.Intn(n)
		chosen := filtered[i]
		if sticky {
			stickyMap.Store(rf.Key, chosen)
		}
		return chosen
	}
}
