/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deliverclient

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/deliverservice/mocks"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/util"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type MsgCryptoSrv interface {
	api.MessageCryptoService
}

var ports = []int{5611, 5612, 5613, 5614}
var endpoints = []comm.EndpointCriteria{
	{Endpoint: "localhost:5611", Organizations: []string{"org1"}},
	{Endpoint: "localhost:5612", Organizations: []string{"org2"}},
	{Endpoint: "localhost:5613", Organizations: []string{"org3"}},
	{Endpoint: "localhost:5614", Organizations: []string{"org4"}},
}

var endpointMap = map[int]comm.EndpointCriteria{
	5611: {Endpoint: "localhost:5611", Organizations: []string{"org1"}},
	5612: {Endpoint: "localhost:5612", Organizations: []string{"org2"}},
	5613: {Endpoint: "localhost:5613", Organizations: []string{"org3"}},
	5614: {Endpoint: "localhost:5614", Organizations: []string{"org4"}},
}

// Scenario: create a client and close it.
func TestBFTDeliverClient_New(t *testing.T) {
	flogging.ActivateSpec("bftDeliveryClient=DEBUG")

	connFactory := func(endpoint comm.EndpointCriteria) (*grpc.ClientConn, error) {
		return &grpc.ClientConn{}, nil
	}

	abcClient := &abclient{}
	clFactory := func(*grpc.ClientConn) orderer.AtomicBroadcastClient {
		return abcClient
	}

	ledgerInfoMock := &mocks.LedgerInfo{}
	bc := NewBFTDeliveryClient("test-chain", connFactory, endpoints, clFactory, ledgerInfoMock, &mockMCS{})
	defer bc.Close()
}

// Scenario: create a client against a set of orderer mocks. Receive several blocks and check block & header reception.
func TestBFTDeliverClient_Recv(t *testing.T) {
	flogging.ActivateSpec("bftDeliveryClient=DEBUG")

	defer ensureNoGoroutineLeak(t)()

	osArray := make([]*mocks.Orderer, 0)
	for port := range endpointMap {
		osArray = append(osArray, mocks.NewOrderer(port, t))
	}
	for _, os := range osArray {
		os.SetNextExpectedSeek(5)
	}

	connFactory := func(endpoint comm.EndpointCriteria) (*grpc.ClientConn, error) {
		ctx, cancel := context.WithTimeout(context.Background(), getConnectionTimeout())
		defer cancel()
		return grpc.DialContext(ctx, endpoint.Endpoint, grpc.WithInsecure(), grpc.WithBlock())
		//return grpc.Dial(endpoint.Endpoint, grpc.WithInsecure(), grpc.WithBlock())
	}
	ledgerInfoMock := &mocks.LedgerInfo{}
	msgVerifierMock := &mocks.MessageCryptoVerifier{}
	bc := NewBFTDeliveryClient("test-chain", connFactory, endpoints, DefaultABCFactory, ledgerInfoMock, msgVerifierMock)
	ledgerInfoMock.On("LedgerHeight").Return(uint64(5), nil)

	go func() {
		for {
			resp, err := bc.Recv()
			if err != nil {
				assert.EqualError(t, err, errClientClosing.Error())
				return
			}
			block := resp.GetBlock()
			assert.NotNil(t, block)
			if block == nil {
				return
			}
			bc.UpdateReceived(block.Header.Number)
		}
	}()

	// all orderers send something: block/header
	beforeSend := time.Now()
	for seq := uint64(5); seq < uint64(10); seq++ {
		for _, os := range osArray {
			os.SendBlock(seq)
		}
	}

	time.Sleep(time.Second)
	bc.Close()

	lastN, lastT := bc.GetNextBlockNumTime()
	assert.Equal(t, uint64(10), lastN)
	assert.True(t, lastT.After(beforeSend))

	headersNum, headerT := bc.GetHeadersBlockNumTime()
	for i, n := range headersNum {
		assert.Equal(t, uint64(10), n)
		assert.True(t, headerT[i].After(beforeSend))
	}

	for _, os := range osArray {
		os.Shutdown()
	}
}

