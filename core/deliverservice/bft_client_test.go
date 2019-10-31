/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deliverclient

import (
	"testing"
	"time"

	"github.com/hyperledger/fabric/gossip/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/mock"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/core/deliverservice/mocks"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type MsgCryptoSrv interface {
	api.MessageCryptoService
}

var endpoints = []comm.EndpointCriteria{
	{Endpoint: "localhost:5611", Organizations: []string{"org1"}},
	{Endpoint: "localhost:5612", Organizations: []string{"org2"}},
	{Endpoint: "localhost:5613", Organizations: []string{"org3"}},
	{Endpoint: "localhost:5614", Organizations: []string{"org4"}},
}

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

func TestBFTDeliverClient_Recv(t *testing.T) {
	flogging.ActivateSpec("bftDeliveryClient=DEBUG")

	defer ensureNoGoroutineLeak(t)()
	// This test configures the client in a similar fashion as will be
	// in production, and tests against a live gRPC server.
	osArray := []*mocks.Orderer{
		mocks.NewOrderer(5611, t),
		mocks.NewOrderer(5612, t),
		mocks.NewOrderer(5613, t),
		mocks.NewOrderer(5614, t)}
	for _, os := range osArray {
		os.SetNextExpectedSeek(5)
	}

	connFactory := func(endpoint comm.EndpointCriteria) (*grpc.ClientConn, error) {
		return grpc.Dial(endpoint.Endpoint, grpc.WithInsecure(), grpc.WithBlock())
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

	beforeSend := time.Now()
	for seq := uint64(5); seq < uint64(10); seq++ {
		for _, os := range osArray {
			os.SendBlock(seq)
		}
	}

	time.Sleep(time.Second)
	bc.Close()

	assert.Equal(t, uint64(10), bc.nextBlockNumber)
	assert.True(t, bc.lastBlockTime.After(beforeSend))

	for _, os := range osArray {
		os.Shutdown()
	}
}

func TestBFTDeliverClient_RecvCensorship(t *testing.T) {
	flogging.ActivateSpec("bftDeliveryClient=DEBUG")
	viper.Set("peer.deliveryclient.bft.blockCensorshipTimeout", 2*time.Second)
	defer viper.Reset()
	defer ensureNoGoroutineLeak(t)()

	assert.Equal(t, util.GetDurationOrDefault("peer.deliveryclient.bft.blockCensorshipTimeout", bftBlockCensorshipTimeout), 2*time.Second)

	// This test configures the client in a similar fashion as will be
	// in production, and tests against a live gRPC server.
	osMap := map[string]*mocks.Orderer{
		"localhost:5611": mocks.NewOrderer(5611, t),
		"localhost:5612": mocks.NewOrderer(5612, t),
		"localhost:5613": mocks.NewOrderer(5613, t),
		"localhost:5614": mocks.NewOrderer(5614, t),
	}
	for _, os := range osMap {
		os.SetNextExpectedSeek(5)
	}

	connFactory := func(endpoint comm.EndpointCriteria) (*grpc.ClientConn, error) {
		return grpc.Dial(endpoint.Endpoint, grpc.WithInsecure(), grpc.WithBlock())
	}
	ledgerInfoMock := &mocks.LedgerInfo{}
	msgVerifierMock := &mocks.MessageCryptoVerifier{}
	bc := NewBFTDeliveryClient("test-chain", connFactory, endpoints, DefaultABCFactory, ledgerInfoMock, msgVerifierMock)
	ledgerInfoMock.On("LedgerHeight").Return(uint64(5), nil)
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
			bc.UpdateReceived(block.Header.Number)
		}
	}()

	var blockEP string
	for i := 0; i < 100; i++ {
		blockEP = bc.GetEndpoint()
		if len(blockEP) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	assert.True(t, len(blockEP) > 0)

	beforeSend := time.Now()
	for seq := uint64(5); seq < uint64(10); seq++ {
		for ep, os := range osMap {
			if seq > 5 && ep == blockEP { //censorship
				continue
			}
			os.SendBlock(seq)
		}
	}

	time.Sleep(4 * time.Second)

	blockEP2 := bc.GetEndpoint()
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
