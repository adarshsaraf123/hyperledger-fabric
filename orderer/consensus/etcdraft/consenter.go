/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package etcdraft

import (
	"bytes"
	"io/ioutil"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	"github.com/hyperledger/fabric/orderer/consensus"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/orderer/etcdraft"

	"code.cloudfoundry.org/clock"
	"github.com/coreos/etcd/raft"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

type Consenter struct {
	cert   []byte
	logger *flogging.FabricLogger
}

func New(conf *localconfig.TopLevel, logger *flogging.FabricLogger) *Consenter {
	cert, err := ioutil.ReadFile(conf.General.TLS.Certificate)
	if err != nil {
		logger.Panicf("Failed to read TLS certificate %s: %v", conf.General.TLS.Certificate, err)
	}

	return &Consenter{
		cert:   cert,
		logger: logger,
	}
}

func (c *Consenter) HandleChain(support consensus.ConsenterSupport, metadata *common.Metadata) (consensus.Chain, error) {
	m := &etcdraft.Metadata{}
	if err := proto.Unmarshal(support.SharedConfig().ConsensusMetadata(), m); err != nil {
		return nil, errors.Errorf("failed to unmarshal consensus metadata: %s", err)
	}

	var id uint64
	for i, cst := range m.Consenters {
		if bytes.Equal(c.cert, cst.ServerTlsCert) {
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
		RaftID:  id,
		Clock:   clock.NewClock(),
		Storage: raft.NewMemoryStorage(),
		Logger:  c.logger,

		// TODO make options for raft configurable via channel configs
		TickInterval:    100 * time.Millisecond,
		ElectionTick:    10,
		HeartbeatTick:   1,
		MaxInflightMsgs: 256,
		MaxSizePerMsg:   1024 * 1024, // This could potentially be deduced from max block size

		Peers: peers,
	}

	return NewChain(support, opts, nil)
}
