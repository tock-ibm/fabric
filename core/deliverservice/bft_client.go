/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deliverclient

import (
	"sync"

	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/deliverservice/blocksprovider"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
)

// streamHeaderSetup is a function that is called immediately after each  successful connection to the ordering service
// in order to request the reception of a stream of headers (rather than blocks)
type streamHeaderSetup func(blocksprovider.BlocksDeliverer) error

type bftDeliveryClient struct {
	mutex    sync.Mutex
	stopFlag int32
	stopChan chan struct{}

	chainID             string
	connectionFactory   comm.ConnectionFactory
	createClient        clientFactory // creates orderer.AtomicBroadcastClient
	ledgerInfoProvider  blocksprovider.LedgerInfo
	onConnectionBlocks  broadcastSetup    // requests a stream of incoming blocks
	onConnectionHeaders streamHeaderSetup // requests a stream of incoming headers
	retryFun            retryPolicy

	endpoints []comm.EndpointCriteria
}

func NewBFTDeliveryClient(
	chainID string,
	connFactory comm.ConnectionFactory,
	endpoints []comm.EndpointCriteria,
	clFactory clientFactory,
	ledgerInfoProvider blocksprovider.LedgerInfo,
	retry retryPolicy,
) *bftDeliveryClient {
	c := &bftDeliveryClient{
		stopChan:           make(chan struct{}, 1),
		chainID:            chainID,
		connectionFactory:  connFactory,
		createClient:       clFactory,
		ledgerInfoProvider: ledgerInfoProvider,
		endpoints:          endpoints,
		retryFun:           retry,
	}
	//TODO

	logger.Infof("Created BFT Delivery Client for chain: %s", chainID)
	return c
}

func (c *bftDeliveryClient) Recv() (*orderer.DeliverResponse, error) {
	//TODO
	return &orderer.DeliverResponse{}, nil
}

func (c *bftDeliveryClient) Send(*common.Envelope) error {
	//TODO
	return nil
}

func (c *bftDeliveryClient) Close() {
	//TODO
}

func (c *bftDeliveryClient) Disconnect() {
	//TODO
}

func (bc *bftDeliveryClient) UpdateEndpoints(endpoints []comm.EndpointCriteria) {
	//TODO
}
