package steward

import (
	"fmt"
	"github.com/dyrkin/zcl-go"
	"github.com/dyrkin/zcl-go/cluster"
	"github.com/dyrkin/zcl-go/frame"
	"github.com/dyrkin/znp-go"
	"time"
)

type Functions struct {
	steward *Steward
}

var nextTransactionId = frame.MakeDefaultTransactionIdProvider()

func (f *Functions) ReadAttributes(nwkAddress string, attributeIds []uint16) (*cluster.ReadAttributesResponse, error) {
	transactionId := nextTransactionId()

	attributesRequest := func(nwkAddress string, transactionId uint8) error {
		return f.steward.coordinator.ReadAttributes(nwkAddress, transactionId, []uint16{0x0004, 0x0005, 0x0007})
	}

	deviceDetailsResponse, err := f.syncDataRequestRetryable(nwkAddress, transactionId, attributesRequest, 10*time.Second, 3)
	if err != nil {
		log.Errorf("Unable to retrieve attributes: %s", err)
		return nil, err
	}
	return deviceDetailsResponse.Data.Command.(*cluster.ReadAttributesResponse), nil
}

func (f *Functions) Bind(sourceAddress string, sourceIeeeAddress string, sourceEndpoint uint8, clusterId uint16, destinationIeeeAddress string, destinationEndpoint uint8) (*znp.ZdoBindRsp, error) {
	return f.steward.coordinator.Bind(sourceAddress, sourceIeeeAddress, sourceEndpoint, clusterId, znp.AddrModeAddr64Bit, destinationIeeeAddress, destinationEndpoint)
}

func (f *Functions) Unbind(sourceAddress string, sourceIeeeAddress string, sourceEndpoint uint8, clusterId uint16, destinationIeeeAddress string, destinationEndpoint uint8) (*znp.ZdoUnbindRsp, error) {
	return f.steward.coordinator.Unbind(sourceAddress, sourceIeeeAddress, sourceEndpoint, clusterId, znp.AddrModeAddr64Bit, destinationIeeeAddress, destinationEndpoint)
}

func (f *Functions) syncDataRequestRetryable(nwkAddress string, transactionId uint8, request func(string, uint8) error, timeout time.Duration, retries int) (*zcl.ZclIncomingMessage, error) {
	zclIncomingMessage, err := f.syncDataRequest(nwkAddress, transactionId, request, timeout)
	switch {
	case err != nil && retries > 0:
		log.Errorf("%s. Retries: %d", err, retries)
		return f.syncDataRequestRetryable(nwkAddress, transactionId, request, timeout, retries-1)
	case err != nil && retries == 0:
		log.Errorf("failure: %s", err)
		return nil, err
	}
	return zclIncomingMessage, nil
}

func (f *Functions) syncDataRequest(nwkAddress string, transactionId uint8, request func(string, uint8) error, timeout time.Duration) (*zcl.ZclIncomingMessage, error) {
	dataConfirmReceiver := make(chan interface{})
	incomingMessageReceiver := make(chan interface{})

	responseChannel := make(chan *zcl.ZclIncomingMessage, 1)
	errorChannel := make(chan error, 1)

	incomingMessageListener := func() {
		deadline := time.NewTimer(timeout)
		f.steward.incomingMessageTopic.Register(incomingMessageReceiver)
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
		f.steward.dataConfirmTopic.Register(dataConfirmReceiver)
		for {
			select {
			case response := <-dataConfirmReceiver:
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
		f.steward.dataConfirmTopic.Unregister(dataConfirmReceiver)
		f.steward.incomingMessageTopic.Unregister(incomingMessageReceiver)
		return nil, fmt.Errorf("unable to send data request: %s", err)
	}

	select {
	case err = <-errorChannel:
		f.steward.dataConfirmTopic.Unregister(dataConfirmReceiver)
		f.steward.incomingMessageTopic.Unregister(incomingMessageReceiver)
		return nil, err
	case zclIncomingMessage := <-responseChannel:
		f.steward.dataConfirmTopic.Unregister(dataConfirmReceiver)
		f.steward.incomingMessageTopic.Unregister(incomingMessageReceiver)
		return zclIncomingMessage, nil
	}
}
