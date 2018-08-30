/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package etcdraft_test

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	mockconfig "github.com/hyperledger/fabric/common/mocks/config"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	etcdraft "github.com/hyperledger/fabric/orderer/consensus/etcdraft"
	consensusmocks "github.com/hyperledger/fabric/orderer/consensus/mocks"
	mockblockcutter "github.com/hyperledger/fabric/orderer/mocks/common/blockcutter"
	"github.com/hyperledger/fabric/protos/common"
	etcdraftproto "github.com/hyperledger/fabric/protos/orderer/etcdraft"
	"github.com/hyperledger/fabric/protos/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Consenter", func() {
	var (
		env     *common.Envelope
		support *consensusmocks.FakeConsenterSupport
		cutter  *mockblockcutter.Receiver
		conf    *localconfig.TopLevel
		cert    *os.File
		logger  *flogging.FabricLogger
	)

	BeforeEach(func() {
		env = &common.Envelope{
			Payload: utils.MarshalOrPanic(&common.Payload{
				Header: &common.Header{ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{Type: int32(common.HeaderType_MESSAGE), ChannelId: "foo"})},
				Data:   []byte("TEST_MESSAGE"),
			}),
		}

		logger = flogging.NewFabricLogger(zap.NewNop())

		var err error
		cert, err = ioutil.TempFile("", "cert-")
		Expect(err).NotTo(HaveOccurred())

		support = &consensusmocks.FakeConsenterSupport{}
		conf = &localconfig.TopLevel{
			General: localconfig.General{
				TLS: localconfig.TLS{
					Certificate: cert.Name(),
				},
			},
		}
	})

	AfterEach(func() {
		err := os.Remove(cert.Name())
		Expect(err).NotTo(HaveOccurred())
	})

	It("successfully constructs a Chain", func() {
		certBytes := []byte("cert.orderer0.org0")
		m := &etcdraftproto.Metadata{
			Consenters: []*etcdraftproto.Consenter{
				{ServerTlsCert: certBytes},
			},
		}
		metadata := utils.MarshalOrPanic(m)
		support.SharedConfigReturns(&mockconfig.Orderer{ConsensusMetadataVal: metadata})
		support.CreateNextBlockReturns(&common.Block{Data: &common.BlockData{Data: [][]byte{[]byte("foo")}}})

		cutter = mockblockcutter.NewReceiver()
		cutter.CutNext = true
		close(cutter.Block)
		support.BlockCutterReturns(cutter)

		_, err := cert.Write(certBytes)
		Expect(err).NotTo(HaveOccurred())

		consenter := etcdraft.New(conf, logger)
		Expect(consenter).NotTo(BeNil())

		chain, err := consenter.HandleChain(support, nil)
		Expect(chain).NotTo(BeNil())
		Expect(err).NotTo(HaveOccurred())

		chain.Start()
		defer chain.Halt()

		// election timeout is hardcoded to 10 * TickInterval,
		// we need to wait at least for 2 * timeout to ensure
		// a leader is elected.
		// TODO consider making clock an injected dependency
		// or timeout configurable to reduce wait time.
		Eventually(func() error {
			return chain.Order(env, 0)
		}, 2*time.Second).ShouldNot(HaveOccurred())

		Eventually(support.WriteBlockCallCount).Should(Equal(1))
	})

	It("fails to handle chain if no matching cert found", func() {
		m := &etcdraftproto.Metadata{
			Consenters: []*etcdraftproto.Consenter{
				{ServerTlsCert: []byte("cert.orderer1.org1")},
			},
		}
		metadata := utils.MarshalOrPanic(m)
		support.SharedConfigReturns(&mockconfig.Orderer{ConsensusMetadataVal: metadata})

		_, err := cert.Write([]byte("cert.orderer0.org0"))
		Expect(err).NotTo(HaveOccurred())

		consenter := etcdraft.New(conf, logger)
		Expect(consenter).NotTo(BeNil())

		chain, err := consenter.HandleChain(support, nil)
		Expect(chain).To(BeNil())
		Expect(err).To(MatchError("failed to detect own raft ID because no matching certificate found"))
	})

	Context("Multi-node raft cluster", func() {
		It("handles new chain with no problem", func() {
			m := &etcdraftproto.Metadata{
				Consenters: []*etcdraftproto.Consenter{
					{ServerTlsCert: []byte("cert.orderer0.org0")},
					{ServerTlsCert: []byte("cert.orderer1.org1")},
					{ServerTlsCert: []byte("cert.orderer2.org2")},
				},
			}
			metadata := utils.MarshalOrPanic(m)
			support.SharedConfigReturns(&mockconfig.Orderer{ConsensusMetadataVal: metadata})

			_, err := cert.Write([]byte("cert.orderer0.org0"))
			Expect(err).NotTo(HaveOccurred())

			consenter := etcdraft.New(conf, logger)
			Expect(consenter).NotTo(BeNil())

			chain, err := consenter.HandleChain(support, nil)
			Expect(chain).NotTo(BeNil())
			Expect(err).NotTo(HaveOccurred())

			chain.Start()
			defer chain.Halt()

			Consistently(func() error {
				return chain.Order(env, 0)
			}).Should(MatchError("no raft leader"))
		})
	})
})
