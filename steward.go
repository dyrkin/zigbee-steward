package steward

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/dyrkin/zcl-go"
	"github.com/dyrkin/zcl-go/cluster"
	"github.com/dyrkin/zcl-go/frame"
	"github.com/dyrkin/zigbee-steward/configuration"
	"github.com/dyrkin/zigbee-steward/coordinator"
	"github.com/dyrkin/zigbee-steward/db"
	"github.com/dyrkin/zigbee-steward/logger"
	"github.com/dyrkin/zigbee-steward/model"
	"github.com/dyrkin/znp-go"
	"github.com/tv42/topic"
	"time"
)

var log = logger.MustGetLogger("steward")
var nextTransactionId = frame.MakeDefaultTransactionIdProvider()

type Steward struct {
	coordinator          *coordinator.Coordinator
	registrationQueue    chan *znp.ZdoEndDeviceAnnceInd
	zcl                  *zcl.Zcl
	incomingMessageTopic *topic.Topic
	dataConfirmTopic     *topic.Topic
}

func New() *Steward {
	return &Steward{
		registrationQueue:    make(chan *znp.ZdoEndDeviceAnnceInd),
		zcl:                  zcl.New(),
		incomingMessageTopic: topic.New(),
		dataConfirmTopic:     topic.New(),
	}
}

func (s *Steward) Start(configPath string) {
	conf, err := configuration.Read(configPath)
	if err != nil {
		panic(err)
	}

	s.coordinator = coordinator.New(conf)
	go s.enableRegistrationQueue()
	go s.enableListeners()
	err = s.coordinator.Start()
	if err != nil {
		panic(err)
	}
}

func (s *Steward) enableListeners() {
	for {
		select {
		case err := <-s.coordinator.OnError():
			log.Errorf("Received error: %s", err)
		case announcedDevice := <-s.coordinator.OnDeviceAnnounce():
			s.registerDevice(announcedDevice)
		case _ = <-s.coordinator.OnDeviceLeave():
		case _ = <-s.coordinator.OnDeviceTc():
		case incomingMessage := <-s.coordinator.OnIncomingMessage():
			s.processIncomingMessage(incomingMessage)
		case dataConfirm := <-s.coordinator.OnDataConfirm():
			s.processDataConfirm(dataConfirm)
		}
	}
}

func (s *Steward) SyncDataRequestRetryable(nwkAddress string, transactionId uint8, request func(string, uint8) error, timeout time.Duration, retries int) (*zcl.ZclIncomingMessage, error) {
	zclIncomingMessage, err := s.SyncDataRequest(nwkAddress, transactionId, request, timeout)
	switch {
	case err != nil && retries > 0:
		log.Errorf("%s. Retries: %d", err, retries)
		return s.SyncDataRequestRetryable(nwkAddress, transactionId, request, timeout, retries-1)
	case err != nil && retries == 0:
		log.Errorf("failure: %s", err)
		return nil, err
	}
	return zclIncomingMessage, nil
}

