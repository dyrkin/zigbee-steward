package functions

import (
	"github.com/dyrkin/zigbee-steward/coordinator"
	"github.com/dyrkin/znp-go"
)

type GenericFunctions struct {
	coordinator *coordinator.Coordinator
}

func (f *GenericFunctions) Bind(sourceAddress string, sourceIeeeAddress string, sourceEndpoint uint8, clusterId uint16, destinationIeeeAddress string, destinationEndpoint uint8) (*znp.ZdoBindRsp, error) {
	return f.coordinator.Bind(sourceAddress, sourceIeeeAddress, sourceEndpoint, clusterId, znp.AddrModeAddr64Bit, destinationIeeeAddress, destinationEndpoint)
}

func (f *GenericFunctions) Unbind(sourceAddress string, sourceIeeeAddress string, sourceEndpoint uint8, clusterId uint16, destinationIeeeAddress string, destinationEndpoint uint8) (*znp.ZdoUnbindRsp, error) {
	return f.coordinator.Unbind(sourceAddress, sourceIeeeAddress, sourceEndpoint, clusterId, znp.AddrModeAddr64Bit, destinationIeeeAddress, destinationEndpoint)
}
