/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deliverclient

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/deliverservice/blocksprovider"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/spf13/viper"
)

var bftLogger = flogging.MustGetLogger("bftDeliveryClient")

var (
	bftBaseBackoffDuration = 10 * time.Millisecond
	bftMaxBackoffDuration  = 10 * time.Second

	errNoBlockReceiver = errors.New("no block receiver")
	errDuplicateBlock  = errors.New("duplicate block")
	errOutOfOrderBlock = errors.New("out-of-order block")
)

//go:generate mockery -dir . -name MessageCryptoVerifier -case underscore -output ./mocks/

// MessageCryptoVerifier defines message verification methods, and is subset of gossip/api/MessageCryptoService.
type MessageCryptoVerifier interface {
	// VerifyHeader returns nil when the header matches the metadata signature, but it does not compute the
	// block.Data.Hash() and compare it to the block.Header.DataHash, or otherwise inspect the block.Data.
	// This is used when the orderer delivers a block with header & metadata only (i.e. block.Data==nil).
	// See: gossip/api/MessageCryptoService
	VerifyHeader(chainID string, signedBlock *common.Block) error
}

type bftDeliveryClient struct {
	mutex     sync.Mutex
	startOnce sync.Once
	stopFlag  bool
	stopChan  chan struct{}

	chainID            string                    // a.k.a. Channel ID
	connectionFactory  comm.ConnectionFactory    // creates a gRPC connection
	createClient       clientFactory             // creates orderer.AtomicBroadcastClient
	ledgerInfoProvider blocksprovider.LedgerInfo // provides access to the ledger height
	msgCryptoVerifier  MessageCryptoVerifier     // verifies headers

	reconnectBackoffThreshold   float64
	reconnectTotalTimeThreshold time.Duration

	endpoints          []comm.EndpointCriteria // a set of endpoints
	blockReceiverIndex int                     // index of the current block receiver endpoint

	blockReceiver   *broadcastClient
	nextBlockNumber uint64
	lastBlockTime   time.Time

	headerReceivers map[string]*bftHeaderReceiver
}

func NewBFTDeliveryClient(
	chainID string,
	connFactory comm.ConnectionFactory,
	endpoints []comm.EndpointCriteria,
	clFactory clientFactory,
	ledgerInfoProvider blocksprovider.LedgerInfo,
	msgVerifier MessageCryptoVerifier,
) *bftDeliveryClient {
	c := &bftDeliveryClient{
		stopChan:                    make(chan struct{}, 1),
		chainID:                     chainID,
		connectionFactory:           connFactory,
		createClient:                clFactory,
		ledgerInfoProvider:          ledgerInfoProvider,
		msgCryptoVerifier:           msgVerifier,
		reconnectBackoffThreshold:   getReConnectBackoffThreshold(),
		reconnectTotalTimeThreshold: getReConnectTotalTimeThreshold(),
		endpoints:                   comm.Shuffle(endpoints),
		blockReceiverIndex:          -1,
		headerReceivers:             make(map[string]*bftHeaderReceiver),
	}

	bftLogger.Infof("[%s] Created BFT Delivery Client", chainID)
	return c
}

func (c *bftDeliveryClient) Recv() (response *orderer.DeliverResponse, err error) {
	bftLogger.Debugf("[%s] Entry", c.chainID)

	c.startOnce.Do(func() {
		var num uint64
		num, err = c.ledgerInfoProvider.LedgerHeight()
		if err != nil {
			return
		}
		c.nextBlockNumber = num
		bftLogger.Debugf("[%s] Starting monitor routine; Initial ledger height: %d", c.chainID, num)
		go c.monitor()
	})
	// can only happen once, after first invocation
	if err != nil {
		return nil, errors.New("cannot access ledger height")
	}

	var numEP int
	var numRetries int

	for !c.shouldStop() {
		if numEP, err = c.assignReceivers(); err != nil {
			return nil, err
		}

		if err = c.launchHeaderReceivers(); err != nil {
			return nil, err
		}

		response, err = c.receiveBlock()
		if err == nil {
			bftLogger.Debugf("[%s] Exit", c.chainID)
			return response, nil // the normal return path
		}

		c.closeBlockReceiver()
		numRetries++
		if numRetries%numEP == 0 {
			backoff(2.0, uint(numRetries/numEP), c.stopChan)
		}
	}

	bftLogger.Debugf("[%s] Exit: %v %v", c.chainID, response, err)
	return response, err
}

