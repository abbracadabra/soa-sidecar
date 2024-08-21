package swimlane

import (
	"strings"
	"test/cluster"
)

// color:染色信息
func FilterLane(ins []*cluster.Instance, color string) []*cluster.Instance {
	colors := strings.Split(color, ",")

	var main = false
	for _, c := range colors {
		if c == "main" {
			main = true
		}
	}
	if !main {
		colors = append(colors, "main")
	}

	var res []*cluster.Instance
	for _, c := range colors {
		match := make([]*cluster.Instance, 0)
		for i := 0; i < len(ins); i++ {
			if ins[i].Meta["color"] == c {
				match = append(match, ins[i])
			}
		}
		if len(match) > 0 {
			res = match
			break
		}
	}
	return res
}
