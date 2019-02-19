package configuration

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

func Default() *Configuration {
	return &Configuration{
		PermitJoin:  false,
		IEEEAddress: "0x7a2d6265656e6574",
		PanId:       0x1234,
		NetworkKey:  [16]uint8{4, 3, 2, 1, 9, 8, 7, 6, 255, 254, 253, 252, 50, 49, 48, 47},
		Channels:    []uint8{11, 12},
		Led:         false,
		Serial: &Serial{
			PortName: "/dev/tty.usbmodem14101",
			BaudRate: 115200,
		},
	}
}
