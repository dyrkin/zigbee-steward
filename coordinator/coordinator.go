package coordinator

import (
	"fmt"
	"github.com/dyrkin/bin"
	"github.com/dyrkin/zcl-go/cluster"
	"github.com/dyrkin/zcl-go/frame"
	"github.com/tv42/topic"
	"reflect"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/dyrkin/unp-go"
	"github.com/dyrkin/zigbee-steward/configuration"
	"github.com/dyrkin/zigbee-steward/logger"
	"github.com/dyrkin/znp-go"
	"go.bug.st/serial.v1"
)

var log = logger.MustGetLogger("coordinator")

const defaultTimeout = 10 * time.Second

type Network struct {
	Address string
}

type MessageChannels struct {
	onError           chan error
	onDeviceAnnounce  chan *znp.ZdoEndDeviceAnnceInd
	onDeviceLeave     chan *znp.ZdoLeaveInd
	onDeviceTc        chan *znp.ZdoTcDevInd
	onIncomingMessage chan *znp.AfIncomingMessage
	onDataConfirm     chan *znp.AfDataConfirm
	broadcast         *topic.Topic
}

type Coordinator struct {
	config           *configuration.Configuration
	started          bool
	networkProcessor *znp.Znp
	messageChannels  *MessageChannels
	network          *Network
}

func (c *Coordinator) OnIncomingMessage() chan *znp.AfIncomingMessage {
	return c.messageChannels.onIncomingMessage
}

func (c *Coordinator) OnDataConfirm() chan *znp.AfDataConfirm {
	return c.messageChannels.onDataConfirm
}

func (c *Coordinator) OnDeviceTc() chan *znp.ZdoTcDevInd {
	return c.messageChannels.onDeviceTc
}

func (c *Coordinator) OnDeviceLeave() chan *znp.ZdoLeaveInd {
	return c.messageChannels.onDeviceLeave
}

func (c *Coordinator) OnDeviceAnnounce() chan *znp.ZdoEndDeviceAnnceInd {
	return c.messageChannels.onDeviceAnnounce
}

func (c *Coordinator) OnError() chan error {
	return c.messageChannels.onError
}

func (c *Coordinator) Network() *Network {
	return c.network
}

func New(config *configuration.Configuration) *Coordinator {
	messageChannels := &MessageChannels{
		onError:           make(chan error, 100),
		onDeviceAnnounce:  make(chan *znp.ZdoEndDeviceAnnceInd, 100),
		onDeviceLeave:     make(chan *znp.ZdoLeaveInd, 100),
		onDeviceTc:        make(chan *znp.ZdoTcDevInd, 100),
		onIncomingMessage: make(chan *znp.AfIncomingMessage, 100),
		onDataConfirm:     make(chan *znp.AfDataConfirm, 100),
		broadcast:         topic.New(),
	}
	return &Coordinator{
		config:          config,
		messageChannels: messageChannels,
		network:         &Network{},
	}
}

func (c *Coordinator) Start() error {
	log.Info("Starting coordinator...")
	port, err := openPort(c.config)
	if err != nil {
		return err
	}
	networkProtocol := unp.New(1, port)
	c.networkProcessor = znp.New(networkProtocol)
	mapMessageChannels(c.messageChannels, c.networkProcessor)
	c.networkProcessor.Start()
	configure(c)
	subscribe(c)
	startup(c)
	enrichNetworkDetails(c)
	switchLed(c)
	registerEndpoints(c)
	permitJoin(c)
	c.started = true
	log.Info("Coordinator started")
	return nil
}

func (c *Coordinator) Reset() {
	reset := func() error {
		return c.networkProcessor.SysResetReq(1)
	}

	_, err := c.syncCallRetryable(reset, SysResetIndType, 15*time.Second, 5)
	if err != nil {
		log.Fatal(err)
	}
}

