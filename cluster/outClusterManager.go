package cluster

import (
	"golang.org/x/sync/singleflight"
)

var outboundClusters singleflight.Group

func GetOrCreate(name string) *Cluster {
	result, _, _ := outboundClusters.Do(name, func() (interface{}, error) {
		return instantCreate(name)
	})
	return result.(*Cluster)
}
