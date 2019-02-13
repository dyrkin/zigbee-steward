package main

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/dyrkin/bin"
	"github.com/dyrkin/unp-go"
	"github.com/dyrkin/zcl-go"
	"github.com/dyrkin/zcl-go/cluster"
	"github.com/dyrkin/zcl-go/frame"
	"github.com/dyrkin/zigbee-steward"
	"github.com/dyrkin/znp-go"
	"go.bug.st/serial.v1"
	"log"
	"reflect"
	"time"
)

func main() {

	stew := steward.New()

	stew.Start("configuration.yaml")
	//Bind IKEA dimmer
	//						// z.ZdoBindReq(msg.NwkAddr, msg.NwkAddr, 1, uint16(cluster.LevelControl), znp.AddrModeAddr16Bit, "0x0000", 1)
	//						z.ZdoBindReq(msg.NwkAddr, msg.IEEEAddr, 1, 8, znp.AddrModeAddr64Bit, "0x00124b00019c2ef9", 1)
	//						z.ZdoBindReq(msg.NwkAddr, msg.IEEEAddr, 1, 6, znp.AddrModeAddr64Bit, "0x00124b00019c2ef9", 1)

	time.Sleep(500 * time.Minute)
}

//TODO remove this
type System struct {
	z *znp.Znp
}

var devices = map[string]*znp.ZdoEndDeviceAnnceInd{}

var callbacks = map[reflect.Type]chan interface{}{}

