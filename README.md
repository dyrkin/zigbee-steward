# ZigBee-Steward

[![Build Status](https://cloud.drone.io/api/badges/dyrkin/zigbee-steward/status.svg??branch=master)](https://cloud.drone.io/dyrkin/zigbee-steward)

## Overview

Bla bla bla

## Example

In this example, we are going to send **toggle** command to IKEA TRÅDFRI bulb using Xiaomi  Aqara Wireless Remote Switch. 
To prepare, follow the steps:

1. Connect ZigBee stick cc2531 to USB;
2. Flash it using instruction: https://www.zigbee2mqtt.io/getting_started/flashing_the_cc2531.html;
3. Run the example;
4. Pair IKEA TRÅDFRI bulb by repeating 6 ONs and 5 OFFs one by one (ON -> OFF, ON -> OFF, ON -> OFF, ON -> OFF, ON -> OFF, ON);
5. Pair Xiaomi  Aqara Wireless Remote Switch by holding the button for ~10 seconds until blue LEDs start blinking.

Now you can toggle the bulb using the remote switch. Just click on it.


```go
import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/dyrkin/zcl-go/cluster"
	"github.com/dyrkin/zigbee-steward"
	"github.com/dyrkin/zigbee-steward/configuration"
	"github.com/dyrkin/zigbee-steward/model"
	"sync"
)

//simple device database
var devices = map[string]*model.Device{}

func main() {

	conf := configuration.Default()
	conf.PermitJoin = true

	stewie := steward.New(conf)

	eventListener := func() {
		for {
			select {
			case device := <-stewie.Channels().OnDeviceRegistered():
				saveDevice(device)
			case device := <-stewie.Channels().OnDeviceUnregistered():
				deleteDevice(device)
			case device := <-stewie.Channels().OnDeviceBecameAvailable():
				saveDevice(device)
			case deviceIncomingMessage := <-stewie.Channels().OnDeviceIncomingMessage():
				fmt.Printf("Device received incoming message:\n%s", spew.Sdump(deviceIncomingMessage))
				toggleIkeaBulb(stewie, deviceIncomingMessage)
			}
		}
	}

	go eventListener()
	stewie.Start()
	infiniteWait()
}

func toggleIkeaBulb(stewie *steward.Steward, message *model.DeviceIncomingMessage) {
	if isXiaomiButtonSingleClick(message) {
		if ikeaBulb, registered := devices["TRADFRI bulb E27 W opal 1000lm"]; registered {
			toggleTarget(stewie, ikeaBulb.NetworkAddress)
		} else {
			fmt.Println("IKEA bulb is not available")
		}
	}
}

func toggleTarget(stewie *steward.Steward, networkAddress string) {
	go func() {
		stewie.Functions().Cluster().Local().OnOff().Toggle(networkAddress, 0xFF)
	}()
}

func isXiaomiButtonSingleClick(message *model.DeviceIncomingMessage) bool {
	command, ok := message.IncomingMessage.Data.Command.(*cluster.ReportAttributesCommand)

	return ok && message.Device.Manufacturer == "LUMI" &&
		message.Device.Model == "lumi.remote.b186acn01\x00\x00\x00" &&
		isSingleClick(command)
}

func isSingleClick(command *cluster.ReportAttributesCommand) bool {
	click, ok := command.AttributeReports[0].Attribute.Value.(uint64)
	return ok && click == uint64(1)
}

func saveDevice(device *model.Device) {
	fmt.Printf("Registering device:\n%s", spew.Sdump(device))
	devices[device.Model] = device
}

func deleteDevice(device *model.Device) {
	fmt.Printf("Unregistering device:\n%s", spew.Sdump(device))
	delete(devices, device.Model)
}

func infiniteWait() {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
```

Full [examples](example/example.go)