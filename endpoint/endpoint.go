package endpoint

import (
	znp "github.com/dyrkin/znp-go"
)

type Endpoint struct {
	Endpoint       uint8
	ProfileID      uint16
	DeviceID       uint16
	DeviceVersion  uint8
	InClusterList  []uint16
	OutClusterList []uint16
}

func New(endpoint uint8, profileId uint16, deviceId uint16, deviceVersion uint8, inClusterList []uint16, outClusterList []uint16) *Endpoint {
	return &Endpoint{
		endpoint,
		profileId,
		deviceId,
		deviceVersion,
		inClusterList,
		outClusterList,
	}
}

func NewFromSimpleDescRsp(rsp *znp.ZdoSimpleDescRsp) *Endpoint {
	return New(rsp.Endpoint, rsp.ProfileID, rsp.DeviceID, rsp.DeviceVersion, rsp.InClusterList, rsp.OutClusterList)
}
