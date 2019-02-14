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

func (d *Device) SupportedInClusters() []*Cluster {
	return d.supportedClusters(func(e *Endpoint) []*Cluster {
		return e.InClusterList
	})
}

func (d *Device) SupportedOutClusters() []*Cluster {
	return d.supportedClusters(func(e *Endpoint) []*Cluster {
		return e.OutClusterList
	})
}

func (d *Device) supportedClusters(clusterListExtractor func(e *Endpoint) []*Cluster) []*Cluster {
	var clusters []*Cluster
	for _, e := range d.Endpoints {
		for _, c := range clusterListExtractor(e) {
			if c.Supported {
				clusters = append(clusters, c)
			}
		}

	}
	return clusters
}

var powerSourceStrings = map[PowerSource]string{
	Unknown:                         "Unknown",
	MainsSinglePhase:                "MainsSinglePhase",
	Mains2Phase:                     "Mains2Phase",
	Battery:                         "Battery",
	DCSource:                        "DCSource",
	EmergencyMainsConstantlyPowered: "EmergencyMainsConstantlyPowered",
	EmergencyMainsAndTransfer:       "EmergencyMainsAndTransfer",
}

func (ps PowerSource) String() string {
	return powerSourceStrings[ps]
}
