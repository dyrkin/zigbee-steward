package functions

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/dyrkin/bin"
	"github.com/dyrkin/zcl-go"
	"github.com/dyrkin/zcl-go/cluster"
	"github.com/dyrkin/zcl-go/frame"
	"github.com/dyrkin/zigbee-steward/coordinator"
	"github.com/dyrkin/znp-go"
)

type LocalClusterFunctions struct {
	onOff        *OnOff
	levelControl *LevelControl
}

type LocalCluster struct {
	clusterId   cluster.ClusterId
	coordinator *coordinator.Coordinator
	zcl         *zcl.Zcl
}

func NewLocalClusterFunctions(coordinator *coordinator.Coordinator, zcl *zcl.Zcl) *LocalClusterFunctions {
	return &LocalClusterFunctions{
		onOff: &OnOff{
			LocalCluster: &LocalCluster{
				clusterId:   cluster.OnOff,
				coordinator: coordinator,
				zcl:         zcl,
			},
		},
		levelControl: &LevelControl{
			LocalCluster: &LocalCluster{
				clusterId:   cluster.LevelControl,
				coordinator: coordinator,
				zcl:         zcl,
			},
		},
	}
}

func (f *LocalClusterFunctions) OnOff() *OnOff {
	return f.onOff
}

func (f *LocalClusterFunctions) LevelControl() *LevelControl {
	return f.levelControl
}

func (f *LocalCluster) localCommand(nwkAddress string, endpoint uint8, commandId uint8, command interface{}) error {
	options := &znp.AfDataRequestOptions{}
	frm, err := frame.New().
		DisableDefaultResponse(false).
		FrameType(frame.FrameTypeLocal).
		Direction(frame.DirectionClientServer).
		CommandId(commandId).
		Command(command).
		Build()

	if err != nil {
		return err
	}

	response, err := f.coordinator.DataRequest(nwkAddress, endpoint, 1, uint16(f.clusterId), options, 15, bin.Encode(frm))
	if err == nil {
		zclIncomingMessage, err := f.zcl.ToZclIncomingMessage(response)
		if err == nil {
			zclCommand := zclIncomingMessage.Data.Command.(*cluster.DefaultResponseCommand)
			if err == nil && zclCommand.Status != cluster.ZclStatusSuccess {
				return fmt.Errorf("unable to run command [%d] on cluster [%d]. Status: [%s]. Reason: %s", commandId, f.clusterId, zclCommand.Status)
			}
			return nil
		} else {
			log.Errorf("Unsupported data response message:\n%s\n", func() string { return spew.Sdump(response) })
		}

	}
	return err
}
