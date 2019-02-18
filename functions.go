package steward

import (
	"github.com/dyrkin/zcl-go"
	"github.com/dyrkin/zigbee-steward/coordinator"
)

type Functions struct {
	generic *GenericFunctions
	cluster *ClusterFunctions
}

func NewFunctions(coordinator *coordinator.Coordinator) *Functions {
	zcl := zcl.New()
	return &Functions{
		generic: &GenericFunctions{coordinator: coordinator},
		cluster: &ClusterFunctions{
			global: &GlobalClusterFunctions{
				coordinator: coordinator,
				zcl:         zcl,
			},
			local: &LocalClusterFunctions{
				coordinator: coordinator,
				zcl:         zcl,
			},
		},
	}
}

func (f *Functions) Generic() *GenericFunctions {
	return f.generic
}

func (f *Functions) Cluster() *ClusterFunctions {
	return f.cluster
}
