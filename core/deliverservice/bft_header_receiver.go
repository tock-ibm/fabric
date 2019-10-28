/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deliverclient

import (
	"math"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
)

const bftHeaderMaxRetryDelay = time.Second * 10
const bftHeaderWrongStatusThreshold = 10

type bftHeaderReceiver struct {
	mutex             sync.Mutex
	chainID           string
	stop              bool
	stopChan          chan interface{}
	started           bool
	endpoint          string
	client            *broadcastClient
	msgCryptoVerifier MessageCryptoVerifier
	lastHeader        *common.Block // a block with Header & Metadata, without Data (i.e. lastHeader.Data==nil)
	lastHeaderTime    time.Time
}

func newBFTHeaderReceiver(
	chainID string,
	endpoint string,
	client *broadcastClient,
	msgVerifier MessageCryptoVerifier,
) *bftHeaderReceiver {
	hRcv := &bftHeaderReceiver{
		chainID:           chainID,
		stopChan:          make(chan interface{}, 1),
		endpoint:          endpoint,
		client:            client,
		msgCryptoVerifier: msgVerifier,
	}
	return hRcv
}

func (hr *bftHeaderReceiver) DeliverHeaders() {
	defer func() {
		hr.Close()
	}()

	bftLogger.Debugf("[%s] Starting to deliver headers from endpoint: %s", hr.chainID, hr.endpoint)
	hr.setStarted()
	errorStatusCounter := 0
	statusCounter := 0

	for !hr.isStopped() {
		msg, err := hr.client.Recv()
		if err != nil {
			bftLogger.Warningf("[%s] Receive error: %s", hr.chainID, err.Error())
			return
		}

		switch t := msg.Type.(type) {
		case *orderer.DeliverResponse_Status:
			if t.Status == common.Status_SUCCESS {
				bftLogger.Warningf("[%s] ERROR! Received success for a seek that should never complete", hr.chainID)
				return
			}
			if t.Status == common.Status_BAD_REQUEST || t.Status == common.Status_FORBIDDEN {
				bftLogger.Errorf("[%s] Got error %v", hr.chainID, t)
				errorStatusCounter++
				if errorStatusCounter > bftHeaderWrongStatusThreshold {
					bftLogger.Criticalf("[%s] Wrong statuses threshold passed, stopping bft header receiver", hr.chainID)
					return
				}
			} else {
				errorStatusCounter = 0
				bftLogger.Warningf("[%s] Got error %v", hr.chainID, t)
			}
			maxDelay := float64(bftHeaderMaxRetryDelay)
			currDelay := float64(time.Duration(math.Pow(2, float64(statusCounter))) * 100 * time.Millisecond)
			time.Sleep(time.Duration(math.Min(maxDelay, currDelay)))
			if currDelay < maxDelay {
				statusCounter++
			}
			hr.client.Disconnect()
			continue

		case *orderer.DeliverResponse_Block:
			errorStatusCounter = 0
			statusCounter = 0
			blockNum := t.Block.Header.Number

			// do not verify, just save for later, in case the block-receiver is suspected of censorship
			bftLogger.Debugf("[%s] Saving block with header & metadata, blockNum = [%d]", hr.chainID, blockNum)
			hr.mutex.Lock()
			hr.lastHeader = t.Block
			hr.lastHeaderTime = time.Now()
			hr.mutex.Unlock()

		default:
			bftLogger.Warningf("[%s] Received unknown: %v", hr.chainID, t)
			return
		}
	}

	bftLogger.Debugf("[%s] Stopped to deliver headers from endpoint: %s", hr.chainID, hr.client.GetEndpoint())
}

func (hr *bftHeaderReceiver) isStopped() bool {
	hr.mutex.Lock()
	defer hr.mutex.Unlock()
	return hr.stop
}

func (hr *bftHeaderReceiver) isStarted() bool {
	hr.mutex.Lock()
	defer hr.mutex.Unlock()
	return hr.started
}

func (hr *bftHeaderReceiver) setStarted() {
	hr.mutex.Lock()
	defer hr.mutex.Unlock()
	hr.started = true
}

func (hr *bftHeaderReceiver) Close() {
	hr.mutex.Lock()
	defer hr.mutex.Unlock()

	if hr.stop {
		return
	}

	hr.stop = true
	hr.client.Close()
}

func (hr *bftHeaderReceiver) LastBlockNum() (uint64, time.Time, error) {
	hr.mutex.Lock()
	defer hr.mutex.Unlock()

	if hr.lastHeader == nil {
		return 0, time.Unix(0, 0), errors.New("Not found")
	}

	err := hr.msgCryptoVerifier.VerifyHeader(hr.chainID, hr.lastHeader)
	if err != nil {
		bftLogger.Warningf("[%s] Last block verification failed: %s", hr.chainID, err)
		return 0, time.Unix(0, 0), errors.Wrapf(err, "Last block verification failed")
	}

	return hr.lastHeader.Header.Number, hr.lastHeaderTime, nil
}