// Scenario: block censorship by orderer. Create a client against a set of orderer mocks.
// Receive one block. Then, the orderer sending blocks stops sending but headers keep coming.
// The client should switch to another orderer and seek from the new height. Check block & header reception.
func TestBFTDeliverClient_Censorship(t *testing.T) {
	flogging.ActivateSpec("bftDeliveryClient,deliveryClient=DEBUG")
	viper.Set("peer.deliveryclient.bft.blockCensorshipTimeout", 2*time.Second)
	defer viper.Reset()
	defer ensureNoGoroutineLeak(t)()

	assert.Equal(t, util.GetDurationOrDefault("peer.deliveryclient.bft.blockCensorshipTimeout", bftBlockCensorshipTimeout), 2*time.Second)

	osMap := make(map[string]*mocks.Orderer, len(endpointMap))
	for port, ep := range endpointMap {
		osMap[ep.Endpoint] = mocks.NewOrderer(port, t)
	}
	for _, os := range osMap {
		os.SetNextExpectedSeek(5)
	}

	connFactory := func(endpoint comm.EndpointCriteria) (*grpc.ClientConn, error) {
		return grpc.Dial(endpoint.Endpoint, grpc.WithInsecure(), grpc.WithBlock())
	}
	var ledgerHeight uint64 = 5
	ledgerInfoMock := &mocks.LedgerInfo{}
	msgVerifierMock := &mocks.MessageCryptoVerifier{}
	bc := NewBFTDeliveryClient("test-chain", connFactory, endpoints, DefaultABCFactory, ledgerInfoMock, msgVerifierMock)
	ledgerInfoMock.On("LedgerHeight").Return(func() uint64 { return atomic.LoadUint64(&ledgerHeight) }, nil)
	msgVerifierMock.On("VerifyHeader", mock.AnythingOfType("string"), mock.AnythingOfType("*common.Block")).Return(nil)

	go func() {
		for {
			resp, err := bc.Recv()
			if err != nil {
				assert.EqualError(t, err, "client is closing")
				return
			}
			block := resp.GetBlock()
			assert.NotNil(t, block)
			if block == nil {
				return
			}
			atomic.StoreUint64(&ledgerHeight, block.Header.Number+1)
			bc.UpdateReceived(block.Header.Number)
		}
	}()

	blockEP, err := waitForBlockEP(bc)
	assert.NoError(t, err)
	// one normal block
	for _, os := range osMap {
		os.SendBlock(5)
	}
	for _, os := range osMap {
		os.SetNextExpectedSeek(6)
	}
	time.Sleep(time.Second)

	// only headers
	beforeSend := time.Now()
	for seq := uint64(6); seq < uint64(10); seq++ {
		for ep, os := range osMap {
			if ep == blockEP { //censorship
				continue
			}
			os.SendBlock(seq)
		}
	}

	time.Sleep(4 * time.Second)

	// the client detected the censorship and switched
	blockEP2, err := waitForBlockEP(bc)
	assert.NoError(t, err)
	assert.True(t, blockEP != blockEP2)

	for seq := uint64(6); seq < uint64(10); seq++ {
		for ep, os := range osMap {
			if ep != blockEP2 {
				continue
			}
			os.SendBlock(seq)
		}
	}

	time.Sleep(1 * time.Second)

	bc.Close()

	assert.Equal(t, uint64(10), bc.nextBlockNumber)
	assert.True(t, bc.lastBlockTime.After(beforeSend))

	for _, os := range osMap {
		os.Shutdown()
	}
}

