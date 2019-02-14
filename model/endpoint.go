package model

type Endpoint struct {
	Id             uint8
	ProfileId      uint16
	DeviceId       uint16
	DeviceVersion  uint8
	InClusterList  []*Cluster
	OutClusterList []*Cluster
}
