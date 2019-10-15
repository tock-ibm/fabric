/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deliverclient

import (
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/deliverservice/mocks"

	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestNewBFTDeliverClient(t *testing.T) {
	connFactory := func(endpoint comm.EndpointCriteria) (*grpc.ClientConn, error) {
		return &grpc.ClientConn{}, nil
	}

	endpoints := []comm.EndpointCriteria{
		{Endpoint: "localhost:5611", Organizations: []string{"org1"}},
		{Endpoint: "localhost:5612", Organizations: []string{"org2"}},
		{Endpoint: "localhost:5613", Organizations: []string{"org3"}},
		{Endpoint: "localhost:5614", Organizations: []string{"org4"}},
	}

	abcClient := &abclient{}
	clFactory := func(*grpc.ClientConn) orderer.AtomicBroadcastClient {
		return abcClient
	}

	ledgerInfoMock := &mocks.LedgerInfo{}
	backoffStrategy := func(attemptNum int, elapsedTime time.Duration) (time.Duration, bool) {
		return time.Duration(0), attemptNum < 1
	}
	bc := NewBFTDeliveryClient("test-chain", connFactory, endpoints, clFactory, ledgerInfoMock, backoffStrategy)
	defer bc.Close()

	_, err := bc.Recv()
	assert.NoError(t, err)
}
