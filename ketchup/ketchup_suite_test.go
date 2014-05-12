package ketchup_test

import (
	"flag"
	"fmt"

	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/etcdstoreadapter"
	"github.com/cloudfoundry/storeadapter/workerpool"
	"github.com/cloudfoundry/yagnats"
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/nats/repnatsclient"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
	"github.com/onsi/auction/visualization"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"io/ioutil"
	"testing"
	"time"
)

var guids []string
var natsAddrs []string

var numReps int
var repResources int

var rules types.AuctionRules
var timeout time.Duration

var auctioneerMode string

// plumbing
var natsClient yagnats.NATSClient
var client types.TestRepPoolClient
var communicator types.AuctionCommunicator
var storeAdapter storeadapter.StoreAdapter

var svgReport *visualization.SVGReport
var reportName string
var reports []*types.Report

func init() {
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
	time.Sleep(10 * time.Second)
	reportName = fmt.Sprintf("./runs/%s_ketchup_pool%d_conc%d.svg", auctioneer.DefaultRules.Algorithm, auctioneer.DefaultRules.MaxBiddingPool, auctioneer.DefaultRules.MaxConcurrent)
	svgReport = visualization.StartSVGReport(reportName, 2, 3)
	svgReport.DrawHeader("Ketchup", auctioneer.DefaultRules)

	guids = []string{}
	for _, name := range []string{"executor_z1", "executor_z2"} {
		for jobIndex := 0; jobIndex < 5; jobIndex++ {
			for index := 0; index < 10; index++ {
				guids = append(guids, fmt.Sprintf("%s-%d-%d", name, jobIndex, index))
			}
		}
	}

	natsAddrs = []string{
		"10.10.50.20:4222",
		"10.10.114.20:4222",
	}

	etcdAddrs := []string{"http://10.10.50.28:4001"}

	storeAdapter = etcdstoreadapter.NewETCDStoreAdapter(etcdAddrs, workerpool.NewWorkerPool(10))
	err := storeAdapter.Connect()
	Ω(err).ShouldNot(HaveOccurred())

	numReps = len(guids)
	repResources = 100

	fmt.Printf("Running in %s auctioneerMode\n", auctioneerMode)

	//parse flags to set up rules
	timeout = time.Second

	rules = auctioneer.DefaultRules

	natsClient = yagnats.NewClient()
	clusterInfo := &yagnats.ConnectionCluster{}

	for _, addr := range natsAddrs {
		clusterInfo.Members = append(clusterInfo.Members, &yagnats.ConnectionInfo{
			Addr: addr,
		})
	}

	err = natsClient.Connect(clusterInfo)
	Ω(err).ShouldNot(HaveOccurred())

	client = repnatsclient.New(natsClient, timeout)

	if auctioneerMode == "inprocess" {
		communicator = func(auctionRequest types.AuctionRequest) types.AuctionResult {
			return auctioneer.Auction(storeAdapter, client, auctionRequest)
		}
	} else if auctioneerMode == "remote" {
		communicator = func(auctionRequest types.AuctionRequest) types.AuctionResult {
			return auctioneer.RemoteAuction(natsClient, auctionRequest)
		}
	} else {
		panic("wat?")
	}
})

var _ = AfterSuite(func() {
	svgReport.Done()
	reportJSONName := fmt.Sprintf("./runs/%s_ketchup_pool%d_conc%d.json", auctioneer.DefaultRules.Algorithm, auctioneer.DefaultRules.MaxBiddingPool, auctioneer.DefaultRules.MaxConcurrent)
	data, err := json.Marshal(reports)
	Ω(err).ShouldNot(HaveOccurred())
	ioutil.WriteFile(reportJSONName, data, 0777)
})

var _ = BeforeEach(func() {
	err := storeAdapter.Delete("/")
	Ω(err).ShouldNot(HaveOccurred())

	time.Sleep(time.Second)
	for _, guid := range guids {
		client.Reset(guid)
	}

	util.ResetGuids()
})
