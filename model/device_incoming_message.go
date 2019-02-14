package model

import "github.com/dyrkin/zcl-go"

type DeviceIncomingMessage struct {
	Device          *Device
	IncomingMessage *zcl.ZclIncomingMessage
}
