package auction_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/cloudfoundry/gunk/natsrunner"
	"github.com/cloudfoundry/storeadapter/storerunner/etcdstorerunner"
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/lossyrep"
	"github.com/onsi/auction/nats/repnatsclient"
	"github.com/onsi/auction/rabbit/reprabbitclient"
	"github.com/onsi/auction/representative"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
	"github.com/onsi/auction/visualization"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"testing"
	"time"
)

const InProcess = "inprocess"
const NATS = "nats"
const Rabbit = "rabbit"
const RemoteAuction = "remote"

// knobs
var communicationMode string
var auctioneerMode string

var rules types.AuctionRules
var timeout time.Duration

var numAuctioneers = 10
var numReps = 100
var repResources = 100

var svgReport *visualization.SVGReport
var reportName string
var reports []*types.Report

// plumbing
var sessionsToTerminate []*gexec.Session
var natsPort int
var natsRunner *natsrunner.NATSRunner
var etcdPort int
var etcdRunner *etcdstorerunner.ETCDClusterRunner
var client types.TestRepPoolClient
var guids []string
var communicator types.AuctionCommunicator
var rabbitSession *gexec.Session

func init() {
	flag.StringVar(&communicationMode, "communicationMode", "inprocess", "one of inprocess, nats, rabbit")
	flag.StringVar(&auctioneerMode, "auctioneerMode", "inprocess", "one of inprocess, remote")

	flag.StringVar(&(auctioneer.DefaultRules.Algorithm), "algorithm", auctioneer.DefaultRules.Algorithm, "the auction algorithm to use")
	flag.IntVar(&(auctioneer.DefaultRules.MaxRounds), "maxRounds", auctioneer.DefaultRules.MaxRounds, "the maximum number of rounds per auction")
	flag.IntVar(&(auctioneer.DefaultRules.MaxBiddingPool), "maxBiddingPool", auctioneer.DefaultRules.MaxBiddingPool, "the maximum number of participants in the pool")
	flag.IntVar(&(auctioneer.DefaultRules.MaxConcurrent), "maxConcurrent", auctioneer.DefaultRules.MaxConcurrent, "the maximum number of concurrent auctions to run")
}

func TestAuction(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auction Suite")
}

var _ = BeforeSuite(func() {
	reportName = fmt.Sprintf("./runs/%s_%s_pool%d_conc%d.svg", auctioneer.DefaultRules.Algorithm, communicationMode, auctioneer.DefaultRules.MaxBiddingPool, auctioneer.DefaultRules.MaxConcurrent)
	svgReport = visualization.StartSVGReport(reportName, 2, 3)
	prettyCommunicationMode := map[string]string{"inprocess": "In-Process", "nats": "NATS"}
	svgReport.DrawHeader(prettyCommunicationMode[communicationMode], auctioneer.DefaultRules)

	fmt.Printf("Running in %s communicationMode\n", communicationMode)
	fmt.Printf("Running in %s auctioneerMode\n", auctioneerMode)

	if auctioneerMode == RemoteAuction && communicationMode != NATS {
		panic("to use remote auctioneers, you must communicate via nats")
	}

	//parse flags to set up rules
	timeout = 500 * time.Millisecond

	natsPort = 5222 + GinkgoParallelNode()
	etcdPort = 4001 + GinkgoParallelNode()

	natsRunner = natsrunner.NewNATSRunner(natsPort)
	natsRunner.Start()

	etcdRunner = etcdstorerunner.NewETCDClusterRunner(etcdPort, 1)
	etcdRunner.Start()

	rules = auctioneer.DefaultRules

	sessionsToTerminate = []*gexec.Session{}

	var err error
	rabbitSession, err = gexec.Start(exec.Command("rabbitmq-server"), GinkgoWriter, GinkgoWriter)
	Ω(err).ShouldNot(HaveOccurred())
	Eventually(rabbitSession, 2).Should(gbytes.Say("Starting broker... completed"))

	client, guids = buildClient(numReps, repResources)

	if auctioneerMode == InProcess {
		communicator = func(auctionRequest types.AuctionRequest) types.AuctionResult {
			return auctioneer.Auction(etcdRunner.Adapter(), client, auctionRequest)
		}
	} else if auctioneerMode == RemoteAuction {
		auctioneerHosts := startAuctioneers(numAuctioneers)
		remotAuctionRouter := auctioneer.NewHTTPRemoteAuctions(auctioneerHosts)
		communicator = remotAuctionRouter.RemoteAuction
	} else {
		panic("wat?")
	}
})

var _ = BeforeEach(func() {
	etcdRunner.Stop()
	etcdRunner.Start()

	for _, guid := range guids {
		client.Reset(guid)
	}

	util.ResetGuids()
})