func (c *Coordinator) ReadAttributes(nwkAddress string, transactionId uint8, attributeIds []uint16) error {
	np := c.networkProcessor
	options := &znp.AfDataRequestOptions{}
	f, err := frame.New().
		DisableDefaultResponse(true).
		FrameType(frame.FrameTypeGlobal).
		Direction(frame.DirectionClientServer).
		CommandId(0x00).
		Command(&cluster.ReadAttributesCommand{AttributeIDs: attributeIds}).
		Build()

	if err != nil {
		return err
	}

	status, err := np.AfDataRequest(nwkAddress, 255, 1, 0x0000, transactionId, options, 15, bin.Encode(f))
	if err == nil && status.Status != znp.StatusSuccess {
		return fmt.Errorf("unable to read attributes. Status: [%s]", status.Status)
	}
	return err
}

func (c *Coordinator) RequestActiveEndpoints(nwkAddress string) (*znp.ZdoActiveEpRsp, error) {
	np := c.networkProcessor
	activeEpReq := func() error {
		status, err := np.ZdoActiveEpReq(nwkAddress, nwkAddress)
		if err == nil && status.Status != znp.StatusSuccess {
			return fmt.Errorf("unable to request active endpoints. Status: [%s]", status.Status)
		}
		return err
	}

	response, err := c.syncCallRetryable(activeEpReq, ZdoActiveEpRspType, defaultTimeout, 3)
	if err == nil {
		return response.(*znp.ZdoActiveEpRsp), nil
	}
	return nil, err
}

func (c *Coordinator) RequestNodeDescription(nwkAddress string) (*znp.ZdoNodeDescRsp, error) {
	np := c.networkProcessor
	activeEpReq := func() error {
		status, err := np.ZdoNodeDescReq(nwkAddress, nwkAddress)
		if err == nil && status.Status != znp.StatusSuccess {
			return fmt.Errorf("unable to request node description. Status: [%s]", status.Status)
		}
		return err
	}

	response, err := c.syncCallRetryable(activeEpReq, ZdoNodeDescRspType, defaultTimeout, 3)
	if err == nil {
		return response.(*znp.ZdoNodeDescRsp), nil
	}
	return nil, err
}

func (c *Coordinator) RequestSimpleDescription(nwkAddress string, endpoint uint8) (*znp.ZdoSimpleDescRsp, error) {
	np := c.networkProcessor
	activeEpReq := func() error {
		status, err := np.ZdoSimpleDescReq(nwkAddress, nwkAddress, endpoint)
		if err == nil && status.Status != znp.StatusSuccess {
			return fmt.Errorf("unable to request simple description. Status: [%s]", status.Status)
		}
		return err
	}

	response, err := c.syncCallRetryable(activeEpReq, ZdoSimpleDescRspType, defaultTimeout, 3)
	if err == nil {
		return response.(*znp.ZdoSimpleDescRsp), nil
	}
	return nil, err
}

func (c *Coordinator) Bind(dstAddr string, srcAddress string, srcEndpoint uint8, clusterId uint16,
	dstAddrMode znp.AddrMode, dstAddress string, dstEndpoint uint8) (*znp.ZdoBindRsp, error) {
	np := c.networkProcessor
	bindReqReq := func() error {
		status, err := np.ZdoBindReq(dstAddr, srcAddress, srcEndpoint, clusterId, dstAddrMode, dstAddress, dstEndpoint)
		if err == nil && status.Status != znp.StatusSuccess {
			return fmt.Errorf("unable to bind. Status: [%s]", status.Status)
		}
		return err
	}

	response, err := c.syncCallRetryable(bindReqReq, ZdoBindRspType, defaultTimeout, 3)
	if err == nil {
		return response.(*znp.ZdoBindRsp), nil
	}
	return nil, err
}

func (c *Coordinator) Unbind(dstAddr string, srcAddress string, srcEndpoint uint8, clusterId uint16,
	dstAddrMode znp.AddrMode, dstAddress string, dstEndpoint uint8) (*znp.ZdoUnbindRsp, error) {
	np := c.networkProcessor
	bindReqReq := func() error {
		status, err := np.ZdoUnbindReq(dstAddr, srcAddress, srcEndpoint, clusterId, dstAddrMode, dstAddress, dstEndpoint)
		if err == nil && status.Status != znp.StatusSuccess {
			return fmt.Errorf("unable to unbind. Status: [%s]", status.Status)
		}
		return err
	}

	response, err := c.syncCallRetryable(bindReqReq, ZdoUnbindRspType, defaultTimeout, 3)
	if err == nil {
		return response.(*znp.ZdoUnbindRsp), nil
	}
	return nil, err
}

