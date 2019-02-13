package coordinator

import (
	"fmt"
	"github.com/tv42/topic"
	"log"
	"reflect"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/dyrkin/unp-go"
	"github.com/dyrkin/zigbee-steward/configuration"
	"github.com/dyrkin/zigbee-steward/logger"
	"github.com/dyrkin/znp-go"
	"go.bug.st/serial.v1"
)

type MessageChannels struct {
	onError           chan error
	onDeviceAnnounce  chan *znp.ZdoEndDeviceAnnceInd
	onDeviceLeave     chan *znp.ZdoLeaveInd
	onDeviceTc        chan *znp.ZdoTcDevInd
	onIncomingMessage chan *znp.AfIncomingMessage
	broadcast         *topic.Topic
}

type Coordinator struct {
	config           *configuration.Configuration
	started          bool
	networkProcessor *znp.Znp
	messageChannels  *MessageChannels
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

func New(config *configuration.Configuration) *Coordinator {
	messageChannels := &MessageChannels{
		onError:           make(chan error, 100),
		onDeviceAnnounce:  make(chan *znp.ZdoEndDeviceAnnceInd, 100),
		onDeviceLeave:     make(chan *znp.ZdoLeaveInd, 100),
		onDeviceTc:        make(chan *znp.ZdoTcDevInd, 100),
		onIncomingMessage: make(chan *znp.AfIncomingMessage, 100),
		broadcast:         topic.New(),
	}
	return &Coordinator{
		config:          config,
		messageChannels: messageChannels,
	}
}

func (c *Coordinator) Start() error {
	logger.Info("Starting coordinator...")
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
	switchLed(c)
	registerEndpoints(c)
	c.started = true
	logger.Info("Coordinator started")
	return nil
}

func switchLed(coordinator *Coordinator) {
	mode := znp.ModeOFF
	if coordinator.config.Led {
		mode = znp.ModeON
	}
	coordinator.networkProcessor.UtilLedControl(1, mode)
}

func startup(coordinator *Coordinator) {
	_, err := coordinator.networkProcessor.ZdoStartupFromApp(30)
	if err != nil {
		log.Fatal(err)
	}
}

func (c *Coordinator) Reset() {
	reset := func() {
		c.networkProcessor.SysResetReq(1)
	}
	_, err := c.SyncCallRetryable(reset, SysResetIndType, 10*time.Second, 3)
	if err != nil {
		log.Fatal(err)
	}
}

func (c *Coordinator) SyncCall(call func(), expectedType reflect.Type, timeout time.Duration) (interface{}, error) {
	receiver := make(chan interface{})
	c.messageChannels.broadcast.Register(receiver)
	responseChannel := make(chan interface{}, 1)
	errorChannel := make(chan error, 1)
	deadline := time.NewTimer(timeout)
	go func() {
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
	call()
	select {
	case err := <-errorChannel:
		c.messageChannels.broadcast.Unregister(receiver)
		return nil, err
	case response := <-responseChannel:
		c.messageChannels.broadcast.Unregister(receiver)
		return response, nil
	}
}

func (c *Coordinator) SyncCallRetryable(call func(), expectedType reflect.Type, timeout time.Duration, retries int) (interface{}, error) {
	response, err := c.SyncCall(call, expectedType, timeout)
	switch {
	case err != nil && retries > 0:
		logger.Errorf("%s. Retries: %d", err, retries)
		return c.SyncCallRetryable(call, expectedType, timeout, retries-1)
	case err != nil && retries == 0:
		logger.Errorf("failure: %s", err)
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
					logger.Debugf(format, func() string { return spew.Sdump(incoming) })
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

	_, err := np.UtilSetPreCfgKey(coordinator.config.NetworkKey)

	if err != nil {
		log.Fatal(err)
	}

	_, err = np.SapiZbWriteConfiguration(0x87, []uint8{0}) //logical type

	if err != nil {
		log.Fatal(err)
	}

	_, err = np.UtilSetPanId(coordinator.config.PanId)

	if err != nil {
		log.Fatal(err)
	}

	//zdo direc cb
	_, err = np.SapiZbWriteConfiguration(0x8F, []uint8{1})

	if err != nil {
		log.Fatal(err)
	}

	//enable security
	_, err = np.SapiZbWriteConfiguration(0x64, []uint8{1})

	if err != nil {
		log.Fatal(err)
	}

	_, err = np.SysSetExtAddr(coordinator.config.IEEEAddress)

	if err != nil {
		log.Fatal(err)
	}

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

	_, err = coordinator.networkProcessor.UtilSetChannels(channels)

	if err != nil {
		log.Fatal(err)
	}
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

func openPort(config *configuration.Configuration) (port serial.Port, err error) {
	logger.Debugf("Opening port [%s] at rate [%d]", config.Serial.PortName, config.Serial.BaudRate)
	mode := &serial.Mode{BaudRate: config.Serial.BaudRate}
	port, err = serial.Open(config.Serial.PortName, mode)
	if err == nil {
		logger.Debugf("Port [%s] is opened", config.Serial.PortName)
		err = port.SetRTS(true)
	}
	return
}
