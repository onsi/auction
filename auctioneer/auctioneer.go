package auctioneer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/cloudfoundry/storeadapter"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
)

var AllBiddersFull = errors.New("all the bidders were full")

var DefaultRules = types.AuctionRules{
	Algorithm:      "all_revote",
	MaxRounds:      100,
	MaxBiddingPool: 20,
	MaxConcurrent:  20,
}

func HoldAuctionsFor(client types.TestRepPoolClient, instances []instance.Instance, representatives []string, rules types.AuctionRules, communicator types.AuctionCommunicator) *types.Report {
	fmt.Printf("\nStarting Auctions\n\n")
	bar := pb.StartNew(len(instances))

	t := time.Now()
	semaphore := make(chan bool, rules.MaxConcurrent)
	c := make(chan types.AuctionResult)
	for _, inst := range instances {
		go func(inst instance.Instance) {
			semaphore <- true
			result := communicator(types.AuctionRequest{
				Instance: inst,
				RepGuids: representatives,
				Rules:    rules,
			})
			result.Duration = time.Since(t)
			c <- result
			<-semaphore
		}(inst)
	}

	results := []types.AuctionResult{}
	for _ = range instances {
		results = append(results, <-c)
		bar.Increment()
	}

	bar.Finish()

	duration := time.Since(t)
	report := &types.Report{
		RepGuids:        representatives,
		AuctionResults:  results,
		InstancesByRep:  types.FetchAndSortInstances(client, representatives),
		AuctionDuration: duration,
	}

	return report
}

type HTTPRemoteAuctions struct {
	hosts []string
}

func NewHTTPRemoteAuctions(hosts []string) *HTTPRemoteAuctions {
	return &HTTPRemoteAuctions{hosts}
}

func (h *HTTPRemoteAuctions) RemoteAuction(auctionRequest types.AuctionRequest) types.AuctionResult {
	host := h.hosts[util.R.Intn(len(h.hosts))]

	var result types.AuctionResult

	payload, _ := json.Marshal(auctionRequest)
	res, err := http.Post("http://"+host+"/auction", "application/json", bytes.NewReader(payload))
	if err != nil {
		fmt.Println("FAILED! TO AUCTION", err)
		return types.AuctionResult{
			Instance: auctionRequest.Instance,
		}
	}

	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var result types.AuctionResult
	json.Unmarshal(data, &result)

	return result
}

func Auction(etcdStoreAdapter storeadapter.StoreAdapter, client types.RepPoolClient, auctionRequest types.AuctionRequest) types.AuctionResult {
	result := types.AuctionResult{
		Instance: auctionRequest.Instance,
	}

	t := time.Now()
	switch auctionRequest.Rules.Algorithm {
	case "all_revote":
		result.Winner, result.NumRounds, result.NumCommunications = allRevoteAuction(client, auctionRequest)
	case "all_reserve":
		result.Winner, result.NumRounds, result.NumCommunications = allReserveAuction(client, auctionRequest)
	case "pick_among_best":
		result.Winner, result.NumRounds, result.NumCommunications = pickAmongBestAuction(client, auctionRequest)
	case "pick_best":
		result.Winner, result.NumRounds, result.NumCommunications = pickBestAuction(client, auctionRequest)
	case "reserve_n_best":
		result.Winner, result.NumRounds, result.NumCommunications = reserveNBestAuction(client, auctionRequest)
	case "random":
		result.Winner, result.NumRounds, result.NumCommunications = randomAuction(client, auctionRequest)
	case "hesitate":
		result.Winner, result.NumRounds, result.NumCommunications = hesitateAuction(etcdStoreAdapter, client, auctionRequest)
	default:
		panic("unkown algorithm " + auctionRequest.Rules.Algorithm)
	}
	result.BiddingDuration = time.Since(t)

	return result
}
