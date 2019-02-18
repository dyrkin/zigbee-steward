package main

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/dyrkin/zcl-go/cluster"
	"github.com/dyrkin/zigbee-steward"
	"github.com/dyrkin/zigbee-steward/configuration"
	"github.com/dyrkin/zigbee-steward/logger"
	"sync"
)

func main() {
	log := logger.MustGetLogger("main")

	conf, err := configuration.Read("configuration.yaml")
	if err != nil {
		panic(err)
	}

	stewie := steward.New(conf)

	go func() {
		for {
			select {
			case device := <-stewie.Channels().OnDeviceRegistered():
				log.Infof("Registered device:\n%s", spew.Sdump(device))
				if device.Manufacturer == "IKEA of Sweden" && device.Model == "TRADFRI wireless dimmer" {
					go func() {
						rsp, err := stewie.Functions().Generic().Bind(device.NetworkAddress, device.IEEEAddress, 1,
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
	infiniteWait()
}

func infiniteWait() {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
