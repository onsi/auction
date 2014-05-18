package inprocess

import (
	"time"

	"github.com/onsi/auction/auctionrep"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
)

var LatencyMin time.Duration
var LatencyMax time.Duration
var Timeout time.Duration
var Flakiness = 1.0

type InprocessClient struct {
	reps map[string]auctionrep.AuctionRep
}

func New(reps map[string]auctionrep.AuctionRep) *InprocessClient {
	return &InprocessClient{
		reps: reps,
	}
}

func randomSleep(min time.Duration, max time.Duration, timeout time.Duration) bool {
	sleepDuration := time.Duration(util.R.Float64()*float64(max-min) + float64(min))
	if sleepDuration <= timeout {
		time.Sleep(sleepDuration)
		return true
	} else {
		time.Sleep(timeout)
		return false
	}
}

func (client *InprocessClient) beSlowAndPossiblyTimeout(guid string) bool {
	sleepDuration := time.Duration(util.R.Float64()*float64(LatencyMax-LatencyMin) + float64(LatencyMin))

	if sleepDuration <= Timeout {
		time.Sleep(sleepDuration)
		return false
	} else {
		time.Sleep(Timeout)
		return true
	}
}

func (client *InprocessClient) testAuctionRep(guid string) auctionrep.TestAuctionRep {
	tar, ok := client.reps[guid].(auctionrep.TestAuctionRep)
	if !ok {
		panic("attempting to do a test-like thing with a non-test-like rep")
	}

	return tar
}

func (client *InprocessClient) TotalResources(guid string) int {
	return client.testAuctionRep(guid).TotalResources()
}

func (client *InprocessClient) Instances(guid string) []types.Instance {
	return client.testAuctionRep(guid).Instances()
}

func (client *InprocessClient) SetInstances(guid string, instances []types.Instance) {
	client.testAuctionRep(guid).SetInstances(instances)
}

func (client *InprocessClient) Reset(guid string) {
	client.testAuctionRep(guid).Reset()
}

func (client *InprocessClient) vote(guid string, instance types.Instance, c chan types.VoteResult) {
	result := types.VoteResult{
		Rep: guid,
	}
	defer func() {
		c <- result
	}()

	if client.beSlowAndPossiblyTimeout(guid) {
		result.Error = "timeout"
		return
	}

	score, err := client.reps[guid].Score(instance)
	if err != nil {
		result.Error = err.Error()
		return
	}

	result.Score = score
	return
}

func (client *InprocessClient) Vote(representatives []string, instance types.Instance) types.VoteResults {
	c := make(chan types.VoteResult)
	for _, guid := range representatives {
		go client.vote(guid, instance, c)
	}

	results := types.VoteResults{}
	for _ = range representatives {
		results = append(results, <-c)
	}

	return results
}

func (client *InprocessClient) reserveAndRecastVote(guid string, instance types.Instance, c chan types.VoteResult) {
	result := types.VoteResult{
		Rep: guid,
	}
	defer func() {
		c <- result
	}()

	if client.beSlowAndPossiblyTimeout(guid) {
		result.Error = "timedout"
		return
	}

	score, err := client.reps[guid].ScoreThenTentativelyReserve(instance)
	if err != nil {
		result.Error = err.Error()
		return
	}

	result.Score = score
	return
}

func (client *InprocessClient) ReserveAndRecastVote(guids []string, instance types.Instance) types.VoteResults {
	c := make(chan types.VoteResult)
	for _, guid := range guids {
		go client.reserveAndRecastVote(guid, instance, c)
	}

	results := types.VoteResults{}
	for _ = range guids {
		results = append(results, <-c)
	}

	return results
}

func (client *InprocessClient) Release(guids []string, instance types.Instance) {
	c := make(chan bool)
	for _, guid := range guids {
		go func(guid string) {
			client.beSlowAndPossiblyTimeout(guid)
			client.reps[guid].ReleaseReservation(instance)
			c <- true
		}(guid)
	}

	for _ = range guids {
		<-c
	}
}

func (client *InprocessClient) Claim(guid string, instance types.Instance) {
	client.beSlowAndPossiblyTimeout(guid)

	client.reps[guid].Claim(instance)
}
