package simulation_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/cloudfoundry/gunk/natsrunner"
	"github.com/cloudfoundry/yagnats"
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/auctionrep"
	"github.com/onsi/auction/communication/inprocess"
	"github.com/onsi/auction/communication/nats/repnatsclient"
	"github.com/onsi/auction/communication/rabbit/reprabbitclient"
	"github.com/onsi/auction/simulation/auctiondistributor"
	"github.com/onsi/auction/simulation/simulationrep"
	"github.com/onsi/auction/simulation/visualization"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"testing"
	"time"
)

var communicationMode string
var auctioneerMode string

const InProcess = "inprocess"
const NATS = "nats"
const Rabbit = "rabbit"
const KetchupNATS = "ketchup-nats"
const Remote = "remote"

//these are const because they are fixed on ketchup
const numAuctioneers = 10
const numReps = 100
const repResources = 100

var maxConcurrent int

var timeout time.Duration
var auctionDistributor *auctiondistributor.AuctionDistributor

var svgReport *visualization.SVGReport
var reportName string
var reports []*visualization.Report

var sessionsToTerminate []*gexec.Session
var natsRunner *natsrunner.NATSRunner
var client types.TestRepPoolClient
var guids []string

func init() {
	flag.StringVar(&communicationMode, "communicationMode", "inprocess", "one of inprocess, nats, rabbit, ketchup")
	flag.StringVar(&auctioneerMode, "auctioneerMode", "inprocess", "one of inprocess, remote")
	flag.DurationVar(&timeout, "timeout", 500*time.Millisecond, "timeout when waiting for responses from remote calls")

	flag.StringVar(&(auctioneer.DefaultRules.Algorithm), "algorithm", auctioneer.DefaultRules.Algorithm, "the auction algorithm to use")
	flag.IntVar(&(auctioneer.DefaultRules.MaxRounds), "maxRounds", auctioneer.DefaultRules.MaxRounds, "the maximum number of rounds per auction")
	flag.Float64Var(&(auctioneer.DefaultRules.MaxBiddingPool), "maxBiddingPool", auctioneer.DefaultRules.MaxBiddingPool, "the maximum number of participants in the pool")

	flag.IntVar(&maxConcurrent, "maxConcurrent", 20, "the maximum number of concurrent auctions to run")
}

func TestAuction(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auction Suite")
}

var _ = BeforeSuite(func() {
	fmt.Printf("Running in %s communicationMode\n", communicationMode)
	fmt.Printf("Running in %s auctioneerMode\n", auctioneerMode)

	startReport()

	sessionsToTerminate = []*gexec.Session{}
	hosts := []string{}
	switch communicationMode {
	case InProcess:
		client, guids = buildInProcessReps()
		if auctioneerMode == Remote {
			panic("it doesn't make sense to use remote auctioneers when the reps are in-process")
		}
	case NATS:
		natsAddrs := startNATS()
		client = repnatsclient.New(natsRunner.MessageBus, timeout)
		guids = launchExternalReps("-natsAddrs", natsAddrs)
		if auctioneerMode == Remote {
			hosts = launchExternalAuctioneers("-natsAddrs", natsAddrs)
		}
	case Rabbit:
		rabbitAddr := startRabbit()
		client = reprabbitclient.New(rabbitAddr, timeout)
		guids = launchExternalReps("-rabbitAddr", rabbitAddr)
		if auctioneerMode == Remote {
			hosts = launchExternalAuctioneers("-rabbitAddr", rabbitAddr)
		}
	case KetchupNATS:
		guids = computeKetchupGuids()
		client = ketchupNATSClient()
		if auctioneerMode == Remote {
			hosts = ketchupAuctioneerHosts()
		}
	default:
		panic(fmt.Sprintf("unknown communication mode: %s", communicationMode))
	}

	if auctioneerMode == InProcess {
		auctionDistributor = auctiondistributor.NewInProcessAuctionDistributor(client, maxConcurrent)
	} else if auctioneerMode == Remote {
		auctionDistributor = auctiondistributor.NewRemoteAuctionDistributor(hosts, client, maxConcurrent)
	}
})

var _ = BeforeEach(func() {
	for _, guid := range guids {
		client.Reset(guid)
	}

	util.ResetGuids()
})

var _ = AfterSuite(func() {
	finishReport()

	for _, sess := range sessionsToTerminate {
		sess.Kill().Wait()
	}

	if natsRunner != nil {
		natsRunner.Stop()
	}
})

func buildInProcessReps() (types.TestRepPoolClient, []string) {
	inprocess.LatencyMin = 2 * time.Millisecond
	inprocess.LatencyMax = 12 * time.Millisecond
	inprocess.Timeout = 50 * time.Millisecond
	inprocess.Flakiness = 0.95

	guids := []string{}
	repMap := map[string]auctionrep.AuctionRep{}

	for i := 0; i < numReps; i++ {
		guid := util.NewGuid("REP")
		guids = append(guids, guid)
		repMap[guid] = simulationrep.New(guid, repResources)
	}

	client := inprocess.New(repMap)
	return client, guids
}