func (c *Coordinator) syncCall(call func() error, expectedType reflect.Type, timeout time.Duration) (interface{}, error) {
	receiver := make(chan interface{})
	responseChannel := make(chan interface{}, 1)
	errorChannel := make(chan error, 1)
	deadline := time.NewTimer(timeout)
	go func() {
		c.messageChannels.broadcast.Register(receiver)
		for {
			select {
			case response := <-receiver:
				if reflect.TypeOf(response) == expectedType {
					deadline.Stop()
					responseChannel <- response
					return
				}
			case _ = <-deadline.C:
				if !deadline.Stop() {
					errorChannel <- fmt.Errorf("timeout. didn't receive response of type: %s", expectedType)
				}
				return
			}
		}
	}()
	err := call()
	if err != nil {
		deadline.Stop()
		c.messageChannels.broadcast.Unregister(receiver)
		return nil, err
	}
	select {
	case err = <-errorChannel:
		c.messageChannels.broadcast.Unregister(receiver)
		return nil, err
	case response := <-responseChannel:
		c.messageChannels.broadcast.Unregister(receiver)
		return response, nil
	}
}

func (c *Coordinator) syncCallRetryable(call func() error, expectedType reflect.Type, timeout time.Duration, retries int) (interface{}, error) {
	response, err := c.syncCall(call, expectedType, timeout)
	switch {
	case err != nil && retries > 0:
		log.Errorf("%s. Retries: %d", err, retries)
		return c.syncCallRetryable(call, expectedType, timeout, retries-1)
	case err != nil && retries == 0:
		log.Errorf("failure: %s", err)
		return nil, err
	}
	return response, nil
}

func mapMessageChannels(messageChannels *MessageChannels, networkProcessor *znp.Znp) {
	go func() {
		for {
			select {
			case err := <-networkProcessor.Errors():
				messageChannels.onError <- err
			case incoming := <-networkProcessor.AsyncInbound():
				debugIncoming := func(format string) {
					log.Debugf(format, func() string { return spew.Sdump(incoming) })
				}
				switch message := incoming.(type) {
				case *znp.ZdoEndDeviceAnnceInd:
					debugIncoming("Device announce:\n%s")
					messageChannels.onDeviceAnnounce <- message
				case *znp.ZdoLeaveInd:
					debugIncoming("Device leave:\n%s")
					messageChannels.onDeviceLeave <- message
				case *znp.ZdoTcDevInd:
					debugIncoming("Device TC:\n%s")
					messageChannels.onDeviceTc <- message
				case *znp.AfIncomingMessage:
					debugIncoming("Incoming message:\n%s")
					messageChannels.onIncomingMessage <- message
				case *znp.AfDataConfirm:
					debugIncoming("Data confirm:\n%s")
					messageChannels.onDataConfirm <- message
				default:
					debugIncoming("Message:\n%s")
					messageChannels.broadcast.Broadcast <- message
				}
			}
		}
	}()
}

