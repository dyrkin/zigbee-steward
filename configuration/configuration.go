package configuration

import (
	"github.com/dyrkin/zigbee-steward/yml"
)

type Serial struct {
	PortName string
	BaudRate int
}

type Configuration struct {
	PermitJoin  bool
	IEEEAddress string
	PanId       uint16
	NetworkKey  [16]uint8
	Channels    []uint8
	Led         bool
	Serial      *Serial
}

func Read(path string) (*Configuration, error) {
	configuration := &Configuration{}
	if err := yml.Read(path, configuration); err != nil {
		return nil, err
	}
	return configuration, nil
}

func Write(path string, configuration *Configuration) error {
	return yml.Write(path, configuration)
}
