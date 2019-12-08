/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package smartbft

import (
	"io/ioutil"
	"testing"

	"encoding/base64"
	"path/filepath"

	"crypto/sha256"
	"encoding/hex"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/crypto/tlsgen"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/orderer/common/cluster"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	"github.com/hyperledger/fabric/orderer/mocks/common/multichannel"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/stretchr/testify/assert"
)

func TestNewBlockPuller(t *testing.T) {
	ca, err := tlsgen.NewCA()
	assert.NoError(t, err)

	blockBytes, err := ioutil.ReadFile("testdata/mychannel.block")
	assert.NoError(t, err)

	goodConfigBlock := &common.Block{}
	assert.NoError(t, proto.Unmarshal(blockBytes, goodConfigBlock))

	lastBlock := &common.Block{
		Metadata: &common.BlockMetadata{
			Metadata: [][]byte{{}, protoutil.MarshalOrPanic(&common.Metadata{
				Value: protoutil.MarshalOrPanic(&common.LastConfig{Index: 42}),
			})},
		},
	}

	cs := &multichannel.ConsenterSupport{
		HeightVal: 100,
		BlockByIndex: map[uint64]*common.Block{
			42: goodConfigBlock,
			99: lastBlock,
		},
	}

	dialer := &cluster.PredicateDialer{
		Config: comm.ClientConfig{
			SecOpts: &comm.SecureOptions{
				Certificate: ca.CertBytes(),
			},
		},
	}

	blockPuller, err := newBlockPuller(cs, dialer,
		localconfig.Cluster{
			ReplicationMaxRetries: 2,
		})
	assert.NoError(t, err)
	assert.NotNil(t, blockPuller)
	blockPuller.Close()
}

