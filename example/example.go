package main

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/dyrkin/zigbee-steward"
	"github.com/dyrkin/zigbee-steward/logger"
	"time"
)

func main() {

	stewie := steward.New()

	log := logger.MustGetLogger("main")

	go func() {
		for {
			select {
			case registered := <-stewie.Channels().OnDeviceRegistered():
				log.Infof("Registered device:\n%s", spew.Sdump(registered))
			case unregistered := <-stewie.Channels().OnDeviceUnregistered():
				log.Infof("Unregistered device:\n%s", spew.Sdump(unregistered))
			case becameAvailable := <-stewie.Channels().OnDeviceBecameAvailable():
				log.Infof("Device became available:\n%s", spew.Sdump(becameAvailable))
			case deviceIncomingMessage := <-stewie.Channels().OnDeviceIncomingMessage():
				log.Infof("Device received incoming message:\n%s", spew.Sdump(deviceIncomingMessage))
			}
		}
	}()

	stewie.Start("configuration.yaml")
	//Bind IKEA dimmer
	//z.ZdoBindReq(msg.NwkAddr, msg.NwkAddr, 1, uint16(cluster.LevelControl), znp.AddrModeAddr16Bit, "0x0000", 1)
	//z.ZdoBindReq(msg.NwkAddr, msg.IEEEAddr, 1, 8, znp.AddrModeAddr64Bit, "0x00124b00019c2ef9", 1)
	//z.ZdoBindReq(msg.NwkAddr, msg.IEEEAddr, 1, 6, znp.AddrModeAddr64Bit, "0x00124b00019c2ef9", 1)

	time.Sleep(500 * time.Minute)
}