// Scenario: server failover. Create a client against a set of orderer mocks.
// Receive one block. Then, the orderer sending blocks fails, but headers keep coming.
// The client should switch to another orderer and seek from the new height. Check block & header reception.
func TestBFTDeliverClient_Failover(t *testing.T) {
	flogging.ActivateSpec("bftDeliveryClient,deliveryClient=DEBUG")
	viper.Set("peer.deliveryclient.bft.blockRcvTotalBackoffDelay", time.Second)
	viper.Set("peer.deliveryclient.connTimeout", 100*time.Millisecond)
	defer viper.Reset()
	defer ensureNoGoroutineLeak(t)()

	osMap := make(map[string]*mocks.Orderer, len(endpointMap))
	for port, ep := range endpointMap {
		osMap[ep.Endpoint] = mocks.NewOrderer(port, t)
	}
	for _, os := range osMap {
		os.SetNextExpectedSeek(5)
	}

	connFactory := func(endpoint comm.EndpointCriteria) (*grpc.ClientConn, error) {
		ctx, cancel := context.WithTimeout(context.Background(), getConnectionTimeout())
		defer cancel()
		return grpc.DialContext(ctx, endpoint.Endpoint, grpc.WithInsecure(), grpc.WithBlock())
		//return grpc.Dial(endpoint.Endpoint, grpc.WithInsecure(), grpc.WithBlock())
	}
	var ledgerHeight uint64 = 5
	ledgerInfoMock := &mocks.LedgerInfo{}
	msgVerifierMock := &mocks.MessageCryptoVerifier{}
	bc := NewBFTDeliveryClient("test-chain", connFactory, endpoints, DefaultABCFactory, ledgerInfoMock, msgVerifierMock)
	ledgerInfoMock.On("LedgerHeight").Return(func() uint64 { return atomic.LoadUint64(&ledgerHeight) }, nil)
	msgVerifierMock.On("VerifyHeader", mock.AnythingOfType("string"), mock.AnythingOfType("*common.Block")).Return(nil)

	go func() {
		for {
			resp, err := bc.Recv()
			if err != nil {
				assert.EqualError(t, err, "client is closing")
				return
			}
			block := resp.GetBlock()
			assert.NotNil(t, block)
			if block == nil {
				return
			}
			atomic.StoreUint64(&ledgerHeight, block.Header.Number+1)
			bc.UpdateReceived(block.Header.Number)
		}
	}()

	blockEP, err := waitForBlockEP(bc)
	assert.NoError(t, err)
	// one normal block
	for _, os := range osMap {
		os.SendBlock(5)
	}
	for _, os := range osMap {
		os.SetNextExpectedSeek(6)
	}
	time.Sleep(time.Second)

	for ep, os := range osMap {
		if ep == blockEP {
			os.Shutdown()
			bftLogger.Infof("TEST: shuting down: %s", ep)
		}
	}

	time.Sleep(10 * time.Second)

	// the client detected the failure and switched
	blockEP2, err := waitForBlockEP(bc)
	assert.NoError(t, err)
	assert.True(t, blockEP != blockEP2)

	beforeSend := time.Now()
	for seq := uint64(6); seq < uint64(10); seq++ {
		for ep, os := range osMap {
			if ep == blockEP { // it is down
				continue
			}
			os.SendBlock(seq)
		}
	}

	time.Sleep(1 * time.Second)

	bc.Close()

	assert.Equal(t, uint64(10), bc.nextBlockNumber)
	assert.True(t, bc.lastBlockTime.After(beforeSend))

	for _, os := range osMap {
		os.Shutdown()
	}
}

func waitForBlockEP(bc *bftDeliveryClient) (string, error) {
	for i := int64(0); ; i++ {
		blockEP := bc.GetEndpoint()
		if len(blockEP) > 0 {
			return blockEP, nil
		}
		time.Sleep(10 * time.Millisecond)
		if time.Duration(i*10*time.Millisecond.Nanoseconds()) > 5*time.Second {
			return "", errors.New("timeout: no block receiver")
		}
	}
}
