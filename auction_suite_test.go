package auction_test

import (
	"flag"
	"fmt"
	"os/exec"

	"github.com/cloudfoundry/gunk/natsrunner"
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/http/rephttpclient"
	"github.com/onsi/auction/lossyrep"
	"github.com/onsi/auction/nats/repnatsclient"
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
const HTTP = "http"
const NATS = "nats"
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

// plumbing
var sessionsToTerminate []*gexec.Session
var natsPort int
var natsRunner *natsrunner.NATSRunner
var client types.TestRepPoolClient
var guids []string
var communicator types.AuctionCommunicator

func init() {
	flag.StringVar(&communicationMode, "communicationMode", "inprocess", "one of inprocess, http, nats")
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
	reportName = fmt.Sprintf("./%s_%s_pool%d_conc%d.svg", auctioneer.DefaultRules.Algorithm, communicationMode, auctioneer.DefaultRules.MaxBiddingPool, auctioneer.DefaultRules.MaxConcurrent)
	svgReport = visualization.StartSVGReport(reportName, 2, 3)
	svgReport.DrawHeader(auctioneer.DefaultRules)

	fmt.Printf("Running in %s communicationMode\n", communicationMode)
	fmt.Printf("Running in %s auctioneerMode\n", auctioneerMode)

	if auctioneerMode == RemoteAuction && communicationMode != NATS {
		panic("to use remote auctioneers, you must communicate via nats")
	}

	//parse flags to set up rules
	timeout = 500 * time.Millisecond
	natsPort = 5222 + GinkgoParallelNode()

	natsRunner = natsrunner.NewNATSRunner(natsPort)

	rules = auctioneer.DefaultRules

	sessionsToTerminate = []*gexec.Session{}

	natsRunner.Start()
	client, guids = buildClient(numReps, repResources)

	if auctioneerMode == InProcess {
		communicator = func(auctionRequest types.AuctionRequest) types.AuctionResult {
			return auctioneer.Auction(client, auctionRequest)
		}
	} else if auctioneerMode == RemoteAuction {
		startAuctioneers(numAuctioneers)
		communicator = func(auctionRequest types.AuctionRequest) types.AuctionResult {
			return auctioneer.RemoteAuction(natsRunner.MessageBus, auctionRequest)
		}
	} else {
		panic("wat?")
	}
})

var _ = BeforeEach(func() {
	for _, guid := range guids {
		client.Reset(guid)
	}

	util.ResetGuids()
})

var _ = AfterSuite(func() {
	svgReport.Done()
	exec.Command("open", "-a", "safari", reportName).Run()

	for _, sess := range sessionsToTerminate {
		sess.Kill().Wait()
	}

	natsRunner.Stop()
})

func startAuctioneers(numAuctioneers int) {
	auctioneerNodeBinary, err := gexec.Build("github.com/onsi/auction/auctioneernode")
	Ω(err).ShouldNot(HaveOccurred())

	for i := 0; i < numAuctioneers; i++ {
		auctioneerCmd := exec.Command(
			auctioneerNodeBinary,
			"-natsAddr", fmt.Sprintf("127.0.0.1:%d", natsPort),
			"-timeout", fmt.Sprintf("%s", timeout),
		)

		sess, err := gexec.Start(auctioneerCmd, GinkgoWriter, GinkgoWriter)
		Ω(err).ShouldNot(HaveOccurred())
		Eventually(sess).Should(gbytes.Say("auctioneering"))
		sessionsToTerminate = append(sessionsToTerminate, sess)
	}
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
			repMap[guid] = representative.New(guid, repResources)
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
	} else if communicationMode == HTTP {
		startPort := 18000 + (numReps * GinkgoParallelNode())
		guids := []string{}

		repMap := map[string]string{}

		for i := 0; i < numReps; i++ {
			guid := util.NewGuid("REP")
			port := startPort + i

			serverCmd := exec.Command(
				repNodeBinary,
				"-guid", guid,
				"-httpAddr", fmt.Sprintf("0.0.0.0:%d", port),
				"-resources", fmt.Sprintf("%d", repResources),
			)

			repMap[guid] = fmt.Sprintf("http://127.0.0.1:%d", port)

			sess, err := gexec.Start(serverCmd, GinkgoWriter, GinkgoWriter)
			Ω(err).ShouldNot(HaveOccurred())
			Eventually(sess).Should(gbytes.Say("serving"))
			sessionsToTerminate = append(sessionsToTerminate, sess)

			guids = append(guids, guid)
		}

		client := rephttpclient.New(repMap, timeout)

		return client, guids
	}

	panic("wat!")
}
