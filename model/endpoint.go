package model

type Endpoint struct {
	Id             uint8
	ProfileId      uint16
	DeviceId       uint16
	DeviceVersion  uint16
	InClusterList  []*uint16
	OutClusterList []*uint16
}