func backoff(base float64, exponent uint, stopChan <-chan struct{}) {
	var backoffDur time.Duration

	if base < 1.0 {
		base = 1.0
	}
	fDur := math.Pow(base, float64(exponent)) * float64(bftBaseBackoffDuration.Nanoseconds())
	if math.IsInf(fDur, 0) || math.IsNaN(fDur) || fDur > float64(bftMaxBackoffDuration) {
		backoffDur = bftMaxBackoffDuration
	} else {
		backoffDur = time.Duration(int64(fDur))
	}

	backoffTimer := time.After(backoffDur)
	select {
	case <-backoffTimer:
	case <-stopChan:
	}
}

// Check block reception progress relative to header reception progress.
// If the orderer associated with the block receiver is suspected of censorship, replace it with another orderer.
func (c *bftDeliveryClient) monitor() {
	bftLogger.Debugf("[%s] Entry", c.chainID)
	for !c.shouldStop() {
		select {
		case <-time.After(100 * time.Millisecond):
		case <-c.stopChan:
		}

		//TODO
	}
	bftLogger.Debugf("[%s] Exit", c.chainID)
}

// (re)-assign a block delivery client and header delivery clients
func (c *bftDeliveryClient) assignReceivers() (int, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	numEP := len(c.endpoints)
	if numEP <= 0 {
		return numEP, errors.New("no endpoints")
	}

	if c.blockReceiver == nil {
		c.blockReceiverIndex = (c.blockReceiverIndex + 1) % numEP
		ep := c.endpoints[c.blockReceiverIndex]
		if headerReceiver, exists := c.headerReceivers[ep.Endpoint]; exists {
			headerReceiver.Close()
			delete(c.headerReceivers, ep.Endpoint)
		}
		c.blockReceiver = c.newBlockClient(ep)
		bftLogger.Debugf("[%s] Created block receiver to: %s", c.chainID, ep.Endpoint)
	}

	hRcvToCreate := make([]comm.EndpointCriteria, 0)
	for i, ep := range c.endpoints {
		if i == c.blockReceiverIndex {
			continue
		}

		if hRcv, exists := c.headerReceivers[ep.Endpoint]; exists {
			if hRcv.isStopped() {
				delete(c.headerReceivers, ep.Endpoint)
			} else {
				continue
			}
		}

		hRcvToCreate = append(hRcvToCreate, ep)
	}

	for _, ep := range hRcvToCreate {
		headerReceiver := newBFTHeaderReceiver(c.chainID, ep.Endpoint, c.newHeaderClient(ep), c.msgCryptoVerifier)
		c.headerReceivers[ep.Endpoint] = headerReceiver
		bftLogger.Debugf("[%s] Created header receiver to: %s", c.chainID, ep.Endpoint)
	}

	bftLogger.Debugf("Exit: number of endpoints: %d", numEP)
	return numEP, nil
}

func (c *bftDeliveryClient) launchHeaderReceivers() error { //TODO do we need the error?
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for ep, hRcv := range c.headerReceivers {
		if !hRcv.isStarted() {
			bftLogger.Debugf("[%s] launching a header receiver to endpoint: %s", c.chainID, ep)
			go hRcv.DeliverHeaders()
		}
	}

	bftLogger.Debugf("[%s] launched %d header receivers", c.chainID, len(c.headerReceivers))

	return nil
}

func (c *bftDeliveryClient) receiveBlock() (*orderer.DeliverResponse, error) {
	c.mutex.Lock()
	receiver := c.blockReceiver
	nextBlockNumber := c.nextBlockNumber
	c.mutex.Unlock()

	// call Recv() without a lock
	if receiver == nil {
		return nil, errNoBlockReceiver
	}
	response, err := receiver.Recv()

	if err != nil {
		return response, err
	}
	// ignore older blocks, filter out-of-order blocks
	switch t := response.Type.(type) {
	case *orderer.DeliverResponse_Block:
		if t.Block.Header.Number > nextBlockNumber {
			bftLogger.Warnf("[%s] Ignoring out-of-order block from orderer: %s; received block number: %d, expected: %d",
				c.chainID, receiver.GetEndpoint(), t.Block.Header.Number, nextBlockNumber)
			return nil, errOutOfOrderBlock
		}
		if t.Block.Header.Number < nextBlockNumber {
			bftLogger.Warnf("[%s] Ignoring duplicate block from orderer: %s; received block number: %d, expected: %d",
				c.chainID, receiver.GetEndpoint(), t.Block.Header.Number, nextBlockNumber)
			return nil, errDuplicateBlock
		}
	}

	return response, err
}

