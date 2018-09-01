/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package etcdraft_test

import (
	"encoding/pem"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"github.com/coreos/etcd/raft"
	"github.com/hyperledger/fabric/common/crypto/tlsgen"
	"github.com/hyperledger/fabric/common/flogging"
	mockconfig "github.com/hyperledger/fabric/common/mocks/config"
	"github.com/hyperledger/fabric/orderer/common/cluster"
	"github.com/hyperledger/fabric/orderer/consensus/etcdraft"
	"github.com/hyperledger/fabric/orderer/consensus/etcdraft/mocks"
	consensusmocks "github.com/hyperledger/fabric/orderer/consensus/mocks"
	mockblockcutter "github.com/hyperledger/fabric/orderer/mocks/common/blockcutter"
	"github.com/hyperledger/fabric/protos/common"
	raftprotos "github.com/hyperledger/fabric/protos/orderer/etcdraft"
	"github.com/hyperledger/fabric/protos/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

var _ = Describe("Chain", func() {
	var (
		m           *common.Envelope
		normalBlock *common.Block
		interval    time.Duration
		channelID   string
		tlsCA       tlsgen.CA
	)

	BeforeEach(func() {
		tlsCA, _ = tlsgen.NewCA()
		channelID = "test-chain"
		m = &common.Envelope{
			Payload: utils.MarshalOrPanic(&common.Payload{
				Header: &common.Header{ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{Type: int32(common.HeaderType_MESSAGE), ChannelId: channelID})},
				Data:   []byte("TEST_MESSAGE"),
			}),
		}
		normalBlock = &common.Block{Data: &common.BlockData{Data: [][]byte{[]byte("foo")}}}
		interval = time.Second
	})

	Describe("Single raft node", func() {
		var (
			commConfigurator  *mocks.Configurator
			consenterMetadata *raftprotos.Metadata
			clock             *fakeclock.FakeClock
			opt               etcdraft.Options
			support           *consensusmocks.FakeConsenterSupport
			cutter            *mockblockcutter.Receiver
			storage           *raft.MemoryStorage
			observe           chan uint64
			chain             *etcdraft.Chain
			logger            *flogging.FabricLogger
		)

		campaign := func() {
			clock.Increment(interval)
			Consistently(observe).ShouldNot(Receive())

			clock.Increment(interval)
			// raft election timeout is randomized in
			// [electiontimeout, 2 * electiontimeout - 1]
			// So we may need one extra tick to trigger
			// leader election.
			clock.Increment(interval)
			Eventually(observe).Should(Receive())
		}

		BeforeEach(func() {
			commConfigurator = &mocks.Configurator{}
			commConfigurator.On("Configure", mock.Anything, mock.Anything)
			clock = fakeclock.NewFakeClock(time.Now())
			storage = raft.NewMemoryStorage()
			logger = flogging.NewFabricLogger(zap.NewNop())
			observe = make(chan uint64, 1)
			opt = etcdraft.Options{
				RaftID:          uint64(1),
				Clock:           clock,
				TickInterval:    interval,
				ElectionTick:    2,
				HeartbeatTick:   1,
				MaxSizePerMsg:   1024 * 1024,
				MaxInflightMsgs: 256,
				Peers:           []raft.Peer{{ID: uint64(1)}},
				Logger:          logger,
				Storage:         storage,
			}
			support = &consensusmocks.FakeConsenterSupport{}
			support.ChainIDReturns(channelID)
			consenterMetadata = createMetadata(3, tlsCA)
			support.SharedConfigReturns(&mockconfig.Orderer{
				BatchTimeoutVal:      time.Hour,
				ConsensusMetadataVal: utils.MarshalOrPanic(consenterMetadata),
			})
			cutter = mockblockcutter.NewReceiver()
			support.BlockCutterReturns(cutter)

			var err error
			chain, err = etcdraft.NewChain(support, opt, observe, commConfigurator)
			Expect(err).NotTo(HaveOccurred())
			chain.Start()

			// When raft node bootstraps, it produces a ConfChange
			// to add itself, which needs to be consumed with Ready().
			// If there are pending configuration changes in raft,
			// it refused to campaign, no matter how many ticks supplied.
			// This is not a problem in production code because eventually
			// raft.Ready will be consumed as real time goes by.
			//
			// However, this is problematic when using fake clock and artificial
			// ticks. Instead of ticking raft indefinitely until raft.Ready is
			// consumed, this check is added to indirectly guarantee
			// that first ConfChange is actually consumed and we can safely
			// proceed to tick raft.
			Eventually(func() error {
				_, err := storage.Entries(1, 1, 1)
				return err
			}).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			chain.Halt()
		})

		Context("when a node starts up", func() {
			It("properly configures the communication layer", func() {
				expectedNodeConfig := nodeConfigFromMetadata(consenterMetadata)
				commConfigurator.AssertCalled(testingInstance, "Configure", channelID, expectedNodeConfig)
			})
		})

		Context("when no raft leader is elected", func() {
			It("fails to order envelope", func() {
				err := chain.Order(m, uint64(0))
				Expect(err).To(MatchError("no raft leader"))
			})
		})

		Context("when raft leader is elected", func() {
			BeforeEach(func() {
				campaign()
			})

			It("fails to order envelope if chain is halted", func() {
				chain.Halt()
				err := chain.Order(m, uint64(0))
				Expect(err).To(MatchError("chain is stopped"))
			})

			It("produces blocks following batch rules", func() {
				close(cutter.Block)
				support.CreateNextBlockReturns(normalBlock)

				By("cutting next batch directly")
				cutter.CutNext = true
				err := chain.Order(m, uint64(0))
				Expect(err).NotTo(HaveOccurred())
				Eventually(support.WriteBlockCallCount).Should(Equal(1))

				By("respecting batch timeout")
				cutter.CutNext = false
				timeout := time.Second
				support.SharedConfigReturns(&mockconfig.Orderer{BatchTimeoutVal: timeout})
				err = chain.Order(m, uint64(0))
				Expect(err).NotTo(HaveOccurred())

				clock.WaitForNWatchersAndIncrement(timeout, 2)
				Eventually(support.WriteBlockCallCount).Should(Equal(2))
			})

			It("does not reset timer for every envelope", func() {
				close(cutter.Block)
				support.CreateNextBlockReturns(normalBlock)

				timeout := time.Second
				support.SharedConfigReturns(&mockconfig.Orderer{BatchTimeoutVal: timeout})

				err := chain.Order(m, uint64(0))
				Expect(err).NotTo(HaveOccurred())
				Eventually(func() int {
					return len(cutter.CurBatch)
				}).Should(Equal(1))

				clock.WaitForNWatchersAndIncrement(timeout/2, 2)

				err = chain.Order(m, uint64(0))
				Expect(err).NotTo(HaveOccurred())
				Eventually(func() int {
					return len(cutter.CurBatch)
				}).Should(Equal(2))

				// second envelope should not reset timer,
				// so that it expires if we increment by another timeout/2
				clock.Increment(timeout / 2)
				Eventually(support.WriteBlockCallCount).Should(Equal(1))
			})

			It("does not write block if halt before timeout", func() {
				close(cutter.Block)
				timeout := time.Second
				support.SharedConfigReturns(&mockconfig.Orderer{BatchTimeoutVal: timeout})

				err := chain.Order(m, uint64(0))
				Expect(err).NotTo(HaveOccurred())
				Eventually(func() int {
					return len(cutter.CurBatch)
				}).Should(Equal(1))

				// wait for timer to start
				Eventually(clock.WatcherCount).Should(Equal(2))

				chain.Halt()
				Consistently(support.WriteBlockCallCount).Should(Equal(0))
			})

			It("stops timer if a batch is cut", func() {
				close(cutter.Block)
				support.CreateNextBlockReturns(normalBlock)

				timeout := time.Second
				support.SharedConfigReturns(&mockconfig.Orderer{BatchTimeoutVal: timeout})

				err := chain.Order(m, uint64(0))
				Expect(err).NotTo(HaveOccurred())
				Eventually(func() int {
					return len(cutter.CurBatch)
				}).Should(Equal(1))

				clock.WaitForNWatchersAndIncrement(timeout/2, 2)

				By("force a batch to be cut before timer expires")
				cutter.CutNext = true
				err = chain.Order(m, uint64(0))
				Expect(err).NotTo(HaveOccurred())
				Eventually(support.WriteBlockCallCount).Should(Equal(1))
				Expect(support.CreateNextBlockArgsForCall(0)).To(HaveLen(2))
				Expect(cutter.CurBatch).To(HaveLen(0))

				// this should start a fresh timer
				cutter.CutNext = false
				err = chain.Order(m, uint64(0))
				Expect(err).NotTo(HaveOccurred())
				Eventually(func() int {
					return len(cutter.CurBatch)
				}).Should(Equal(1))

				clock.WaitForNWatchersAndIncrement(timeout/2, 2)
				Consistently(support.WriteBlockCallCount).Should(Equal(1))

				clock.Increment(timeout / 2)
				Eventually(support.WriteBlockCallCount).Should(Equal(2))
				Expect(support.CreateNextBlockArgsForCall(1)).To(HaveLen(1))
			})

			It("cut two batches if new envelope does not fit into current one", func() {
				close(cutter.Block)
				support.CreateNextBlockReturns(normalBlock)

				timeout := time.Second
				support.SharedConfigReturns(&mockconfig.Orderer{BatchTimeoutVal: timeout})

				err := chain.Order(m, uint64(0))
				Expect(err).NotTo(HaveOccurred())
				Eventually(func() int {
					return len(cutter.CurBatch)
				}).Should(Equal(1))

				cutter.IsolatedTx = true
				err = chain.Order(m, uint64(0))
				Expect(err).NotTo(HaveOccurred())

				Eventually(support.CreateNextBlockCallCount).Should(Equal(2))
				Eventually(support.WriteBlockCallCount).Should(Equal(2))
			})

			Context("revalidation", func() {
				It("enque if an envelope is still valid", func() {
					close(cutter.Block)
					support.CreateNextBlockReturns(normalBlock)

					timeout := time.Hour
					support.SharedConfigReturns(&mockconfig.Orderer{BatchTimeoutVal: timeout})
					support.ProcessNormalMsgReturns(1, nil)

					support.SequenceReturns(1)

					err := chain.Order(m, uint64(0))
					Expect(err).NotTo(HaveOccurred())
					Eventually(func() int {
						return len(cutter.CurBatch)
					}).Should(Equal(1))

					err = chain.Order(m, uint64(0))
					Expect(err).NotTo(HaveOccurred())
					Eventually(func() int {
						return len(cutter.CurBatch)
					}).Should(Equal(2))
				})

				It("does not enque if an envelope is not valid", func() {
					close(cutter.Block)
					support.CreateNextBlockReturns(normalBlock)

					timeout := time.Hour
					support.SharedConfigReturns(&mockconfig.Orderer{BatchTimeoutVal: timeout})
					support.ProcessNormalMsgReturns(1, errors.Errorf("Envelope is invalid"))

					support.SequenceReturns(1)

					err := chain.Order(m, uint64(0))
					Expect(err).NotTo(HaveOccurred())
					Eventually(func() int {
						return len(cutter.CurBatch)
					}).Should(Equal(1))

					err = chain.Order(m, uint64(0))
					Expect(err).NotTo(HaveOccurred())
					Consistently(func() int {
						return len(cutter.CurBatch)
					}).Should(Equal(1))
				})
			})

			It("unblocks Errored if chain is halted", func() {
				Expect(chain.Errored()).NotTo(Receive())
				chain.Halt()
				Expect(chain.Errored()).Should(BeClosed())
			})

			It("config message is not yet supported", func() {
				c := &common.Envelope{
					Payload: utils.MarshalOrPanic(&common.Payload{
						Header: &common.Header{ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{Type: int32(common.HeaderType_CONFIG), ChannelId: channelID})},
						Data:   []byte("TEST_MESSAGE"),
					}),
				}

				Expect(func() {
					chain.Configure(c, uint64(0))
				}).To(Panic())
			})
		})
	})
})

