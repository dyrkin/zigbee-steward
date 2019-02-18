package steward

import (
	"github.com/dyrkin/zcl-go"
	"github.com/dyrkin/zigbee-steward/coordinator"
)

type LocalClusterFunctions struct {
	coordinator *coordinator.Coordinator
	zcl         *zcl.Zcl
}
