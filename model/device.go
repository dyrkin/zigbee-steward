package model

import "github.com/dyrkin/znp-go"

type PowerSource uint8

const (
	Unknown PowerSource = iota
	MainsSinglePhase
	Mains2Phase
	Battery
	DCSource
	EmergencyMainsConstantlyPowered
	EmergencyMainsAndTransfer
)

type Device struct {
	Manufacturer   string
	ManufacturerId uint16
	Model          string
	LogicalType    znp.LogicalType
	MainPowered    bool
	PowerSource    PowerSource
	NetworkAddress string
	IEEEAddress    string
	Endpoints      []*Endpoint
}
