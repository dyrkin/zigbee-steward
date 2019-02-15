package steward

import "github.com/dyrkin/zigbee-steward/model"

type Channels struct {
	onDeviceRegistered      chan *model.Device
	onDeviceUnregistered    chan *model.Device
	onDeviceBecameAvailable chan *model.Device
	onDeviceIncomingMessage chan *model.DeviceIncomingMessage
}

func (c *Channels) OnDeviceRegistered() chan *model.Device {
	return c.onDeviceRegistered
}

func (c *Channels) OnDeviceBecameAvailable() chan *model.Device {
	return c.onDeviceBecameAvailable
}

func (c *Channels) OnDeviceUnregistered() chan *model.Device {
	return c.onDeviceUnregistered
}

func (c *Channels) OnDeviceIncomingMessage() chan *model.DeviceIncomingMessage {
	return c.onDeviceIncomingMessage
}
