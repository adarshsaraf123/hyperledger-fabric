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
	"sync"
	"testing"

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
	bw.committingBlock.Lock()
	bw.committingBlock.Unlock()

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
		createdBlocks: make(chan *cb.Block, 10),
	}

	ctx := makeConfigTx(genesisconfig.TestChainID, 1)
	block := cb.NewBlock(1, genesisBlock.Header.Hash())
	block.Data.Data = [][]byte{utils.MarshalOrPanic(ctx)}
	consenterMetadata := []byte("foo")
	bw.WriteConfigBlock(block, consenterMetadata)

	// Wait for the commit to complete
	bw.committingBlock.Lock()
	bw.committingBlock.Unlock()

	t.Run("propose multiple blocks without having to write them out; decouples CreateNextBlock from WriteBlock ", func(t *testing.T) {
		// Scenario:
		//   We create five blocks initially and then write only two of them. We further create five more blocks
		//   and write out the remaining 8 blocks in the propose stack. This should succeed since the written
		//   blocks are not divergent from the created blocks.
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
		})
	})

	t.Run("created blocks should always be over written blocks", func(t *testing.T) {
		// Scenario:
		//   We will create
		//     1. a propose stack with 5 blocks over baseLastCreatedBlock, and
		//     2. an alternate block over baseLastCreatedBlock.
		//   We will write out this alternate block and verify that the subsequent block is created over this alternate block
		//   and not on the existing propose stack.

		baseLastCreatedBlock := bw.lastCreatedBlock

		// Create the stack of five blocks without writing them out
		chain := []*cb.Block{}
		for i := 0; i < 5; i++ {
			chain = append(chain, bw.CreateNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}}))
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
	})

	t.Run("allow parallel goroutines to call CreateNextBlock", func(t *testing.T) {
		blocks := []*cb.Block{}
		wg := sync.WaitGroup{}

		wg.Add(10)
		for i := 0; i < 10; i++ {
			go func() {
				blocks = append(blocks, bw.CreateNextBlock([]*cb.Envelope{{Payload: []byte("test envelope")}}))
				wg.Done()
			}()
		}

		wg.Wait()
		for i := 0; i < 9; i++ {
			assert.Equal(t, blocks[i].Header.Number+1, blocks[i+1].Header.Number)
		}

		for i := 0; i < 10; i++ {
			bw.WriteBlock(blocks[i], nil)
		}
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
