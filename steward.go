package steward

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/dyrkin/zcl-go"
	"github.com/dyrkin/zcl-go/cluster"
	"github.com/dyrkin/zigbee-steward/configuration"
	"github.com/dyrkin/zigbee-steward/coordinator"
	"github.com/dyrkin/zigbee-steward/db"
	"github.com/dyrkin/zigbee-steward/logger"
	"github.com/dyrkin/zigbee-steward/model"
	"github.com/dyrkin/znp-go"
)

var log = logger.MustGetLogger("steward")

type Steward struct {
	configuration     *configuration.Configuration
	coordinator       *coordinator.Coordinator
	registrationQueue chan *znp.ZdoEndDeviceAnnceInd
	zcl               *zcl.Zcl
	channels          *Channels
	functions         *Functions
}

func New(configuration *configuration.Configuration) *Steward {
	coordinator := coordinator.New(configuration)
	steward := &Steward{
		configuration:     configuration,
		coordinator:       coordinator,
		registrationQueue: make(chan *znp.ZdoEndDeviceAnnceInd),
		zcl:               zcl.New(),
		channels: &Channels{
			onDeviceRegistered:      make(chan *model.Device, 10),
			onDeviceBecameAvailable: make(chan *model.Device, 10),
			onDeviceUnregistered:    make(chan *model.Device, 10),
			onDeviceIncomingMessage: make(chan *model.DeviceIncomingMessage, 100),
		},
	}
	steward.functions = NewFunctions(coordinator)
	return steward
}

func (s *Steward) Start() {
	go s.enableRegistrationQueue()
	go s.enableListeners()
	err := s.coordinator.Start()
	if err != nil {
		panic(err)
	}
}

func (s *Steward) Channels() *Channels {
	return s.channels
}

func (s *Steward) Functions() *Functions {
	return s.functions
}

func (s *Steward) Network() *coordinator.Network {
	return s.coordinator.Network()
}

func (s *Steward) Configuration() *configuration.Configuration {
	return s.configuration
}

func (s *Steward) enableListeners() {
	for {
		select {
		case err := <-s.coordinator.OnError():
			log.Errorf("Received error: %s", err)
		case announcedDevice := <-s.coordinator.OnDeviceAnnounce():
			s.registrationQueue <- announcedDevice
		case deviceLeave := <-s.coordinator.OnDeviceLeave():
			s.unregisterDevice(deviceLeave)
		case _ = <-s.coordinator.OnDeviceTc():
		case incomingMessage := <-s.coordinator.OnIncomingMessage():
			s.processIncomingMessage(incomingMessage)
		}
	}
}

func (s *Steward) enableRegistrationQueue() {
	for announcedDevice := range s.registrationQueue {
		s.registerDevice(announcedDevice)
	}
}

func (s *Steward) registerDevice(announcedDevice *znp.ZdoEndDeviceAnnceInd) {
	ieeeAddress := announcedDevice.IEEEAddr
	log.Infof("Registering device [%s]", ieeeAddress)
	if device, ok := db.Database().Tables().Devices.Get(ieeeAddress); ok {
		log.Debugf("Device [%s] already exists in DB. Updating network address", ieeeAddress)
		device.NetworkAddress = announcedDevice.NwkAddr
		db.Database().Tables().Devices.Add(device)
		select {
		case s.channels.onDeviceBecameAvailable <- device:
		default:
			log.Errorf("onDeviceBecameAvailable channel has no capacity. Maybe channel has no subscribers")
		}
		return
	}

	device := &model.Device{Endpoints: []*model.Endpoint{}}
	device.IEEEAddress = ieeeAddress
	nwkAddress := announcedDevice.NwkAddr
	device.NetworkAddress = nwkAddress
	if announcedDevice.Capabilities.MainPowered > 0 {
		device.MainPowered = true
	}

	deviceDetails, err := s.Functions().Cluster().Global().ReadAttributes(nwkAddress, []uint16{0x0004, 0x0005, 0x0007})
	if err != nil {
		log.Errorf("Unable to register device: %s", err)
		return
	}
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
	nodeDescription, err := s.coordinator.NodeDescription(nwkAddress)
	if err != nil {
		log.Errorf("Unable to register device: %s", err)
		return
	}

	device.LogicalType = nodeDescription.LogicalType
	device.ManufacturerId = nodeDescription.ManufacturerCode

	log.Debugf("Request active endpoints: [%s]", ieeeAddress)
	activeEndpoints, err := s.coordinator.ActiveEndpoints(nwkAddress)
	if err != nil {
		log.Errorf("Unable to register device: %s", err)
		return
	}

	for _, ep := range activeEndpoints.ActiveEPList {
		log.Debugf("Request endpoint description: [%s], ep: [%d]", ieeeAddress, ep)
		simpleDescription, err := s.coordinator.SimpleDescription(nwkAddress, ep)
		if err != nil {
			log.Errorf("Unable to receive endpoint data: %d. Reason: %s", ep, err)
			continue
		}
		endpoint := s.createEndpoint(simpleDescription)
		device.Endpoints = append(device.Endpoints, endpoint)
	}

	db.Database().Tables().Devices.Add(device)
	select {
	case s.channels.onDeviceRegistered <- device:
	default:
		log.Errorf("onDeviceRegistered channel has no capacity. Maybe channel has no subscribers")
	}

	log.Infof("Registered new device [%s]. Manufacturer: [%s], Model: [%s], Logical type: [%s]",
		ieeeAddress, device.Manufacturer, device.Model, device.LogicalType)
	log.Debugf("Registered new device:\n%s", func() string { return spew.Sdump(device) })
}