func startNATS() string {
	natsPort := 5222 + GinkgoParallelNode()
	natsAddrs := []string{fmt.Sprintf("127.0.0.1:%d", natsPort)}
	natsRunner = natsrunner.NewNATSRunner(natsPort)
	natsRunner.Start()
	return strings.Join(natsAddrs, ",")
}

func startRabbit() string {
	rabbitSession, err := gexec.Start(exec.Command("rabbitmq-server"), GinkgoWriter, GinkgoWriter)
	Ω(err).ShouldNot(HaveOccurred())
	Eventually(rabbitSession, 2).Should(gbytes.Say("Starting broker... completed"))
	sessionsToTerminate = append(sessionsToTerminate, rabbitSession)
	return "amqp://127.0.0.1"
}

func launchExternalReps(communicationFlag string, communicationValue string) []string {
	repNodeBinary, err := gexec.Build("github.com/onsi/auction/simulation/repnode")
	Ω(err).ShouldNot(HaveOccurred())

	guids := []string{}

	for i := 0; i < numReps; i++ {
		guid := util.NewGuid("REP")

		serverCmd := exec.Command(
			repNodeBinary,
			"-guid", guid,
			communicationFlag, communicationValue,
			"-resources", fmt.Sprintf("%d", repResources),
		)

		sess, err := gexec.Start(serverCmd, GinkgoWriter, GinkgoWriter)
		Ω(err).ShouldNot(HaveOccurred())
		Eventually(sess).Should(gbytes.Say("listening"))
		sessionsToTerminate = append(sessionsToTerminate, sess)

		guids = append(guids, guid)
	}

	return guids
}

func launchExternalAuctioneers(communicationFlag string, communicationValue string) []string {
	auctioneerNodeBinary, err := gexec.Build("github.com/onsi/auction/simulation/auctioneernode")
	Ω(err).ShouldNot(HaveOccurred())

	auctioneerHosts := []string{}
	for i := 0; i < numAuctioneers; i++ {
		port := 48710 + i
		auctioneerCmd := exec.Command(
			auctioneerNodeBinary,
			communicationFlag, communicationValue,
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

func computeKetchupGuids() []string {
	guids = []string{}
	for _, name := range []string{"executor_z1", "executor_z2"} {
		for jobIndex := 0; jobIndex < 5; jobIndex++ {
			for index := 0; index < 10; index++ {
				guids = append(guids, fmt.Sprintf("%s-%d-%d", name, jobIndex, index))
			}
		}
	}

	return guids
}

func ketchupAuctioneerHosts() []string {
	return []string{
		"10.10.50.23:48710",
		"10.10.50.24:48710",
		"10.10.50.25:48710",
		"10.10.50.26:48710",
		"10.10.50.27:48710",
		"10.10.114.23:48710",
		"10.10.114.24:48710",
		"10.10.114.25:48710",
		"10.10.114.26:48710",
		"10.10.114.27:48710",
	}
}

func ketchupNATSClient() types.TestRepPoolClient {
	natsAddrs := []string{
		"10.10.50.20:4222",
		"10.10.114.20:4222",
	}

	natsClient := yagnats.NewClient()
	clusterInfo := &yagnats.ConnectionCluster{}

	for _, addr := range natsAddrs {
		clusterInfo.Members = append(clusterInfo.Members, &yagnats.ConnectionInfo{
			Addr: addr,
		})
	}

	err := natsClient.Connect(clusterInfo)
	Ω(err).ShouldNot(HaveOccurred())

	return repnatsclient.New(natsClient, timeout)
}

func startReport() {
	reportName = fmt.Sprintf("./runs/%s_%s_pool%.1f_conc%d.svg", auctioneer.DefaultRules.Algorithm, communicationMode, auctioneer.DefaultRules.MaxBiddingPool, maxConcurrent)
	svgReport = visualization.StartSVGReport(reportName, 2, 3)
	svgReport.DrawHeader(communicationMode, auctioneer.DefaultRules, maxConcurrent)
}

func finishReport() {
	svgReport.Done()
	exec.Command("open", "-a", "safari", reportName).Run()

	reportJSONName := fmt.Sprintf("./runs/%s_%s_pool%.1f_conc%d.json", auctioneer.DefaultRules.Algorithm, communicationMode, auctioneer.DefaultRules.MaxBiddingPool, maxConcurrent)
	data, err := json.Marshal(reports)
	Ω(err).ShouldNot(HaveOccurred())
	ioutil.WriteFile(reportJSONName, data, 0777)
}