func TestIsConsenterOfChannel(t *testing.T) {
	certInsideConfigBlock, err := base64.StdEncoding.DecodeString("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURDakNDQXJDZ0F3SUJB" +
		"Z0lRSVdvNWZYbDVFT3RIbmlqUXdMSVhXekFLQmdncWhrak9QUVFEQWpCc01Rc3cKQ1FZRFZRUUdFd0pWVXpFVE1CRUdBMVVFQ0JNS1EyRnNhV1p2Y201cFlUR" +
		"VdNQlFHQTFVRUJ4TU5VMkZ1SUVaeQpZVzVqYVhOamJ6RVVNQklHQTFVRUNoTUxaWGhoYlhCc1pTNWpiMjB4R2pBWUJnTlZCQU1URVhSc2MyTmhMbVY0CllXM" +
		"XdiR1V1WTI5dE1CNFhEVEU1TVRBeU1qRXlNVFF3TUZvWERUSTVNVEF4T1RFeU1UUXdNRm93V1RFTE1Ba0cKQTFVRUJoTUNWVk14RXpBUkJnTlZCQWdUQ2tOaG" +
		"JHbG1iM0p1YVdFeEZqQVVCZ05WQkFjVERWTmhiaUJHY21GdQpZMmx6WTI4eEhUQWJCZ05WQkFNVEZHOXlaR1Z5WlhJeUxtVjRZVzF3YkdVdVkyOXRNRmt3RX" +
		"dZSEtvWkl6ajBDCkFRWUlLb1pJemowREFRY0RRZ0FFS1ZIYnEzUENJTnJzdmpWRXQ1S29aNjFpbTQ0cStuNVFjd1ltaFhBUlVPSU0Kanl2dyt0bXpDcTlCeW" +
		"tLRTRBY1pKUHVib1lRa2JhS2RTMkVqbGF3YmJxT0NBVVV3Z2dGQk1BNEdBMVVkRHdFQgovd1FFQXdJRm9EQWRCZ05WSFNVRUZqQVVCZ2dyQmdFRkJRY0RBUV" +
		"lJS3dZQkJRVUhBd0l3REFZRFZSMFRBUUgvCkJBSXdBREFyQmdOVkhTTUVKREFpZ0NDMHhsZkxxOGNBTTNjdElJN2Jiek9HWFhtQ3JqeFBCb3pObCtKL2UrSj" +
		"cKRXpDQjFBWURWUjBSQklITU1JSEpnaFJ2Y21SbGNtVnlNaTVsZUdGdGNHeGxMbU52YllJSWIzSmtaWEpsY2pLQwpDV3h2WTJGc2FHOXpkSUlKTVRJM0xqQX" +
		"VNQzR4Z2dNNk9qR0NGRzl5WkdWeVpYSXlMbVY0WVcxd2JHVXVZMjl0CmdnaHZjbVJsY21WeU1vSUpiRzlqWVd4b2IzTjBnZ2t4TWpjdU1DNHdMakdDQXpvNk" +
		"1ZSVViM0prWlhKbGNqSXUKWlhoaGJYQnNaUzVqYjIyQ0NHOXlaR1Z5WlhJeWdnbHNiMk5oYkdodmMzU0NDVEV5Tnk0d0xqQXVNWUlET2pveApod1IvQUFBQ" +
		"mh4QUFBQUFBQUFBQUFBQUFBQUFBQUFBQk1Bb0dDQ3FHU000OUJBTUNBMGdBTUVVQ0lRQ1Nod1N1CitLd3YvaVdobHVJMU5UNDYyYzFNamNvam8wNTVKOG9Q" +
		"M0VLKzN3SWdhMFhqT0JZd3dFSTZQRlZCY0V5MjA4TG4Kc0NCUE5NdTdNeFFRR3NFbnc3ST0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=")
	assert.NoError(t, err)

	loadBlock := func(fileName string) *common.Block {
		b, err := ioutil.ReadFile(filepath.Join("testdata", fileName))
		assert.NoError(t, err)
		block := &common.Block{}
		err = proto.Unmarshal(b, block)
		assert.NoError(t, err)
		return block
	}
	for _, testCase := range []struct {
		name          string
		expectedError string
		configBlock   *common.Block
		certificate   []byte
	}{
		{
			name:          "nil block",
			expectedError: "nil block",
		},
		{
			name:          "no block data",
			expectedError: "block data is nil",
			configBlock:   &common.Block{},
		},
		{
			name: "invalid envelope inside block",
			expectedError: "failed to unmarshal payload from envelope:" +
				" error unmarshaling Payload: proto: common.Payload: illegal tag 0 (wire type 1)",
			configBlock: &common.Block{
				Data: &common.BlockData{
					Data: [][]byte{protoutil.MarshalOrPanic(&common.Envelope{
						Payload: []byte{1, 2, 3},
					})},
				},
			},
		},
		{
			name:          "valid config block with cert mismatch",
			configBlock:   loadBlock("smartbft_genesis_block.pb"),
			certificate:   certInsideConfigBlock[2:],
			expectedError: cluster.ErrNotInChannel.Error(),
		},
		{
			name:          "etcdraft genesis block",
			configBlock:   loadBlock("etcdraftgenesis.block"),
			expectedError: "not a SmartBFT config block",
		},
		{
			name:        "valid config block with matching cert",
			configBlock: loadBlock("smartbft_genesis_block.pb"),
			certificate: certInsideConfigBlock,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := ConsenterCertificate(testCase.certificate).IsConsenterOfChannel(testCase.configBlock)
			if testCase.expectedError != "" {
				assert.EqualError(t, err, testCase.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func makeTx(nonce, creator []byte) []byte {
	return protoutil.MarshalOrPanic(&common.Envelope{
		Payload: protoutil.MarshalOrPanic(&common.Payload{
			Header: &common.Header{
				ChannelHeader: protoutil.MarshalOrPanic(&common.ChannelHeader{
					Type:      int32(common.HeaderType_ENDORSER_TRANSACTION),
					ChannelId: "test-chain",
				}),
				SignatureHeader: protoutil.MarshalOrPanic(&common.SignatureHeader{
					Creator: creator,
					Nonce:   nonce,
				}),
			},
		}),
	})
}

func TestRequestID(t *testing.T) {
	ri := &RequestInspector{
		ValidateIdentityStructure: func(identity *msp.SerializedIdentity) error {
			return nil
		},
	}

	nonce := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	creator := protoutil.MarshalOrPanic(&msp.SerializedIdentity{
		Mspid:   "SampleOrg",
		IdBytes: []byte{1, 2, 3},
	})

	tx := makeTx(nonce, creator)

	var expectedTxID []byte
	expectedTxID = append(expectedTxID, nonce...)
	expectedTxID = append(expectedTxID, creator...)

	txID := sha256.Sum256(expectedTxID)
	expectedTxString := hex.EncodeToString(txID[:])

	expectedClient := sha256.Sum256(creator)

	info := ri.RequestID(tx)
	assert.Equal(t, expectedTxString, info.ID)
	assert.Equal(t, hex.EncodeToString(expectedClient[:]), info.ClientID)
}