func (c *bftDeliveryClient) closeBlockReceiver() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.blockReceiver != nil {
		c.blockReceiver.Close()
		c.blockReceiver = nil
	}
}

// create a broadcastClient that delivers blocks
func (c *bftDeliveryClient) newBlockClient(endpoint comm.EndpointCriteria) *broadcastClient {
	requester := &blocksRequester{
		tls:     viper.GetBool("peer.tls.enabled"),
		chainID: c.chainID,
	}
	broadcastSetup := func(bd blocksprovider.BlocksDeliverer) error {
		return requester.RequestBlocks(c.ledgerInfoProvider)
	}
	backoffPolicy := func(attemptNum int, elapsedTime time.Duration) (time.Duration, bool) {
		if elapsedTime >= c.reconnectTotalTimeThreshold {
			return 0, false
		}
		sleepIncrement := float64(time.Millisecond * 500)
		attempt := float64(attemptNum)
		return time.Duration(math.Min(math.Pow(2, attempt)*sleepIncrement, c.reconnectBackoffThreshold)), true
	}
	//Only a single endpoint
	connProd := comm.NewConnectionProducer(c.connectionFactory, []comm.EndpointCriteria{endpoint})
	blockClient := NewBroadcastClient(connProd, c.createClient, broadcastSetup, backoffPolicy)
	requester.client = blockClient
	return blockClient
}

// create a broadcastClient that delivers headers
func (c *bftDeliveryClient) newHeaderClient(endpoint comm.EndpointCriteria) *broadcastClient {
	requester := &blocksRequester{
		tls:     viper.GetBool("peer.tls.enabled"),
		chainID: c.chainID,
	}
	broadcastSetup := func(bd blocksprovider.BlocksDeliverer) error {
		return requester.RequestHeaders(c.ledgerInfoProvider)
	}
	backoffPolicy := func(attemptNum int, elapsedTime time.Duration) (time.Duration, bool) {
		if elapsedTime >= c.reconnectTotalTimeThreshold {
			return 0, false
		}
		sleepIncrement := float64(time.Millisecond * 500)
		attempt := float64(attemptNum)
		return time.Duration(math.Min(math.Pow(2, attempt)*sleepIncrement, c.reconnectBackoffThreshold)), true
	}
	//Only a single endpoint
	connProd := comm.NewConnectionProducer(c.connectionFactory, []comm.EndpointCriteria{endpoint})
	headerClient := NewBroadcastClient(connProd, c.createClient, broadcastSetup, backoffPolicy)
	requester.client = headerClient
	return headerClient
}

func (c *bftDeliveryClient) Send(*common.Envelope) error {
	return errors.New("should never be called")
}

func (c *bftDeliveryClient) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.stopFlag {
		return
	}

	c.stopFlag = true
	close(c.stopChan)

	if c.blockReceiver != nil {
		ep := c.blockReceiver.GetEndpoint()
		c.blockReceiver.Close()
		bftLogger.Debugf("[%s] closed block receiver to: %s", c.chainID, ep)
		c.blockReceiver = nil
	}

	for ep, hRcv := range c.headerReceivers {
		hRcv.Close()
		bftLogger.Debugf("[%s] closed header receiver to: %s", c.chainID, ep)
		delete(c.headerReceivers, ep)
	}

	bftLogger.Debugf("Exit")
}

// Disconnect just the block receiver client, so that the next Recv() will choose a new one.
func (c *bftDeliveryClient) Disconnect() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.blockReceiver != nil {
		c.blockReceiver.Disconnect()
		c.blockReceiver = nil
	}
}

// UpdateReceived allows the client to track the reception of valid blocks.
// This is needed because blocks are verified by the blockprovider, not here.
func (c *bftDeliveryClient) UpdateReceived(blockNumber uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.nextBlockNumber = blockNumber + 1
	c.lastBlockTime = time.Now()
}

func (c *bftDeliveryClient) UpdateEndpoints(endpoints []comm.EndpointCriteria) {
	//TODO
}

func (c *bftDeliveryClient) shouldStop() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.stopFlag
}
