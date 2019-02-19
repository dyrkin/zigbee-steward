package main

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/dyrkin/zcl-go/cluster"
	"github.com/dyrkin/zigbee-steward"
	"github.com/dyrkin/zigbee-steward/configuration"
	"github.com/dyrkin/zigbee-steward/logger"
	"math/rand"
	"sync"
)

func main() {
	log := logger.MustGetLogger("main")

	conf := configuration.Default()

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
				//Set random brightness to IKEA bulb
				if device.Manufacturer == "IKEA of Sweden" && device.Model == "TRADFRI bulb E27 W opal 1000lm" {
					go func() {
						err := stewie.Functions().Cluster().Local().LevelControl().
							MoveToLevel(device.NetworkAddress, 1, uint8(rand.Intn(255)), 0xffff)
						log.Infof("Move to level error if present: [%s]", err)
					}()
				}
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
