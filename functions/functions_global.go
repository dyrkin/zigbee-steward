package functions

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/dyrkin/bin"
	"github.com/dyrkin/zcl-go"
	"github.com/dyrkin/zcl-go/cluster"
	"github.com/dyrkin/zcl-go/frame"
	"github.com/dyrkin/zigbee-steward/coordinator"
	"github.com/dyrkin/znp-go"
)

type GlobalClusterFunctions struct {
	coordinator *coordinator.Coordinator
	zcl         *zcl.Zcl
}

func (f *GlobalClusterFunctions) ReadAttributes(nwkAddress string, clusterId cluster.ClusterId, attributeIds []uint16) (*cluster.ReadAttributesResponse, error) {
	response, err := f.globalCommand(nwkAddress, clusterId, 0x00, &cluster.ReadAttributesCommand{attributeIds})

	if err == nil {
		return response.(*cluster.ReadAttributesResponse), nil
	}
	return nil, err
}

func (f *GlobalClusterFunctions) WriteAttributes(nwkAddress string, clusterId cluster.ClusterId, writeAttributeRecords []*cluster.WriteAttributeRecord) (*cluster.WriteAttributesResponse, error) {
	response, err := f.globalCommand(nwkAddress, clusterId, 0x02, &cluster.WriteAttributesCommand{writeAttributeRecords})

	if err == nil {
		return response.(*cluster.WriteAttributesResponse), nil
	}
	return nil, err
}

func (f *GlobalClusterFunctions) globalCommand(nwkAddress string, clusterId cluster.ClusterId, commandId uint8, command interface{}) (interface{}, error) {
	options := &znp.AfDataRequestOptions{}
	frm, err := frame.New().
		DisableDefaultResponse(true).
		FrameType(frame.FrameTypeGlobal).
		Direction(frame.DirectionClientServer).
		CommandId(commandId).
		Command(command).
		Build()

	if err != nil {
		return nil, err
	}

	response, err := f.coordinator.DataRequest(nwkAddress, 255, 1, uint16(clusterId), options, 15, bin.Encode(frm))
	if err == nil {
		zclIncomingMessage, err := f.zcl.ToZclIncomingMessage(response)
		if err == nil {
			return zclIncomingMessage.Data.Command, nil
		} else {
			log.Errorf("Unsupported data response message:\n%s\n", func() string { return spew.Sdump(response) })
		}

	}
	return nil, err
}
