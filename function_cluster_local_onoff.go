package steward

import (
	"github.com/dyrkin/zcl-go/cluster"
)

type OnOff struct {
	*LocalCluster
}

func (f *OnOff) Off(nwkAddress string, endpoint uint8) error {
	return f.localCommand(nwkAddress, endpoint, 0x00, &cluster.OffCommand{})
}

func (f *OnOff) On(nwkAddress string, endpoint uint8) error {
	return f.localCommand(nwkAddress, endpoint, 0x01, &cluster.OnCommand{})
}

func (f *OnOff) Toggle(nwkAddress string, endpoint uint8) error {
	return f.localCommand(nwkAddress, endpoint, 0x02, &cluster.ToggleCommand{})
}

func (f *OnOff) OffWithEffect(nwkAddress string, endpoint uint8, effectId uint8, effectVariant uint8) error {
	return f.localCommand(nwkAddress, endpoint, 0x40, &cluster.OffWithEffectCommand{effectId, effectVariant})
}

func (f *OnOff) OnWithRecallGlobalScene(nwkAddress string, endpoint uint8, effectId uint8, effectVariant uint8) error {
	return f.localCommand(nwkAddress, endpoint, 0x41, &cluster.OnWithRecallGlobalSceneCommand{})
}

func (f *OnOff) OnWithTimedOff(nwkAddress string, endpoint uint8, onOffControl uint8, onTime uint16, offWaitTime uint16) error {
	return f.localCommand(nwkAddress, endpoint, 0x42, &cluster.OnWithTimedOffCommand{onOffControl, onTime, offWaitTime})
}
