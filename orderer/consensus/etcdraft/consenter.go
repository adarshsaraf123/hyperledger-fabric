/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package etcdraft

import (
	"bytes"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	"github.com/hyperledger/fabric/orderer/consensus"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer/etcdraft"

	"code.cloudfoundry.org/clock"
	"github.com/coreos/etcd/raft"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Consenter struct {
	Config localconfig.EtcdRaft
	Cert   []byte
}

func (c *Consenter) HandleChain(support consensus.ConsenterSupport, metadata *common.Metadata) (consensus.Chain, error) {
	m := &etcdraft.Metadata{}
	if err := proto.Unmarshal(support.SharedConfig().ConsensusMetadata(), m); err != nil {
		return nil, errors.Errorf("failed to unmarshal consensus metadata: %s", err)
	}

	var id uint64
	for i, cst := range m.Consenters {
		if bytes.Equal(c.Cert, cst.ServerTlsCert) {
			id = uint64(i + 1)
			break
		}
	}

	if id == 0 {
		return nil, errors.Errorf("failed to detect own raft ID because no matching certificate found")
	}

	peers := make([]raft.Peer, len(m.Consenters))
	for i := range peers {
		peers[i].ID = uint64(i + 1)
	}

	opts := Options{
		RaftID:          id,
		Clock:           clock.NewClock(),
		Storage:         raft.NewMemoryStorage(),
		Logger:          flogging.NewFabricLogger(zap.NewExample()),
		TickInterval:    c.Config.TickInterval,
		ElectionTick:    c.Config.ElectionTick,
		HeartbeatTick:   c.Config.HeartbeatTick,
		MaxInflightMsgs: c.Config.MaxInflightMsgs,
		Peers:           peers,
	}

	return NewChain(support, opts, nil)
}
