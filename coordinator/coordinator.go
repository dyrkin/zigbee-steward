package coordinator

import (
	"fmt"
	"github.com/dyrkin/zcl-go/frame"
	"github.com/tv42/topic"
	"reflect"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/dyrkin/unp-go"
	"github.com/dyrkin/zigbee-steward/configuration"
	"github.com/dyrkin/zigbee-steward/logger"
	"github.com/dyrkin/znp-go"
	"go.bug.st/serial.v1"
)

var log = logger.MustGetLogger("coordinator")

var nextTransactionId = frame.MakeDefaultTransactionIdProvider()

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
}

type Coordinator struct {
	config           *configuration.Configuration
	started          bool
	networkProcessor *znp.Znp
	messageChannels  *MessageChannels
	network          *Network
	broadcast        *topic.Topic
}

func (c *Coordinator) OnIncomingMessage() chan *znp.AfIncomingMessage {
	return c.messageChannels.onIncomingMessage
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
	}
	return &Coordinator{
		config:          config,
		messageChannels: messageChannels,
		network:         &Network{},
		broadcast:       topic.New(),
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
	c.mapMessageChannels()
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

func (c *Coordinator) ActiveEndpoints(nwkAddress string) (*znp.ZdoActiveEpRsp, error) {
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

func (c *Coordinator) NodeDescription(nwkAddress string) (*znp.ZdoNodeDescRsp, error) {
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

func (c *Coordinator) SimpleDescription(nwkAddress string, endpoint uint8) (*znp.ZdoSimpleDescRsp, error) {
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

func (c *Coordinator) DataRequest(dstAddr string, dstEndpoint uint8, srcEndpoint uint8, clusterId uint16, options *znp.AfDataRequestOptions, radius uint8, data []uint8) (*znp.AfIncomingMessage, error) {
	np := c.networkProcessor
	dataRequest := func(networkAddress string, transactionId uint8) error {
		status, err := np.AfDataRequest(networkAddress, dstEndpoint, srcEndpoint, clusterId, transactionId, options, radius, data)
		if err == nil && status.Status != znp.StatusSuccess {
			return fmt.Errorf("unable to unbind. Status: [%s]", status.Status)
		}
		return err
	}

	return c.syncDataRequestRetryable(dataRequest, dstAddr, nextTransactionId(), defaultTimeout, 3)
}

func (c *Coordinator) syncCall(call func() error, expectedType reflect.Type, timeout time.Duration) (interface{}, error) {
	receiver := make(chan interface{})
	responseChannel := make(chan interface{}, 1)
	errorChannel := make(chan error, 1)
	deadline := time.NewTimer(timeout)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		c.broadcast.Register(receiver)
		wg.Done()
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
	wg.Wait()
	err := call()
	if err != nil {
		deadline.Stop()
		c.broadcast.Unregister(receiver)
		return nil, err
	}
	select {
	case err = <-errorChannel:
		c.broadcast.Unregister(receiver)
		return nil, err
	case response := <-responseChannel:
		c.broadcast.Unregister(receiver)
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

func (c *Coordinator) syncDataRequestRetryable(request func(string, uint8) error, nwkAddress string, transactionId uint8, timeout time.Duration, retries int) (*znp.AfIncomingMessage, error) {
	incomingMessage, err := c.syncDataRequest(request, nwkAddress, transactionId, timeout)
	switch {
	case err != nil && retries > 0:
		log.Errorf("%s. Retries: %d", err, retries)
		return c.syncDataRequestRetryable(request, nwkAddress, transactionId, timeout, retries-1)
	case err != nil && retries == 0:
		log.Errorf("failure: %s", err)
		return nil, err
	}
	return incomingMessage, nil
}

func (c *Coordinator) syncDataRequest(request func(string, uint8) error, nwkAddress string, transactionId uint8, timeout time.Duration) (*znp.AfIncomingMessage, error) {
	messageReceiver := make(chan interface{})

	responseChannel := make(chan *znp.AfIncomingMessage, 1)
	errorChannel := make(chan error, 1)

	incomingMessageListener := func() {
		deadline := time.NewTimer(timeout)
		for {
			select {
			case response := <-messageReceiver:
				if incomingMessage, ok := response.(*znp.AfIncomingMessage); ok {
					frm := frame.Decode(incomingMessage.Data)
					if (frm.TransactionSequenceNumber == transactionId) &&
						(incomingMessage.SrcAddr == nwkAddress) {
						deadline.Stop()
						responseChannel <- incomingMessage
						return
					}
				}
			case _ = <-deadline.C:
				if !deadline.Stop() {
					errorChannel <- fmt.Errorf("timeout. didn't receive response for transcation: %d", transactionId)
				}
				return
			}
		}
	}

	confirmListener := func() {
		deadline := time.NewTimer(timeout)
		for {
			c.broadcast.Register(messageReceiver)
			select {
			case response := <-messageReceiver:
				if dataConfirm, ok := response.(*znp.AfDataConfirm); ok {
					if dataConfirm.TransID == transactionId {
						deadline.Stop()
						switch dataConfirm.Status {
						case znp.StatusSuccess:
							go incomingMessageListener()
						default:
							errorChannel <- fmt.Errorf("invalid transcation status: [%s]", dataConfirm.Status)
						}
						return
					}
				}
			case _ = <-deadline.C:
				if !deadline.Stop() {
					errorChannel <- fmt.Errorf("timeout. didn't receive confiramtion for transcation: %d", transactionId)
				}
				return
			}
		}
	}
	go confirmListener()
	err := request(nwkAddress, transactionId)

	if err != nil {
		c.broadcast.Unregister(messageReceiver)
		return nil, fmt.Errorf("unable to send data request: %s", err)
	}

	select {
	case err = <-errorChannel:
		c.broadcast.Unregister(messageReceiver)
		return nil, err
	case zclIncomingMessage := <-responseChannel:
		c.broadcast.Unregister(messageReceiver)
		return zclIncomingMessage, nil
	}
}

func (c *Coordinator) mapMessageChannels() {
	go func() {
		for {
			select {
			case err := <-c.networkProcessor.Errors():
				c.messageChannels.onError <- err
			case incoming := <-c.networkProcessor.AsyncInbound():
				debugIncoming := func(format string) {
					log.Debugf(format, func() string { return spew.Sdump(incoming) })
				}
				c.broadcast.Broadcast <- incoming
				switch message := incoming.(type) {
				case *znp.ZdoEndDeviceAnnceInd:
					debugIncoming("Device announce:\n%s")
					c.messageChannels.onDeviceAnnounce <- message
				case *znp.ZdoLeaveInd:
					debugIncoming("Device leave:\n%s")
					c.messageChannels.onDeviceLeave <- message
				case *znp.ZdoTcDevInd:
					debugIncoming("Device TC:\n%s")
					c.messageChannels.onDeviceTc <- message
				case *znp.AfIncomingMessage:
					debugIncoming("Incoming message:\n%s")
					c.messageChannels.onIncomingMessage <- message
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
