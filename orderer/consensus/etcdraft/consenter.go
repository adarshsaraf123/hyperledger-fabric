/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package etcdraft

import (
	"bytes"
	"reflect"
	"time"

	"code.cloudfoundry.org/clock"
	"github.com/coreos/etcd/raft"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/orderer/common/cluster"
	"github.com/hyperledger/fabric/orderer/common/multichannel"
	"github.com/hyperledger/fabric/orderer/consensus"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer"
	"github.com/hyperledger/fabric/protos/orderer/etcdraft"
	"github.com/pkg/errors"
)

//go:generate mockery -dir . -name ChainGetter -case underscore -output mocks

type ChainGetter interface {
	GetChain(chainID string) (*multichannel.ChainSupport, bool)
}

//go:generate mockery -dir . -name Configurator -case underscore -output mocks
type Configurator interface {
	Configure(channel string, newNodes []cluster.RemoteNode)
}

// Consenter implements etddraft consenter
type Consenter struct {
	Configurator Configurator
	*Dispatcher
	Chains ChainGetter
	Logger *flogging.FabricLogger
	Cert   []byte
}

func (c *Consenter) TargetChannel(message proto.Message) string {
	if stepReq, isStepReq := message.(*orderer.StepRequest); isStepReq {
		return stepReq.Channel
	}
	if submitReq, isSubmitReq := message.(*orderer.SubmitRequest); isSubmitReq {
		return submitReq.Channel
	}
	return ""
}

func (c *Consenter) ReceiverByChain(channelID string) MessageReceiver {
	chain, exists := c.Chains.GetChain(channelID)
	if !exists {
		return nil
	}
	if chain.Chain == nil {
		c.Logger.Panicf("Programming error - Chain %s is nil although it exists in the mapping", channelID)
	}
	if etcdRaftChain, isEtcdRaftChain := chain.Chain.(*Chain); isEtcdRaftChain {
		return etcdRaftChain
	}
	c.Logger.Warningf("Chain %s is of type %v and not etcdraft.Chain", channelID, reflect.TypeOf(chain.Chain))
	return nil
}

func (c *Consenter) detectSelfID(m *etcdraft.Metadata) (uint64, error) {
	var serverCertificates []string
	for i, cst := range m.Consenters {
		serverCertificates = append(serverCertificates, string(cst.ServerTlsCert))
		if bytes.Equal(c.Cert, cst.ServerTlsCert) {
			return uint64(i + 1), nil
		}
	}

	c.Logger.Error("Could not find", string(c.Cert), "among", serverCertificates)
	return 0, errors.Errorf("failed to detect own Raft ID because no matching certificate found")
}

func (c *Consenter) HandleChain(support consensus.ConsenterSupport, metadata *common.Metadata) (consensus.Chain, error) {
	m := &etcdraft.Metadata{}
	if err := proto.Unmarshal(support.SharedConfig().ConsensusMetadata(), m); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal consensus metadata")
	}
	id, err := c.detectSelfID(m)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	peers := make([]raft.Peer, len(m.Consenters))
	for i := range peers {
		peers[i].ID = uint64(i + 1)
	}

	opts := Options{
		RaftID:  id,
		Clock:   clock.NewClock(),
		Storage: raft.NewMemoryStorage(),
		Logger:  c.Logger,

		// TODO make options for raft configurable via channel configs
		TickInterval:    100 * time.Millisecond,
		ElectionTick:    10,
		HeartbeatTick:   1,
		MaxInflightMsgs: 256,
		MaxSizePerMsg:   1024 * 1024, // This could potentially be deduced from max block size

		Peers: peers,
	}

	return NewChain(support, opts, nil, c.Configurator)
}
