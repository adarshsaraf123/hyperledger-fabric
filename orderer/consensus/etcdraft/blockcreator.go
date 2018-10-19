/*
Copyright IBM Corp. 2017 All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package etcdraft

import (
	"bytes"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	cb "github.com/hyperledger/fabric/protos/common"
)

// This governs the max number of created blocks in-flight; i.e. blocks
// that were created but not written.
// CreateNextBLock returns nil once this number of blocks are in-flight.
const createdBlocksBuffersize = 20

// blockCreator optimistically creates blocks in a chain. The created
// blocks may not be written out eventually. This enables us to pipeline
// the creation of blocks with achieving consensus on them leading to
// performance improvements. The created chain is discarded if a
// diverging block is committed
// To safely use blockCreator, only one thread should interact with it.
type blockCreator struct {
	createdBlocks      chan *cb.Block
	lastCreatedBlock   *cb.Block
	lastCommittedBlock *cb.Block
	logger             *flogging.FabricLogger
}

func newBlockCreator(lastBlock *cb.Block, logger *flogging.FabricLogger) *blockCreator {
	bc := &blockCreator{
		createdBlocks:      make(chan *cb.Block, createdBlocksBuffersize),
		lastCreatedBlock:   lastBlock,
		lastCommittedBlock: lastBlock,
		logger:             logger,
	}

	logger.Debugf("Initialized block creator with (lastblockNumber=%d)", lastBlock.Header.Number)
	return bc
}

// CreateNextBlock creates a new block with the next block number, and the
// given contents.
// Returns the created block if the block could be created else nil.
// It can fail when the there are `createdBlocksBuffersize` blocks already
// created and no more can be accomodated in the `createdBlocks` channel.
func (bc *blockCreator) createNextBlock(messages []*cb.Envelope) *cb.Block {
	previousBlockHash := bc.lastCreatedBlock.Header.Hash()

	data := &cb.BlockData{
		Data: make([][]byte, len(messages)),
	}

	var err error
	for i, msg := range messages {
		data.Data[i], err = proto.Marshal(msg)
		if err != nil {
			bc.logger.Panicf("Could not marshal envelope: %s", err)
		}
	}

	block := cb.NewBlock(bc.lastCreatedBlock.Header.Number+1, previousBlockHash)
	block.Header.DataHash = data.Hash()
	block.Data = data

	select {
	case bc.createdBlocks <- block:
		bc.lastCreatedBlock = block
		bc.logger.Debugf("Created block %d", bc.lastCreatedBlock.Header.Number)
		return block
	default:
		return nil
	}
}

// ResetCreatedBlocks resets the queue of created blocks.
// Subsequent blocks will be created over the block that was last committed
// using CommitBlock.
func (bc *blockCreator) resetCreatedBlocks() {
	bc.createdBlocks = make(chan *cb.Block, createdBlocksBuffersize)
	bc.lastCreatedBlock = bc.lastCommittedBlock
}

// commitBlock should be invoked for all blocks to let the blockCreator know
// which blocks have been committed. If the committed block is divergent from
// the stack of created blocks then the stack is reset.
func (bc *blockCreator) commitBlock(block *cb.Block) {
	bc.lastCommittedBlock = block

	// check if the committed block diverges from the created blocks
	select {
	case b := <-bc.createdBlocks:
		if !bytes.Equal(b.Header.Bytes(), block.Header.Bytes()) {
			// the written block is diverging from the createBlocks stack
			// discard the created blocks
			bc.resetCreatedBlocks()
		}
		// else the written block is part of the createBlocks stack; nothing to be done
	default:
		// No created blocks; set this block as the last created block.
		// This happens when calls to WriteBlock are being made without a CreateNextBlock being called.
		// For example, in the case of etcd/raft, the leader proposes blocks and the followers
		// only write the proposed blocks.
		bc.lastCreatedBlock = block
	}
}
