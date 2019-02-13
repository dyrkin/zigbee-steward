package model

import "github.com/dyrkin/znp-go"

type Endpoint struct {
	Id             uint8
	ProfileId      uint16
	DeviceId       uint16
	DeviceVersion  uint8
	InClusterList  []uint16
	OutClusterList []uint16
}

func NewEndpoint(rsp *znp.ZdoSimpleDescRsp) *Endpoint {
	return &Endpoint{rsp.Endpoint, rsp.ProfileID, rsp.DeviceID, rsp.DeviceVersion, rsp.InClusterList, rsp.OutClusterList}
}
