/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

                 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package multichannel

import (
	"bytes"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"

	newchannelconfig "github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/ledger/blockledger"
	mockconfigtx "github.com/hyperledger/fabric/common/mocks/configtx"
	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/utils"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

type mockBlockWriterSupport struct {
	*mockconfigtx.Validator
	crypto.LocalSigner
	blockledger.ReadWriter
}

func (mbws mockBlockWriterSupport) Update(bundle *newchannelconfig.Bundle) {}

func (mbws mockBlockWriterSupport) CreateBundle(channelID string, config *cb.Config) (*newchannelconfig.Bundle, error) {
	return nil, nil
}

func (mbws mockBlockWriterSupport) ChainID() string {
	return genesisconfig.TestChainID
}

func TestCreateBlock(t *testing.T) {
	seedBlock := cb.NewBlock(7, []byte("lasthash"))
	seedBlock.Data.Data = [][]byte{[]byte("somebytes")}

	bw := &BlockWriter{
		lastCreatedBlock: seedBlock,
		createdBlocks:    make(chan *cb.Block, 10),
		support:          &mockBlockWriterSupport{},
	}
	block := bw.CreateNextBlock([]*cb.Envelope{
		{Payload: []byte("some other bytes")},
	})

	assert.Equal(t, seedBlock.Header.Number+1, block.Header.Number)
	assert.Equal(t, block.Data.Hash(), block.Header.DataHash)
	assert.Equal(t, seedBlock.Header.Hash(), block.Header.PreviousHash)
}

func TestBlockSignature(t *testing.T) {
	bw := &BlockWriter{
		support: &mockBlockWriterSupport{
			LocalSigner: mockCrypto(),
		},
	}

	block := cb.NewBlock(7, []byte("foo"))
	bw.addBlockSignature(block)

	md := utils.GetMetadataFromBlockOrPanic(block, cb.BlockMetadataIndex_SIGNATURES)
	assert.Nil(t, md.Value, "Value is empty in this case")
	assert.NotNil(t, md.Signatures, "Should have signature")
}

func TestBlockLastConfig(t *testing.T) {
	lastConfigSeq := uint64(6)
	newConfigSeq := lastConfigSeq + 1
	newBlockNum := uint64(9)

	bw := &BlockWriter{
		support: &mockBlockWriterSupport{
			LocalSigner: mockCrypto(),
			Validator: &mockconfigtx.Validator{
				SequenceVal: newConfigSeq,
			},
		},
		lastConfigSeq: lastConfigSeq,
	}

	block := cb.NewBlock(newBlockNum, []byte("foo"))
	bw.addLastConfigSignature(block)

	assert.Equal(t, newBlockNum, bw.lastConfigBlockNum)
	assert.Equal(t, newConfigSeq, bw.lastConfigSeq)

	md := utils.GetMetadataFromBlockOrPanic(block, cb.BlockMetadataIndex_LAST_CONFIG)
	assert.NotNil(t, md.Value, "Value not be empty in this case")
	assert.NotNil(t, md.Signatures, "Should have signature")

	lc := utils.GetLastConfigIndexFromBlockOrPanic(block)
	assert.Equal(t, newBlockNum, lc)
}

func TestWriteConfigBlock(t *testing.T) {
	// TODO, use assert.PanicsWithValue once available
	t.Run("EmptyBlock", func(t *testing.T) {
		assert.Panics(t, func() { (&BlockWriter{}).WriteConfigBlock(&cb.Block{}, nil) })
	})
	t.Run("BadPayload", func(t *testing.T) {
		assert.Panics(t, func() {
			(&BlockWriter{}).WriteConfigBlock(&cb.Block{
				Data: &cb.BlockData{
					Data: [][]byte{
						utils.MarshalOrPanic(&cb.Envelope{Payload: []byte("bad")}),
					},
				},
			}, nil)
		})
	})
	t.Run("MissingHeader", func(t *testing.T) {
		assert.Panics(t, func() {
			(&BlockWriter{}).WriteConfigBlock(&cb.Block{
				Data: &cb.BlockData{
					Data: [][]byte{
						utils.MarshalOrPanic(&cb.Envelope{
							Payload: utils.MarshalOrPanic(&cb.Payload{}),
						}),
					},
				},
			}, nil)
		})
	})
	t.Run("BadChannelHeader", func(t *testing.T) {
		assert.Panics(t, func() {
			(&BlockWriter{}).WriteConfigBlock(&cb.Block{
				Data: &cb.BlockData{
					Data: [][]byte{
						utils.MarshalOrPanic(&cb.Envelope{
							Payload: utils.MarshalOrPanic(&cb.Payload{
								Header: &cb.Header{
									ChannelHeader: []byte("bad"),
								},
							}),
						}),
					},
				},
			}, nil)
		})
	})
	t.Run("BadChannelHeaderType", func(t *testing.T) {
		assert.Panics(t, func() {
			(&BlockWriter{}).WriteConfigBlock(&cb.Block{
				Data: &cb.BlockData{
					Data: [][]byte{
						utils.MarshalOrPanic(&cb.Envelope{
							Payload: utils.MarshalOrPanic(&cb.Payload{
								Header: &cb.Header{
									ChannelHeader: utils.MarshalOrPanic(&cb.ChannelHeader{}),
								},
							}),
						}),
					},
				},
			}, nil)
		})
	})
}

func TestGoodWriteConfig(t *testing.T) {
	l := NewRAMLedger(10)

	bw := &BlockWriter{
		support: &mockBlockWriterSupport{
			LocalSigner: mockCrypto(),
			ReadWriter:  l,
			Validator:   &mockconfigtx.Validator{},
		},
	}

	ctx := makeConfigTx(genesisconfig.TestChainID, 1)
	block := cb.NewBlock(1, genesisBlock.Header.Hash())
	block.Data.Data = [][]byte{utils.MarshalOrPanic(ctx)}
	consenterMetadata := []byte("foo")
	bw.WriteConfigBlock(block, consenterMetadata)

	// Wait for the commit to complete
	bw.writingBlock.Lock()
	bw.writingBlock.Unlock()

	cBlock := blockledger.GetBlock(l, block.Header.Number)
	assert.Equal(t, block.Header, cBlock.Header)
	assert.Equal(t, block.Data, cBlock.Data)

	omd := utils.GetMetadataFromBlockOrPanic(block, cb.BlockMetadataIndex_ORDERER)
	assert.Equal(t, consenterMetadata, omd.Value)
}

func TestValidCreatedBlocksQueue(t *testing.T) {
	logger.Info("testing propose blocks")
	l := NewRAMLedger(10)

	bw := &BlockWriter{
		support: &mockBlockWriterSupport{
			LocalSigner: mockCrypto(),
			ReadWriter:  l,
			Validator:   &mockconfigtx.Validator{},
		},
		createdBlocks: make(chan *cb.Block, createdBlocksBuffersize),
	}

	ctx := makeConfigTx(genesisconfig.TestChainID, 1)
	block := cb.NewBlock(1, genesisBlock.Header.Hash())
	block.Data.Data = [][]byte{utils.MarshalOrPanic(ctx)}
	consenterMetadata := []byte("foo")
	bw.WriteConfigBlock(block, consenterMetadata)

	// Wait for the commit to complete
	bw.writingBlock.Lock()
	bw.writingBlock.Unlock()

	t.Run("propose multiple blocks without having to write them out; decouples CreateNextBlock from WriteBlock ", func(t *testing.T) {
		/*
		 * Scenario:
		 *   We create five blocks initially and then write only two of them. We further create five more blocks
		 *   and write out the remaining 8 blocks in the propose stack. This should succeed since the written
		 *   blocks are not divergent from the created blocks.
		 * This scenario fails if WriteBlock and CreateNextBlock are not decoupled.
		 */
		assert.NotPanics(t, func() {
			blocks := []*cb.Block{}
			// Create five blocks without writing them out
			for i := 0; i < 5; i++ {
				blocks = append(blocks, bw.CreateNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}}))
			}

			// Write two of these out
			for i := 0; i < 2; i++ {
				bw.WriteBlock(blocks[i], nil)
			}

			// Create five more blocks; these should be created over the previous five blocks created
			for i := 0; i < 5; i++ {
				blocks = append(blocks, bw.CreateNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}}))
			}

			// Write out the remaining eight blocks; can only succeed if all the blocks were created in a single stack else will panic
			for i := 2; i < 10; i++ {
				bw.WriteBlock(blocks[i], nil)
			}
			// 11 blocks have been written so far
		})
	})

	t.Run("CreateNextBlock returns nil after createdBlocksBuffersize blocks have been created", func(t *testing.T) {
		numBlocks := createdBlocksBuffersize
		blocks := []*cb.Block{}

		for i := 0; i < numBlocks; i++ {
			blocks = append(blocks, bw.CreateNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}}))
		}

		block := bw.CreateNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}})

		assert.Nil(t, block)

		for i := 0; i < numBlocks; i++ {
			bw.WriteBlock(blocks[i], nil)
		}
		// 31 blocks have been written so far
	})

	t.Run("created blocks should always be over written blocks", func(t *testing.T) {
		/*
		 * Scenario:
		 *   We will create
		 *     1. a propose stack with 5 blocks over baseLastCreatedBlock, and
		 *     2. an alternate block over baseLastCreatedBlock.
		 *   We will write out this alternate block and verify that the subsequent block is created over this alternate block
		 *   and not on the existing propose stack.
		 * This scenario fails if WriteBlock does not flush the createdBlocks queue when the written block is divergent from the
		 * created blocks.
		 */

		baseLastCreatedBlock := bw.lastCreatedBlock

		// Create the stack of five blocks without writing them out
		blocks := []*cb.Block{}
		for i := 0; i < 5; i++ {
			blocks = append(blocks, bw.CreateNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}}))
		}

		// create and write out the alternate block
		alternateBlock := createBlockOverSpecifiedBlock(baseLastCreatedBlock, []*cb.Envelope{{Payload: []byte("alternate test envelope")}})
		bw.WriteBlock(alternateBlock, nil)

		// assert that CreateNextBlock creates the next block over this alternateBlock
		createdBlockOverAlternateBlock := bw.CreateNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}})
		synthesizedBlockOverAlternateBlock := createBlockOverSpecifiedBlock(alternateBlock, []*cb.Envelope{{Payload: []byte("test envelope")}})
		assert.True(t,
			bytes.Equal(createdBlockOverAlternateBlock.Header.Bytes(), synthesizedBlockOverAlternateBlock.Header.Bytes()),
			"created and synthesized blocks should be equal",
		)
		bw.WriteBlock(createdBlockOverAlternateBlock, nil)
		// 33 blocks have been written so far
	})

	// race between CreateNextBlock and mutex
	t.Run("allow parallel goroutines to call CreateNextBlock", func(t *testing.T) {
		/*
		 * NOTE: This is not something that we see being useful right now but we do get this for free,
		 *       and an additional test never hurts anyone.
		 * Scenario:
		 *   We launch multiple goroutines each of which calls CreateNextBlock.
		 *   They should all succeed, creating blocks in the right order.
		 * This scenario fails if CreateNextBlock does not acquire a mutex before accessing lastCreatedBlock.
		 */
		blocks := []*cb.Block{}
		wg := sync.WaitGroup{}
		mutex := sync.Mutex{} // for protecting access to the `blocks` slice
		numBlocks := 10

		wg.Add(numBlocks)
		for i := 0; i < numBlocks; i++ {
			go func() {
				b := bw.CreateNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}})
				mutex.Lock()
				blocks = append(blocks, b)
				mutex.Unlock()
				wg.Done()
			}()
		}

		wg.Wait()

		assert.Equal(t, numBlocks, len(blocks))

		// sort the `blocks` slice since the goroutines may acquire `mutex` in an order different
		// from `bw.creatingBlock` resulting in the blocks being appended to `blocks` in an order
		// different from the creation order.
		sort.Slice(blocks, func(i, j int) bool {
			return blocks[i].Header.Number < blocks[j].Header.Number
		})

		for i := 0; i < numBlocks-1; i++ {
			assert.Equal(t, blocks[i].Header.Number+1, blocks[i+1].Header.Number)
		}

		for i := 0; i < numBlocks; i++ {
			bw.WriteBlock(blocks[i], nil)
		}
	})

	t.Run("DiscardCreatedBlocks empties the createdBlocks stack", func(t *testing.T) {
		numBlocks := 10
		blocks := []*cb.Block{}

		for i := 0; i < numBlocks; i++ {
			blocks = append(blocks, bw.CreateNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}}))
		}

		bw.DiscardCreatedBlocks()

		assert.True(t,
			bytes.Equal(bw.lastWrittenBlock.Header.Bytes(), bw.lastCreatedBlock.Header.Bytes()),
			"created and synthesized blocks should be equal",
		)
		assert.Empty(t, bw.createdBlocks)
	})

	// race between CreateNextBlock and mutex
	t.Run("ensure parallel WriteBlock resets the createdBlocks queue", func(t *testing.T) {
		/*
		 * Scenario:
		 *   We launch multiple goroutines each of which calls CreateNextBlock after a random delay.
		 *   We also launch another goroutine to write an alternate block.
		 *   Idea is that the call to WriteBlock should interfere with the calls to CreateNextBlock causing
		 *   a reset of the createdBlocks queue exactly once. This is to test that concurrent setting
		 *   of the bw.lastCreatedBlock from WriteBlock is handled correctly.
		 * This scenario fails if WriteBlock does not acquire the creatingBlock lock before setting lastCreatedBlock,
		 * since the value it writes could be over-written by other CreateNextBlock goroutines.
		 */
		blocks := []*cb.Block{}
		wg := sync.WaitGroup{}
		mutex := sync.Mutex{} // for protecting access to the `blocks` slice
		numBlocks := 10

		rand.Seed(time.Now().UTC().UnixNano())

		baseLastCreatedBlock := bw.lastCreatedBlock
		alternateBlock := createBlockOverSpecifiedBlock(baseLastCreatedBlock, []*cb.Envelope{{Payload: []byte("alternate test envelope")}})

		// ensure at least one CreateBlock is called before WriteBlock
		blocks = append(blocks, bw.CreateNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}}))
		wg.Add(numBlocks - 1)
		// further calls to CreateNextBlock and one to WriteBlock hoping that their would be a race condition on their access
		// of bw.lastProposedBlock which ought to be protected in the implementation by a mutex
		for i := 0; i < numBlocks-2; i++ {
			go func() {
				// sleep for a random time
				time.Sleep(time.Duration(rand.Intn(10)*50) * time.Millisecond)
				b := bw.CreateNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}})

				mutex.Lock()
				blocks = append(blocks, b)
				mutex.Unlock()
				wg.Done()
			}()
		}
		go func() {
			bw.WriteBlock(alternateBlock, nil)
			wg.Done()
		}()
		wg.Wait()
		// ensure at least one CreateNextBlock is called after WriteBlock
		blocks = append(blocks, bw.CreateNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}}))

		// sort the `blocks` slice since the goroutines may acquire `mutex` in an order different
		// from `bw.creatingBlock` resulting in the blocks being appended to `blocks` in an order
		// different from the creation order.
		sort.Slice(blocks, func(i, j int) bool {
			return blocks[i].Header.Number < blocks[j].Header.Number
		})

		// Since we create atleast one block in the to-be-discarded create stack, the first block
		// after sorting in `blocks` must be at the same height as `alternateBlock`, and there can
		// be only one such block.
		alternateBlockNumber := alternateBlock.Header.Number
		alternateBlockHash := alternateBlock.Header.Hash()
		assert.Equal(t, alternateBlockNumber, blocks[0].Header.Number)

		// There can be atleast 1 and atmost 2 blocks at height `atlernateBlock.Height + 1`,
		// one optionally before the writing of the `alternateBlock` and one compulsarily
		// after the writing of `alternateBlock`. We have to verify the existence of this
		// second block.
		found := false
		assert.Equal(t, alternateBlockNumber+1, blocks[1].Header.Number)
		if bytes.Equal(alternateBlockHash, blocks[1].Header.PreviousHash) {
			found = true
		} else {
			if alternateBlockNumber+1 == blocks[2].Header.Number &&
				bytes.Equal(alternateBlockHash, blocks[2].Header.PreviousHash) {
				found = true
			}
		}
		assert.True(t, found)

		bw.DiscardCreatedBlocks()
	})

}

func createBlockOverSpecifiedBlock(baseBlock *cb.Block, messages []*cb.Envelope) *cb.Block {
	previousBlockHash := baseBlock.Header.Hash()

	data := &cb.BlockData{
		Data: make([][]byte, len(messages)),
	}

	var err error
	for i, msg := range messages {
		data.Data[i], err = proto.Marshal(msg)
		if err != nil {
			logger.Panicf("Could not marshal envelope: %s", err)
		}
	}

	block := cb.NewBlock(baseBlock.Header.Number+1, previousBlockHash)
	block.Header.DataHash = data.Hash()
	block.Data = data

	return block
}