func (s *Steward) SyncDataRequest(nwkAddress string, transactionId uint8, request func(string, uint8) error, timeout time.Duration) (*zcl.ZclIncomingMessage, error) {
	dataConfirmReceiver := make(chan interface{})
	incomingMessageReceiver := make(chan interface{})
	s.dataConfirmTopic.Register(dataConfirmReceiver)

	responseChannel := make(chan *zcl.ZclIncomingMessage, 1)
	errorChannel := make(chan error, 1)

	incomingMessageListener := func() {
		deadline := time.NewTimer(timeout)
		s.incomingMessageTopic.Register(incomingMessageReceiver)
		for {
			select {
			case response := <-incomingMessageReceiver:
				incomingMessage, ok := response.(*zcl.ZclIncomingMessage)
				if (ok && incomingMessage.Data.TransactionSequenceNumber == transactionId) &&
					(incomingMessage.SrcAddr == nwkAddress) {
					deadline.Stop()
					responseChannel <- incomingMessage
					return
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
			select {
			case response := <-dataConfirmReceiver:
				dataConfirm := response.(*znp.AfDataConfirm)
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
		s.dataConfirmTopic.Unregister(dataConfirmReceiver)
		s.incomingMessageTopic.Unregister(incomingMessageReceiver)
		return nil, fmt.Errorf("unable to send data request: %s", err)
	}

	select {
	case err = <-errorChannel:
		s.dataConfirmTopic.Unregister(dataConfirmReceiver)
		s.incomingMessageTopic.Unregister(incomingMessageReceiver)
		return nil, err
	case zclIncomingMessage := <-responseChannel:
		s.dataConfirmTopic.Unregister(dataConfirmReceiver)
		s.incomingMessageTopic.Unregister(incomingMessageReceiver)
		return zclIncomingMessage, nil
	}
}

func (s *Steward) enableRegistrationQueue() {
	for announcedDevice := range s.registrationQueue {
		ieeeAddress := announcedDevice.IEEEAddr
		log.Infof("Registering device: [%s]", ieeeAddress)
		device := &model.Device{Endpoints: []*model.Endpoint{}}
		device.IEEEAddress = ieeeAddress
		nwkAddress := announcedDevice.NwkAddr
		device.NetworkAddress = nwkAddress
		if announcedDevice.Capabilities.MainPowered > 0 {
			device.MainPowered = true
		}
		transactionId := nextTransactionId()

		attributesRequest := func(nwkAddress string, transactionId uint8) error {
			return s.coordinator.ReadAttributes(nwkAddress, transactionId, []uint16{0x0004, 0x0005, 0x0007})
		}

		log.Debugf("Request device attributes: [%s]", ieeeAddress)
		deviceDetailsResponse, err := s.SyncDataRequestRetryable(nwkAddress, transactionId, attributesRequest, 10*time.Second, 3)
		if err != nil {
			log.Errorf("Unable to register device: %s", err)
			continue
		}
		deviceDetails := deviceDetailsResponse.Data.Command.(*cluster.ReadAttributesResponse)
		if manufacturer, ok := deviceDetails.ReadAttributeStatuses[0].Attribute.Value.(string); ok {
			device.Manufacturer = manufacturer
		}
		if modelId, ok := deviceDetails.ReadAttributeStatuses[1].Attribute.Value.(string); ok {
			device.Model = modelId
		}
		if powerSource, ok := deviceDetails.ReadAttributeStatuses[2].Attribute.Value.(uint64); ok {
			device.PowerSource = model.PowerSource(powerSource)
		}

		log.Debugf("Request node description: [%s]", ieeeAddress)
		nodeDescription, err := s.coordinator.RequestNodeDescription(nwkAddress)
		if err != nil {
			log.Errorf("Unable to register device: %s", err)
			continue
		}

		device.LogicalType = nodeDescription.LogicalType
		device.ManufacturerId = nodeDescription.ManufacturerCode

		log.Debugf("Request active endpoints: [%s]", ieeeAddress)
		activeEndpoints, err := s.coordinator.RequestActiveEndpoints(nwkAddress)
		if err != nil {
			log.Errorf("Unable to register device: %s", err)
			continue
		}

		for _, ep := range activeEndpoints.ActiveEPList {
			log.Debugf("Request endpoint description: [%s], ep: [%d]", ieeeAddress, ep)
			simpleDescription, err := s.coordinator.RequestSimpleDescription(nwkAddress, ep)
			if err != nil {
				log.Errorf("Unable to receive endpoint data: %d. Reason: %s", ep, err)
				continue
			}
			endpoint := model.NewEndpoint(simpleDescription)
			device.Endpoints = append(device.Endpoints, endpoint)
		}

		db.Database.AddDevice(device)

		log.Infof("Registered device [%s]. Manufacturer: [%s], Model: [%s], Logical type: [%s]",
			ieeeAddress, device.Manufacturer, device.Model, device.LogicalType)
		log.Debugf("Registered device:\n%s", func() string { return spew.Sdump(device) })
	}
}

func (s *Steward) registerDevice(announcedDevice *znp.ZdoEndDeviceAnnceInd) {
	s.registrationQueue <- announcedDevice
}

func (s *Steward) processIncomingMessage(incomingMessage *znp.AfIncomingMessage) {
	zclIncomingMessage, err := s.zcl.ToZclIncomingMessage(incomingMessage)
	if err == nil {
		log.Debugf("Foundation Frame Payload\n%s\n", func() string { return spew.Sdump(zclIncomingMessage) })
		s.incomingMessageTopic.Broadcast <- zclIncomingMessage
	} else {
		log.Errorf("Unsupported incoming message:\n%s\n", func() string { return spew.Sdump(incomingMessage) })
	}
}

func (s *Steward) processDataConfirm(dataConfirm *znp.AfDataConfirm) {
	s.dataConfirmTopic.Broadcast <- dataConfirm
}