func configure(coordinator *Coordinator) {
	coordinator.Reset()
	np := coordinator.networkProcessor

	mandatorySetting := func(call func() error) {
		err := call()
		if err != nil {
			log.Fatal(err)
		}
	}

	t := time.Now()
	np.SysSetTime(0, uint8(t.Hour()), uint8(t.Minute()), uint8(t.Second()),
		uint8(t.Month()), uint8(t.Day()), uint16(t.Year()))

	mandatorySetting(func() error {
		_, err := np.UtilSetPreCfgKey(coordinator.config.NetworkKey)
		return err
	})
	//logical type
	mandatorySetting(func() error {
		_, err := np.SapiZbWriteConfiguration(0x87, []uint8{0})
		return err
	})
	mandatorySetting(func() error {
		_, err := np.UtilSetPanId(coordinator.config.PanId)
		return err
	})
	//zdo direc cb
	mandatorySetting(func() error {
		_, err := np.SapiZbWriteConfiguration(0x8F, []uint8{1})
		return err
	})
	//enable security
	mandatorySetting(func() error {
		_, err := np.SapiZbWriteConfiguration(0x64, []uint8{1})
		return err
	})
	mandatorySetting(func() error {
		_, err := np.SysSetExtAddr(coordinator.config.IEEEAddress)
		return err
	})

	channels := &znp.Channels{}
	for _, v := range coordinator.config.Channels {
		switch v {
		case 11:
			channels.Channel11 = 1
		case 12:
			channels.Channel12 = 1
		case 13:
			channels.Channel13 = 1
		case 14:
			channels.Channel14 = 1
		case 15:
			channels.Channel15 = 1
		case 16:
			channels.Channel16 = 1
		case 17:
			channels.Channel17 = 1
		case 18:
			channels.Channel18 = 1
		case 19:
			channels.Channel19 = 1
		case 20:
			channels.Channel20 = 1
		case 21:
			channels.Channel21 = 1
		case 22:
			channels.Channel22 = 1
		case 23:
			channels.Channel23 = 1
		case 24:
			channels.Channel24 = 1
		case 25:
			channels.Channel25 = 1
		case 26:
			channels.Channel26 = 1
		}
	}

	mandatorySetting(func() error {
		_, err := coordinator.networkProcessor.UtilSetChannels(channels)
		return err
	})
	coordinator.Reset()
}

func subscribe(coordinator *Coordinator) {
	np := coordinator.networkProcessor
	_, err := np.UtilCallbackSubCmd(znp.SubsystemIdAllSubsystems, znp.ActionEnable)

	if err != nil {
		log.Fatal(err)
	}
}

func registerEndpoints(coordinator *Coordinator) {
	np := coordinator.networkProcessor
	np.AfRegister(0x01, 0x0104, 0x0005, 0x1, znp.LatencyNoLatency, []uint16{}, []uint16{})

	np.AfRegister(0x02, 0x0101, 0x0005, 0x1, znp.LatencyNoLatency, []uint16{}, []uint16{})

	np.AfRegister(0x03, 0x0105, 0x0005, 0x1, znp.LatencyNoLatency, []uint16{}, []uint16{})

	np.AfRegister(0x04, 0x0107, 0x0005, 0x1, znp.LatencyNoLatency, []uint16{}, []uint16{})

	np.AfRegister(0x05, 0x0108, 0x0005, 0x1, znp.LatencyNoLatency, []uint16{}, []uint16{})

	np.AfRegister(0x06, 0x0109, 0x0005, 0x1, znp.LatencyNoLatency, []uint16{}, []uint16{})
}

func enrichNetworkDetails(coordinator *Coordinator) {
	deviceInfo, err := coordinator.networkProcessor.UtilGetDeviceInfo()
	if err != nil {
		log.Fatal(err)
	}
	coordinator.network.Address = deviceInfo.ShortAddr
}

func permitJoin(coordinator *Coordinator) {
	var timeout uint8 = 0x00
	if coordinator.config.PermitJoin {
		timeout = 0xFF
	}
	coordinator.networkProcessor.SapiZbPermitJoiningRequest(coordinator.network.Address, timeout)
}

func switchLed(coordinator *Coordinator) {
	mode := znp.ModeOFF
	if coordinator.config.Led {
		mode = znp.ModeON
	}
	log.Debugf("Led mode [%s]", mode)
	coordinator.networkProcessor.UtilLedControl(1, mode)
}

func startup(coordinator *Coordinator) {
	_, err := coordinator.networkProcessor.SapiZbStartRequest()
	if err != nil {
		log.Fatal(err)
	}
}

func openPort(config *configuration.Configuration) (port serial.Port, err error) {
	log.Debugf("Opening port [%s] at rate [%d]", config.Serial.PortName, config.Serial.BaudRate)
	mode := &serial.Mode{BaudRate: config.Serial.BaudRate}
	port, err = serial.Open(config.Serial.PortName, mode)
	if err == nil {
		log.Debugf("Port [%s] is opened", config.Serial.PortName)
		err = port.SetRTS(true)
	}
	return
}
