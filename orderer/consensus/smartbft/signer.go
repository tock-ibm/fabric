/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package smartbft

import (
	"github.com/SmartBFT-Go/consensus/pkg/types"
	"github.com/hyperledger/fabric/internal/pkg/identity"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protoutil"
)

//go:generate mockery -dir . -name SignerSerializer -case underscore -output ./mocks/

type SignerSerializer interface {
	identity.SignerSerializer
}

type Signer struct {
	ID                 uint64
	SignerSerializer   identity.SignerSerializer
	Logger             PanicLogger
	LastConfigBlockNum func() uint64
}

func (s *Signer) Sign(msg []byte) []byte {
	signature, err := s.SignerSerializer.Sign(msg)
	if err != nil {
		s.Logger.Panicf("Failed signing message: %v", err)
	}
	return signature
}

func (s *Signer) SignProposal(proposal types.Proposal) *types.Signature {
	block, err := ProposalToBlock(proposal)
	if err != nil {
		s.Logger.Panicf("Tried to sign bad proposal: %v", err)
	}
	sig := Signature{
		BlockHeader:     protoutil.BlockHeaderBytes(block.Header),
		SignatureHeader: protoutil.MarshalOrPanic(protoutil.NewSignatureHeaderOrPanic(s.SignerSerializer)),
		OrdererBlockMetadata: protoutil.MarshalOrPanic(&common.OrdererBlockMetadata{
			LastConfig:        &common.LastConfig{Index: uint64(s.LastConfigBlockNum())},
			ConsenterMetadata: proposal.Metadata,
		}),
	}

	signature := protoutil.SignOrPanic(s.SignerSerializer, sig.AsBytes(s.SignerSerializer))
	return &types.Signature{
		ID:    s.ID,
		Value: signature,
		Msg:   sig.Marshal(),
	}
}