var _ = AfterSuite(func() {
	svgReport.Done()
	// exec.Command("open", "-a", "safari", reportName).Run()

	reportJSONName := fmt.Sprintf("./runs/%s_%s_pool%d_conc%d.json", auctioneer.DefaultRules.Algorithm, communicationMode, auctioneer.DefaultRules.MaxBiddingPool, auctioneer.DefaultRules.MaxConcurrent)
	data, err := json.Marshal(reports)
	Ω(err).ShouldNot(HaveOccurred())
	ioutil.WriteFile(reportJSONName, data, 0777)

	for _, sess := range sessionsToTerminate {
		sess.Kill().Wait()
	}

	natsRunner.Stop()
	etcdRunner.Stop()
	rabbitSession.Kill().Wait()
})

func startAuctioneers(numAuctioneers int) []string {
	auctioneerNodeBinary, err := gexec.Build("github.com/onsi/auction/auctioneernode")
	Ω(err).ShouldNot(HaveOccurred())

	auctioneerHosts := []string{}
	for i := 0; i < numAuctioneers; i++ {
		port := 48710 + i
		auctioneerCmd := exec.Command(
			auctioneerNodeBinary,
			"-natsAddrs", fmt.Sprintf("127.0.0.1:%d", natsPort),
			"-timeout", fmt.Sprintf("%s", timeout),
			"-httpAddr", fmt.Sprintf("127.0.0.1:%d", port),
		)
		auctioneerHosts = append(auctioneerHosts, fmt.Sprintf("127.0.0.1:%d", port))

		sess, err := gexec.Start(auctioneerCmd, GinkgoWriter, GinkgoWriter)
		Ω(err).ShouldNot(HaveOccurred())
		Eventually(sess).Should(gbytes.Say("auctioneering"))
		sessionsToTerminate = append(sessionsToTerminate, sess)
	}
	return auctioneerHosts
}

func buildClient(numReps int, repResources int) (types.TestRepPoolClient, []string) {
	repNodeBinary, err := gexec.Build("github.com/onsi/auction/repnode")
	Ω(err).ShouldNot(HaveOccurred())

	if communicationMode == InProcess {
		lossyrep.LatencyMin = 2 * time.Millisecond
		lossyrep.LatencyMax = 12 * time.Millisecond
		lossyrep.Timeout = 50 * time.Millisecond
		lossyrep.Flakiness = 0.95

		guids := []string{}
		repMap := map[string]*representative.Representative{}

		for i := 0; i < numReps; i++ {
			guid := util.NewGuid("REP")
			guids = append(guids, guid)
			repMap[guid] = representative.New(etcdRunner.Adapter(), guid, repResources)
		}

		client := lossyrep.New(repMap, map[string]bool{})
		return client, guids
	} else if communicationMode == NATS {
		guids := []string{}

		for i := 0; i < numReps; i++ {
			guid := util.NewGuid("REP")

			serverCmd := exec.Command(
				repNodeBinary,
				"-guid", guid,
				"-natsAddrs", fmt.Sprintf("127.0.0.1:%d", natsPort),
				"-etcdCluster", strings.Join(etcdRunner.NodeURLS(), ","),
				"-resources", fmt.Sprintf("%d", repResources),
			)

			sess, err := gexec.Start(serverCmd, GinkgoWriter, GinkgoWriter)
			Ω(err).ShouldNot(HaveOccurred())
			Eventually(sess).Should(gbytes.Say("listening"))
			sessionsToTerminate = append(sessionsToTerminate, sess)

			guids = append(guids, guid)
		}

		client := repnatsclient.New(natsRunner.MessageBus, timeout)

		return client, guids
	} else if communicationMode == Rabbit {
		guids := []string{}

		for i := 0; i < numReps; i++ {
			guid := util.NewGuid("REP")

			serverCmd := exec.Command(
				repNodeBinary,
				"-guid", guid,
				"-rabbitAddr", "amqp://127.0.0.1",
				"-etcdCluster", strings.Join(etcdRunner.NodeURLS(), ","),
				"-resources", fmt.Sprintf("%d", repResources),
			)

			sess, err := gexec.Start(serverCmd, GinkgoWriter, GinkgoWriter)
			Ω(err).ShouldNot(HaveOccurred())
			Eventually(sess).Should(gbytes.Say("listening"))
			sessionsToTerminate = append(sessionsToTerminate, sess)

			guids = append(guids, guid)
		}

		client := reprabbitclient.New("amqp://127.0.0.1", timeout)

		return client, guids
	}

	panic("wat!")
}
