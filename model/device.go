package model

type PowerSource uint8

const (
	Unknown                         PowerSource = 0
	MainsSinglePhase                PowerSource = 1
	Mains2Phase                     PowerSource = 2
	Battery                         PowerSource = 3
	DCSource                        PowerSource = 4
	EmergencyMainsConstantlyPowered PowerSource = 5
	EmergencyMainsAndTransfer       PowerSource = 6
)

type Device struct {
	Manufacturer   string
	ManufacturerId uint16
	Model          string
	PowerSource    PowerSource
	IEEEAddress    string
	Endpoints      []*Endpoint
}
