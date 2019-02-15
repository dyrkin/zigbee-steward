package main

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/dyrkin/zcl-go/cluster"
	"github.com/dyrkin/zigbee-steward"
	"github.com/dyrkin/zigbee-steward/configuration"
	"github.com/dyrkin/zigbee-steward/logger"
	"time"
)

func main() {
	conf, err := configuration.Read("configuration.yaml")
	if err != nil {
		panic(err)
	}

	stewie := steward.New(conf)

	log := logger.MustGetLogger("main")

	go func() {
		for {
			select {
			case device := <-stewie.Channels().OnDeviceRegistered():
				log.Infof("Registered device:\n%s", spew.Sdump(device))
				if device.Manufacturer == "IKEA of Sweden" && device.Model == "TRADFRI wireless dimmer" {
					go func() {
						rsp, err := stewie.Functions().Bind(device.NetworkAddress, device.IEEEAddress, 1,
							uint16(cluster.LevelControl), stewie.Configuration().IEEEAddress, 1)
						log.Infof("Bind result: [%v] [%s]", rsp, err)
					}()
				}
			case device := <-stewie.Channels().OnDeviceUnregistered():
				log.Infof("Unregistered device:\n%s", spew.Sdump(device))
			case device := <-stewie.Channels().OnDeviceBecameAvailable():
				log.Infof("Device became available:\n%s", spew.Sdump(device))
			case deviceIncomingMessage := <-stewie.Channels().OnDeviceIncomingMessage():
				log.Infof("Device received incoming message:\n%s", spew.Sdump(deviceIncomingMessage))
			}
		}
	}()

	stewie.Start()
	//Bind IKEA dimmer
	//z.ZdoBindReq(msg.NwkAddr, msg.NwkAddr, 1, uint16(cluster.LevelControl), znp.AddrModeAddr16Bit, "0x0000", 1)
	//z.ZdoBindReq(msg.NwkAddr, msg.IEEEAddr, 1, 8, znp.AddrModeAddr64Bit, "0x00124b00019c2ef9", 1)
	//z.ZdoBindReq(msg.NwkAddr, msg.IEEEAddr, 1, 6, znp.AddrModeAddr64Bit, "0x00124b00019c2ef9", 1)

	time.Sleep(500 * time.Minute)
}