func (s *Steward) createEndpoint(simpleDescription *znp.ZdoSimpleDescRsp) *model.Endpoint {
	return &model.Endpoint{
		Id:             simpleDescription.Endpoint,
		ProfileId:      simpleDescription.ProfileID,
		DeviceId:       simpleDescription.DeviceID,
		DeviceVersion:  simpleDescription.DeviceVersion,
		InClusterList:  s.createClusters(simpleDescription.InClusterList),
		OutClusterList: s.createClusters(simpleDescription.OutClusterList),
	}
}

func (s *Steward) createClusters(clusterIds []uint16) []*model.Cluster {
	var clusters []*model.Cluster
	for _, clusterId := range clusterIds {
		cl := &model.Cluster{Id: clusterId}
		if c, ok := s.zcl.ClusterLibrary().Clusters()[cluster.ClusterId(clusterId)]; ok {
			cl.Supported = true
			cl.Name = c.Name
		}
		clusters = append(clusters, cl)
	}
	return clusters
}

func (s *Steward) processIncomingMessage(incomingMessage *znp.AfIncomingMessage) {
	zclIncomingMessage, err := s.zcl.ToZclIncomingMessage(incomingMessage)
	if err == nil {
		log.Debugf("Foundation Frame Payload\n%s\n", func() string { return spew.Sdump(zclIncomingMessage) })
		if device, ok := db.Database().Tables().Devices.GetByNetworkAddress(incomingMessage.SrcAddr); ok {
			deviceIncomingMessage := &model.DeviceIncomingMessage{
				Device:          device,
				IncomingMessage: zclIncomingMessage,
			}
			select {
			case s.channels.onDeviceIncomingMessage <- deviceIncomingMessage:
			default:
				log.Errorf("onDeviceIncomingMessage channel has no capacity. Maybe channel has no subscribers")
			}
		} else {
			log.Errorf("Received message from unknown device [%s]", incomingMessage.SrcAddr)
		}
	} else {
		log.Errorf("Unsupported incoming message:\n%s\n", func() string { return spew.Sdump(incomingMessage) })
	}
}

func (s *Steward) unregisterDevice(deviceLeave *znp.ZdoLeaveInd) {
	ieeeAddress := deviceLeave.ExtAddr
	if device, ok := db.Database().Tables().Devices.Get(ieeeAddress); ok {
		log.Infof("Unregistering device: [%s]", ieeeAddress)
		db.Database().Tables().Devices.Remove(ieeeAddress)
		select {
		case s.channels.onDeviceUnregistered <- device:
		default:
			log.Errorf("onDeviceUnregistered channel has no capacity. Maybe channel has no subscribers")
		}

		log.Infof("Unregistered device [%s]. Manufacturer: [%s], Model: [%s], Logical type: [%s]",
			ieeeAddress, device.Manufacturer, device.Model, device.LogicalType)
	}
}