func main2() {
	mode := &serial.Mode{BaudRate: 115200}

	port, err := serial.Open("/dev/tty.usbmodem14101", mode)
	if err != nil {
		log.Fatal(err)
	}
	port.SetRTS(true)

	u := unp.New(1, port)
	z := znp.New(u)
	z.Start()
	system := &System{z}
	zc := zcl.New()

	// 	65281 - 0xFF01 report:
	// { '1': 3069,	=	Battery
	// '3': 23, 			= soc_temperature
	// '4': 5117,
	// '5': 34,
	// '6': 0,
	// '10': 0 }

	var tId uint8 = 0

	transId := func() uint8 {
		tId = tId + 1
		if tId > 255 {
			tId = 1
		}
		return tId
	}

	go func() {
		for {
			select {
			case err := <-z.Errors():
				fmt.Printf("Error: %s\n", err)
			case async := <-z.AsyncInbound():
				switch msg := async.(type) {
				case *znp.ZdoEndDeviceAnnceInd:
					fmt.Printf("ZdoEndDeviceAnnceInd: %s\n", spew.Sdump(msg))
					go func() {

						options := &znp.AfDataRequestOptions{}

						devices[msg.NwkAddr] = msg
						z.ZdoNodeDescReq(msg.NwkAddr, msg.NwkAddr)
						f, err := frame.New().
							DisableDefaultResponse(true).
							FrameType(frame.FrameTypeGlobal).
							Direction(frame.DirectionClientServer).
							CommandId(0x00).
							Command(&cluster.ReadAttributesCommand{AttributeIDs: []uint16{0x0004, 0x0005, 0x0007}}).
							Build()
						if err != nil {
							panic("Wrong frame")
						}

						z.AfDataRequest(msg.NwkAddr, 255, 1, 0x0000, transId(), options, 15, bin.Encode(f))

						// frame, _ = zc.GlobalFrame(zcl.CommandByName("ConfigureReporting"), []*cluster.AttributeReportingConfigurationRecord{
						// 	&cluster.AttributeReportingConfigurationRecord{
						// 		cluster.ReportDirectionAttributeReported, "", 33, cluster.ZclDataTypeUint8, 0, 58000, &cluster.Attribute{cluster.ZclDataTypeNoData, 0}, 0,
						// 	},
						// })

						// z.AfDataRequest(msg.NwkAddr, 255, 1, uint16(cluster.PowerConfiguration), transId(), options, 15, bin.Encode(frame))

						// frame, _ = zc.GlobalFrame(zcl.CommandByName("ConfigureReporting"), []*cluster.AttributeReportingConfigurationRecord{
						// 	&cluster.AttributeReportingConfigurationRecord{
						// 		cluster.ReportDirectionAttributeReported, "", 0, cluster.ZclDataTypeUint8, 0, 1000, &cluster.Attribute{cluster.ZclDataTypeUint8, uint64(0)}, 0,
						// 	},
						// })

						// z.AfDataRequest(msg.NwkAddr, 255, 1, uint16(cluster.LevelControl), transId(), options, 15, bin.Encode(frame))

						// frame, _ = zc.GlobalFrame(zcl.CommandByName("ConfigureReporting"), []*cluster.AttributeReportingConfigurationRecord{
						// 	&cluster.AttributeReportingConfigurationRecord{
						// 		cluster.ReportDirectionAttributeReported, "", 0, cluster.ZclDataTypeBoolean, 0, 1000, &cluster.Attribute{cluster.ZclDataTypeBoolean, false}, 0,
						// 	},
						// })

						// z.AfDataRequest(msg.NwkAddr, 255, 1, uint16(6), transId(), options, 15, bin.Encode(frame))
						// z.ZdoBindReq(msg.NwkAddr, msg.NwkAddr, 1, uint16(cluster.LevelControl), znp.AddrModeAddr16Bit, "0x0000", 1)
						z.ZdoBindReq(msg.NwkAddr, msg.IEEEAddr, 1, 8, znp.AddrModeAddr64Bit, "0x00124b00019c2ef9", 1)
						z.ZdoBindReq(msg.NwkAddr, msg.IEEEAddr, 1, 6, znp.AddrModeAddr64Bit, "0x00124b00019c2ef9", 1)
					}()

				case *znp.ZdoLeaveInd:
					fmt.Printf("ZdoLeaveInd request: %s\n", spew.Sdump(msg))
				case *znp.ZdoTcDevInd:
					fmt.Printf("ZdoTcDevInd request: %s\n", spew.Sdump(msg))

				case *znp.ZdoNodeDescRsp:
					fmt.Printf("ZdoNodeDescRsp: %s\n", spew.Sdump(msg))
					z.ZdoActiveEpReq(msg.SrcAddr, msg.SrcAddr)
				case *znp.ZdoActiveEpRsp:
					fmt.Printf("Active EP desc: %s\n", spew.Sdump(msg))
					for _, v := range msg.ActiveEPList {
						fmt.Printf("Sending simple desc req:%d\n", v)
						status, _ := z.ZdoSimpleDescReq(msg.NWKAddr, msg.NWKAddr, v)
						fmt.Printf("Sent. Status: %s\n", status)
					}
					status, _ := z.ZdoUserDescReq(msg.NWKAddr, msg.NWKAddr)
					fmt.Printf("Sent ZdoUserDescReq. Status: %s\n", status)
				case *znp.AfIncomingMessage:
					//fmt.Printf("AfIncomingMessage:\n%s\n", spew.Sdump(msg))
					//frame := frame.Decode(msg.Data)
					//fmt.Printf("Zcl Frame:\n%s\n", spew.Sdump(frame))
					res, err := zc.ToZclIncomingMessage(msg)
					if err == nil {
						fmt.Printf("Foundation Frame Payload\n%s\n", spew.Sdump(res))
					} else {
						fmt.Println(err)
					}
				case *znp.ZdoSimpleDescRsp:
					fmt.Printf("ZdoSimpleDescRsp:\n%s\n", spew.Sdump(msg))
				case *znp.ZdoMsgCbIncoming:
					fmt.Printf("ZdoMsgCbIncoming:\n%s\n", spew.Sdump(msg))
					frame := frame.Decode(msg.Data)
					fmt.Printf("Zcl Frame:\n%s\n", spew.Sdump(frame))
					// res, err := zc.ToZclIncomingMessage(msg)
					// if err == nil {
					// 	fmt.Printf("Foundation Frame Payload\n%s\n", spew.Sdump(res))
					// } else {
					// 	fmt.Println(err)
					// }
				default:
					tp := reflect.TypeOf(async)
					if callback, ok := callbacks[tp]; ok {
						delete(callbacks, tp)
						callback <- async
					} else {
						fmt.Printf("Unknown async: %s\n", spew.Sdump(async))
					}
				}

			case frame := <-z.OutFramesLog():
				fmt.Printf("Frame sent: %s\n", spew.Sdump(frame))
			case frame := <-z.InFramesLog():
				fmt.Printf("Frame received: %s\n", spew.Sdump(frame))
			}
		}
	}()

	system.initializeCoordinator()
	z.Start()
}

func (s *System) await(response interface{}) interface{} {
	tp := reflect.TypeOf(response)
	ch := make(chan interface{}, 1)
	callbacks[tp] = ch
	return <-ch
}

func (s *System) reset() *znp.SysResetInd {
	s.z.SysResetReq(1)
	return s.await(&znp.SysResetInd{}).(*znp.SysResetInd)
}

func (s *System) version() *znp.SysVersionResponse {
	version, _ := s.z.SysVersion()
	return version
}

