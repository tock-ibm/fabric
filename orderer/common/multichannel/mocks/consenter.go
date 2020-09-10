// Code generated by counterfeiter. DO NOT EDIT.
package mocks

import (
	"sync"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric/orderer/consensus"
)

type Consenter struct {
	HandleChainStub        func(consensus.ConsenterSupport, *common.Metadata) (consensus.Chain, error)
	handleChainMutex       sync.RWMutex
	handleChainArgsForCall []struct {
		arg1 consensus.ConsenterSupport
		arg2 *common.Metadata
	}
	handleChainReturns struct {
		result1 consensus.Chain
		result2 error
	}
	handleChainReturnsOnCall map[int]struct {
		result1 consensus.Chain
		result2 error
	}
	IsChannelMemberStub        func(*common.Block) (bool, error)
	isChannelMemberMutex       sync.RWMutex
	isChannelMemberArgsForCall []struct {
		arg1 *common.Block
	}
	isChannelMemberReturns struct {
		result1 bool
		result2 error
	}
	isChannelMemberReturnsOnCall map[int]struct {
		result1 bool
		result2 error
	}
	RemoveInactiveChainRegistryStub        func()
	removeInactiveChainRegistryMutex       sync.RWMutex
	removeInactiveChainRegistryArgsForCall []struct {
	}
	SetInactiveChainRegistryStub        func(consensus.InactiveChainRegistry)
	setInactiveChainRegistryMutex       sync.RWMutex
	setInactiveChainRegistryArgsForCall []struct {
		arg1 consensus.InactiveChainRegistry
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *Consenter) HandleChain(arg1 consensus.ConsenterSupport, arg2 *common.Metadata) (consensus.Chain, error) {
	fake.handleChainMutex.Lock()
	ret, specificReturn := fake.handleChainReturnsOnCall[len(fake.handleChainArgsForCall)]
	fake.handleChainArgsForCall = append(fake.handleChainArgsForCall, struct {
		arg1 consensus.ConsenterSupport
		arg2 *common.Metadata
	}{arg1, arg2})
	fake.recordInvocation("HandleChain", []interface{}{arg1, arg2})
	fake.handleChainMutex.Unlock()
	if fake.HandleChainStub != nil {
		return fake.HandleChainStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.handleChainReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *Consenter) HandleChainCallCount() int {
	fake.handleChainMutex.RLock()
	defer fake.handleChainMutex.RUnlock()
	return len(fake.handleChainArgsForCall)
}

func (fake *Consenter) HandleChainCalls(stub func(consensus.ConsenterSupport, *common.Metadata) (consensus.Chain, error)) {
	fake.handleChainMutex.Lock()
	defer fake.handleChainMutex.Unlock()
	fake.HandleChainStub = stub
}

func (fake *Consenter) HandleChainArgsForCall(i int) (consensus.ConsenterSupport, *common.Metadata) {
	fake.handleChainMutex.RLock()
	defer fake.handleChainMutex.RUnlock()
	argsForCall := fake.handleChainArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *Consenter) HandleChainReturns(result1 consensus.Chain, result2 error) {
	fake.handleChainMutex.Lock()
	defer fake.handleChainMutex.Unlock()
	fake.HandleChainStub = nil
	fake.handleChainReturns = struct {
		result1 consensus.Chain
		result2 error
	}{result1, result2}
}

func (fake *Consenter) HandleChainReturnsOnCall(i int, result1 consensus.Chain, result2 error) {
	fake.handleChainMutex.Lock()
	defer fake.handleChainMutex.Unlock()
	fake.HandleChainStub = nil
	if fake.handleChainReturnsOnCall == nil {
		fake.handleChainReturnsOnCall = make(map[int]struct {
			result1 consensus.Chain
			result2 error
		})
	}
	fake.handleChainReturnsOnCall[i] = struct {
		result1 consensus.Chain
		result2 error
	}{result1, result2}
}

func (fake *Consenter) IsChannelMember(arg1 *common.Block) (bool, error) {
	fake.isChannelMemberMutex.Lock()
	ret, specificReturn := fake.isChannelMemberReturnsOnCall[len(fake.isChannelMemberArgsForCall)]
	fake.isChannelMemberArgsForCall = append(fake.isChannelMemberArgsForCall, struct {
		arg1 *common.Block
	}{arg1})
	fake.recordInvocation("IsChannelMember", []interface{}{arg1})
	fake.isChannelMemberMutex.Unlock()
	if fake.IsChannelMemberStub != nil {
		return fake.IsChannelMemberStub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.isChannelMemberReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *Consenter) IsChannelMemberCallCount() int {
	fake.isChannelMemberMutex.RLock()
	defer fake.isChannelMemberMutex.RUnlock()
	return len(fake.isChannelMemberArgsForCall)
}

func (fake *Consenter) IsChannelMemberCalls(stub func(*common.Block) (bool, error)) {
	fake.isChannelMemberMutex.Lock()
	defer fake.isChannelMemberMutex.Unlock()
	fake.IsChannelMemberStub = stub
}

func (fake *Consenter) IsChannelMemberArgsForCall(i int) *common.Block {
	fake.isChannelMemberMutex.RLock()
	defer fake.isChannelMemberMutex.RUnlock()
	argsForCall := fake.isChannelMemberArgsForCall[i]
	return argsForCall.arg1
}

func (fake *Consenter) IsChannelMemberReturns(result1 bool, result2 error) {
	fake.isChannelMemberMutex.Lock()
	defer fake.isChannelMemberMutex.Unlock()
	fake.IsChannelMemberStub = nil
	fake.isChannelMemberReturns = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *Consenter) IsChannelMemberReturnsOnCall(i int, result1 bool, result2 error) {
	fake.isChannelMemberMutex.Lock()
	defer fake.isChannelMemberMutex.Unlock()
	fake.IsChannelMemberStub = nil
	if fake.isChannelMemberReturnsOnCall == nil {
		fake.isChannelMemberReturnsOnCall = make(map[int]struct {
			result1 bool
			result2 error
		})
	}
	fake.isChannelMemberReturnsOnCall[i] = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *Consenter) RemoveInactiveChainRegistry() {
	fake.removeInactiveChainRegistryMutex.Lock()
	fake.removeInactiveChainRegistryArgsForCall = append(fake.removeInactiveChainRegistryArgsForCall, struct {
	}{})
	fake.recordInvocation("RemoveInactiveChainRegistry", []interface{}{})
	fake.removeInactiveChainRegistryMutex.Unlock()
	if fake.RemoveInactiveChainRegistryStub != nil {
		fake.RemoveInactiveChainRegistryStub()
	}
}

func (fake *Consenter) RemoveInactiveChainRegistryCallCount() int {
	fake.removeInactiveChainRegistryMutex.RLock()
	defer fake.removeInactiveChainRegistryMutex.RUnlock()
	return len(fake.removeInactiveChainRegistryArgsForCall)
}

func (fake *Consenter) RemoveInactiveChainRegistryCalls(stub func()) {
	fake.removeInactiveChainRegistryMutex.Lock()
	defer fake.removeInactiveChainRegistryMutex.Unlock()
	fake.RemoveInactiveChainRegistryStub = stub
}

func (fake *Consenter) SetInactiveChainRegistry(arg1 consensus.InactiveChainRegistry) {
	fake.setInactiveChainRegistryMutex.Lock()
	fake.setInactiveChainRegistryArgsForCall = append(fake.setInactiveChainRegistryArgsForCall, struct {
		arg1 consensus.InactiveChainRegistry
	}{arg1})
	fake.recordInvocation("SetInactiveChainRegistry", []interface{}{arg1})
	fake.setInactiveChainRegistryMutex.Unlock()
	if fake.SetInactiveChainRegistryStub != nil {
		fake.SetInactiveChainRegistryStub(arg1)
	}
}

func (fake *Consenter) SetInactiveChainRegistryCallCount() int {
	fake.setInactiveChainRegistryMutex.RLock()
	defer fake.setInactiveChainRegistryMutex.RUnlock()
	return len(fake.setInactiveChainRegistryArgsForCall)
}

func (fake *Consenter) SetInactiveChainRegistryCalls(stub func(consensus.InactiveChainRegistry)) {
	fake.setInactiveChainRegistryMutex.Lock()
	defer fake.setInactiveChainRegistryMutex.Unlock()
	fake.SetInactiveChainRegistryStub = stub
}

func (fake *Consenter) SetInactiveChainRegistryArgsForCall(i int) consensus.InactiveChainRegistry {
	fake.setInactiveChainRegistryMutex.RLock()
	defer fake.setInactiveChainRegistryMutex.RUnlock()
	argsForCall := fake.setInactiveChainRegistryArgsForCall[i]
	return argsForCall.arg1
}

func (fake *Consenter) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.handleChainMutex.RLock()
	defer fake.handleChainMutex.RUnlock()
	fake.isChannelMemberMutex.RLock()
	defer fake.isChannelMemberMutex.RUnlock()
	fake.removeInactiveChainRegistryMutex.RLock()
	defer fake.removeInactiveChainRegistryMutex.RUnlock()
	fake.setInactiveChainRegistryMutex.RLock()
	defer fake.setInactiveChainRegistryMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *Consenter) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}
