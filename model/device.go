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

func (d *Device) SupportedInClusters() []uint16 {
	var clusters []uint16
	for _, e := range d.Endpoints {
		clusters = append(clusters, e.InClusterList...)
	}
	return clusters
}

func (d *Device) SupportedOutClusters() []uint16 {
	var clusters []uint16
	for _, e := range d.Endpoints {
		clusters = append(clusters, e.OutClusterList...)
	}
	return clusters
}

func (ps PowerSource) String() string {
	switch ps {
	case MainsSinglePhase:
		return "MainsSinglePhase"
	case Mains2Phase:
		return "Mains2Phase"
	case Battery:
		return "Battery"
	case DCSource:
		return "DCSource"
	case EmergencyMainsConstantlyPowered:
		return "EmergencyMainsConstantlyPowered"
	case EmergencyMainsAndTransfer:
		return "EmergencyMainsAndTransfer"
	default:
		return "Unknown"
	}
}