func nodeConfigFromMetadata(consenterMetadata *raftprotos.Metadata) []cluster.RemoteNode {
	var nodes []cluster.RemoteNode
	for i, consenter := range consenterMetadata.Consenters {
		// For now, skip ourselves
		if i == 0 {
			continue
		}
		serverDER, _ := pem.Decode(consenter.ServerTlsCert)
		clientDER, _ := pem.Decode(consenter.ClientTlsCert)
		node := cluster.RemoteNode{
			ID:            uint64(i + 1),
			Endpoint:      "localhost:7050",
			ServerTLSCert: serverDER.Bytes,
			ClientTLSCert: clientDER.Bytes,
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func createMetadata(nodeCount int, tlsCA tlsgen.CA) *raftprotos.Metadata {
	md := &raftprotos.Metadata{}
	for i := 0; i < nodeCount; i++ {
		md.Consenters = append(md.Consenters, &raftprotos.Consenter{
			Host:          "localhost",
			Port:          7050,
			ServerTlsCert: serverTLSCert(tlsCA),
			ClientTlsCert: clientTLSCert(tlsCA),
		})
	}
	return md
}

func serverTLSCert(tlsCA tlsgen.CA) []byte {
	cert, err := tlsCA.NewServerCertKeyPair("localhost")
	if err != nil {
		panic(err)
	}
	return cert.Cert
}

func clientTLSCert(tlsCA tlsgen.CA) []byte {
	cert, err := tlsCA.NewClientCertKeyPair()
	if err != nil {
		panic(err)
	}
	return cert.Cert
}