func (s *System) writeConfig() {
	networkKey := [16]uint8{1, 3, 5, 7, 9, 11, 13, 15, 0, 2, 4, 6, 8, 10, 12, 13}
	_, err := s.z.UtilSetPreCfgKey(networkKey)

	if err != nil {
		log.Fatal(err)
	}

	// s.z.SapiZbSystemReset()

	// linkKey := [16]uint8{1, 3, 5, 7, 9, 11, 13, 15, 0, 2, 4, 6, 8, 10, 12, 13}
	// _, err = s.z.ZdoSetLinkKey()

	if err != nil {
		log.Fatal(err)
	}

	// //security
	_, err = s.z.SapiZbWriteConfiguration(0x87, []uint8{0}) //logical type

	if err != nil {
		log.Fatal(err)
	}

	_, err = s.z.UtilSetPanId(0x1a62)

	if err != nil {
		log.Fatal(err)
	}

	//zdo direc cb
	_, err = s.z.SapiZbWriteConfiguration(0x8F, []uint8{1})

	if err != nil {
		log.Fatal(err)
	}

	//enable security
	_, err = s.z.SapiZbWriteConfiguration(0x64, []uint8{1})

	if err != nil {
		log.Fatal(err)
	}

	_, err = s.z.SysSetExtAddr("0x00124b00019c2ef9")

	if err != nil {
		log.Fatal(err)
	}

	//
	// _, err = s.z.SapiZbWriteConfiguration(0x03, []uint8{0x04})

	// if err != nil {
	// 	log.Fatal(err)
	// }
	//

	//

	// // //extended pan id
	// _, err = s.z.SapiZbWriteConfiguration(0x2D, []uint8{0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD})

	// if err != nil {
	// 	log.Fatal(err)
	// }

	//
	// //logical type
	// _, err = s.z.SapiZbWriteConfiguration(0x63, []uint8{0})

	// if err != nil {
	// 	log.Fatal(err)
	// }

	// //logical type
	// _, err = s.z.SapiZbWriteConfiguration(0x63, []uint8{0})

	// if err != nil {
	// 	log.Fatal(err)
	// }

	_, err = s.z.UtilSetChannels(&znp.Channels{Channel11: 1})

	if err != nil {
		log.Fatal(err)
	}

}

func (s *System) subscribe() {
	_, err := s.z.UtilCallbackSubCmd(znp.SubsystemIdAllSubsystems, znp.ActionEnable)

	_, err = s.z.UtilCallbackSubCmd(znp.SubsystemIdAf, znp.ActionEnable)

	if err != nil {
		log.Fatal(err)
	}

	_, err = s.z.ZdoMsgCbRegister(uint16(cluster.Basic))

	if err != nil {
		log.Fatal(err)
	}

}

func (s *System) printInfo() {
	nvInfo, err := s.z.UtilGetNvInfo()

	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Network Info: %s\n", spew.Sdump(nvInfo))

	devInfo, err := s.z.UtilGetDeviceInfo()

	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Device Info: %s\n", spew.Sdump(devInfo))
}

func (s *System) initializeCoordinator() {
	resetInd := s.reset()
	fmt.Printf("Reset response: %s\n", spew.Sdump(resetInd))
	version := s.version()
	fmt.Printf("Version response: %s\n", spew.Sdump(version))

	s.writeConfig()

	s.reset()
	fmt.Println("Configured")
	s.subscribe()
	fmt.Println("Subscribed")

	s.z.ZdoStartupFromApp(30)
	s.z.UtilLedControl(1, znp.ModeOFF)
	// s.z.SapiZbStartRequest()
	s.printInfo()
	s.z.AfRegister(0x01, 0x0104, 0x0005, 0x1, znp.LatencyNoLatency, []uint16{}, []uint16{})

	s.z.AfRegister(0x02, 0x0101, 0x0005, 0x1, znp.LatencyNoLatency, []uint16{}, []uint16{})

	s.z.AfRegister(0x03, 0x0105, 0x0005, 0x1, znp.LatencyNoLatency, []uint16{}, []uint16{})

	s.z.AfRegister(0x04, 0x0107, 0x0005, 0x1, znp.LatencyNoLatency, []uint16{}, []uint16{})

	s.z.AfRegister(0x05, 0x0108, 0x0005, 0x1, znp.LatencyNoLatency, []uint16{}, []uint16{})

	s.z.AfRegister(0x06, 0x0109, 0x0005, 0x1, znp.LatencyNoLatency, []uint16{}, []uint16{})

	//s.z.AfRegister(0x09, 260, 2064, 0x2, znp.LatencyNoLatency, []uint16{3, 4, 6, 8, 25, 4096}, []uint16{0, 1, 3, 9, 2821, 4096})
	//
	// s.z.ZdoSimpleDescReq("0x0000", "0x0000", 1)
	// s.z.AfRegister(0x1, 0x104, 0x0400, 0x1, znp.LatencyNoLatency, []uint16{uint16(cluster.Basic), uint16(cluster.Identify)}, []uint16{uint16(cluster.Basic), uint16(cluster.Identify)})

	// s.z.SapiZbStartRequest()
	// s.z.ZdoMgmtPermitJoinReq(znp.AddrModeAddrBroadcast, "0xFFFC", 255, 0)

	s.z.SapiZbPermitJoiningRequest("0x0000", 0xff)

	fmt.Println("Started")

	time.Sleep(500 * time.Minute)
}
