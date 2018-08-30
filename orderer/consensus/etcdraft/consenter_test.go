package etcdraft_test

import (
	"github.com/hyperledger/fabric/protos/common"
	"time"

	mockconfig "github.com/hyperledger/fabric/common/mocks/config"
	"github.com/hyperledger/fabric/orderer/common/localconfig"
	etcdraft "github.com/hyperledger/fabric/orderer/consensus/etcdraft"
	consensusmocks "github.com/hyperledger/fabric/orderer/consensus/mocks"
	mockblockcutter "github.com/hyperledger/fabric/orderer/mocks/common/blockcutter"
	etcdraftproto "github.com/hyperledger/fabric/protos/orderer/etcdraft"
	"github.com/hyperledger/fabric/protos/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Consenter", func() {
	var (
		env     *common.Envelope
		support *consensusmocks.FakeConsenterSupport
		cutter  *mockblockcutter.Receiver
	)

	BeforeEach(func() {
		env = &common.Envelope{
			Payload: utils.MarshalOrPanic(&common.Payload{
				Header: &common.Header{ChannelHeader: utils.MarshalOrPanic(&common.ChannelHeader{Type: int32(common.HeaderType_MESSAGE), ChannelId: "foo"})},
				Data:   []byte("TEST_MESSAGE"),
			}),
		}

		support = &consensusmocks.FakeConsenterSupport{}

		m := &etcdraftproto.Metadata{
			Consenters: []*etcdraftproto.Consenter{
				{ServerTlsCert: []byte("cert.orderer0.org0")},
			},
		}
		metadata := utils.MarshalOrPanic(m)
		support.SharedConfigReturns(&mockconfig.Orderer{ConsensusMetadataVal: metadata})

		cutter = mockblockcutter.NewReceiver()
		support.BlockCutterReturns(cutter)
	})

	It("successfully constructs a Chain", func() {
		consenter := etcdraft.Consenter{
			Config: localconfig.EtcdRaft{
				TickInterval:    100 * time.Millisecond,
				ElectionTick:    2,
				HeartbeatTick:   1,
				MaxSizePerMsg:   1024 * 1024,
				MaxInflightMsgs: 256,
			},
			Cert: []byte("cert.orderer0.org0"),
		}

		chain, err := consenter.HandleChain(support, nil)
		Expect(chain).NotTo(BeNil())
		Expect(err).NotTo(HaveOccurred())

		chain.Start()
		defer chain.Halt()

		Eventually(func() error {
			return chain.Order(env, 0)
		}).ShouldNot(HaveOccurred())
	})
})
