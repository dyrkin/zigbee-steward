package steward

import (
	"github.com/dyrkin/zcl-go/cluster"
)

type LevelControl struct {
	*LocalCluster
}

func (f *LevelControl) MoveToLevel(nwkAddress string, endpoint uint8, level uint8, transitionTime uint16) error {
	return f.localCommand(nwkAddress, endpoint, 0x00, &cluster.MoveToLevelCommand{level, transitionTime})
}

func (f *LevelControl) Move(nwkAddress string, endpoint uint8, moveMode uint8, rate uint8) error {
	return f.localCommand(nwkAddress, endpoint, 0x01, &cluster.MoveCommand{moveMode, rate})
}

func (f *LevelControl) Step(nwkAddress string, endpoint uint8, stepMode uint8, stepSize uint8, transitionTime uint16) error {
	return f.localCommand(nwkAddress, endpoint, 0x02, &cluster.StepCommand{stepMode, stepSize, transitionTime})
}

func (f *LevelControl) Stop(nwkAddress string, endpoint uint8) error {
	return f.localCommand(nwkAddress, endpoint, 0x03, &cluster.StopCommand{})
}

func (f *LevelControl) MoveToLevelOnOff(nwkAddress string, endpoint uint8, level uint8, transitionTime uint16) error {
	return f.localCommand(nwkAddress, endpoint, 0x04, &cluster.MoveToLevelOnOffCommand{level, transitionTime})
}

func (f *LevelControl) MoveOnOff(nwkAddress string, endpoint uint8, moveMode uint8, rate uint8) error {
	return f.localCommand(nwkAddress, endpoint, 0x05, &cluster.MoveOnOffCommand{moveMode, rate})
}

func (f *LevelControl) StepOnOff(nwkAddress string, endpoint uint8, stepMode uint8, stepSize uint8, transitionTime uint16) error {
	return f.localCommand(nwkAddress, endpoint, 0x06, &cluster.StepOnOffCommand{stepMode, stepSize, transitionTime})
}

func (f *LevelControl) StopOnOff(nwkAddress string, endpoint uint8) error {
	return f.localCommand(nwkAddress, endpoint, 0x07, &cluster.StopOnOffCommand{})
}
