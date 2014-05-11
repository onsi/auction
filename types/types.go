package types

import (
	"sort"
	"time"

	"github.com/GaryBoone/GoStats/stats"
	"github.com/onsi/auction/instance"
)

type VoteResult struct {
	Rep   string  `json:"r"`
	Score float64 `json:"s"`
	Error string  `json:"e"`
}

type AuctionRequest struct {
	Instance instance.Instance `json:"i"`
	RepGuids []string          `json:"rg"`
	Rules    AuctionRules      `json:"r"`
}

type AuctionResult struct {
	Instance          instance.Instance `json:"i"`
	Winner            string            `json:"w"`
	NumRounds         int               `json:"nr"`
	NumCommunications int               `json:"nc"`
	BiddingDuration   time.Duration     `json:"bd"`
	Duration          time.Duration     `json:"d"`
}

type AuctionRules struct {
	Algorithm      string `json:"alg"`
	MaxRounds      int    `json:"mr"`
	MaxBiddingPool int    `json:"mb"`
	MaxConcurrent  int    `json:"mc"`
}

type AuctionCommunicator func(AuctionRequest) AuctionResult

type RepPoolClient interface {
	Vote(guids []string, instance instance.Instance) []VoteResult
	ReserveAndRecastVote(guid string, instance instance.Instance) (float64, error)
	Release(guid string, instance instance.Instance)
	Claim(guid string, instance instance.Instance)
}

type TestRepPoolClient interface {
	RepPoolClient

	TotalResources(guid string) int
	Instances(guid string) []instance.Instance
	SetInstances(guid string, instances []instance.Instance)
	Reset(guid string)
}

type Report struct {
	RepGuids                     []string
	AuctionResults               []AuctionResult
	InstancesByRep               map[string][]instance.Instance
	AuctionDuration              time.Duration
	auctionedInstancesByInstGuid map[string]bool
}

type Stat struct {
	Min    float64
	Max    float64
	Mean   float64
	StdDev float64
	Total  float64
}

func NewStat(data []float64) Stat {
	return Stat{
		Min:    stats.StatsMin(data),
		Max:    stats.StatsMax(data),
		Mean:   stats.StatsMean(data),
		StdDev: stats.StatsPopulationStandardDeviation(data),
		Total:  stats.StatsSum(data),
	}
}

func (r *Report) IsAuctionedInstance(inst instance.Instance) bool {
	if r.auctionedInstancesByInstGuid == nil {
		r.auctionedInstancesByInstGuid = map[string]bool{}
		for _, result := range r.AuctionResults {
			r.auctionedInstancesByInstGuid[result.Instance.InstanceGuid] = true
		}
	}

	return r.auctionedInstancesByInstGuid[inst.InstanceGuid]
}

func (r *Report) NAuctions() int {
	return len(r.AuctionResults)
}

func (r *Report) NReps() int {
	return len(r.RepGuids)
}

func (r *Report) InitialDistributionScore() float64 {
	instanceCounts := []float64{}
	for _, instances := range r.InstancesByRep {
		count := 0
		for _, instance := range instances {
			if !r.IsAuctionedInstance(instance) {
				count++
			}
		}
		instanceCounts = append(instanceCounts, float64(count))
	}

	if stats.StatsSum(instanceCounts) == 0 {
		return 0
	}

	return stats.StatsPopulationStandardDeviation(instanceCounts) / stats.StatsMean(instanceCounts)
}

func (r *Report) DistributionScore() float64 {
	instanceCounts := []float64{}
	for _, instances := range r.InstancesByRep {
		instanceCounts = append(instanceCounts, float64(len(instances)))
	}

	return stats.StatsPopulationStandardDeviation(instanceCounts) / stats.StatsMean(instanceCounts)
}

func (r *Report) AuctionsPerSecond() float64 {
	return float64(r.NAuctions()) / r.AuctionDuration.Seconds()
}

func (r *Report) CommStats() Stat {
	comms := []float64{}
	for _, result := range r.AuctionResults {
		comms = append(comms, float64(result.NumCommunications))
	}

	return NewStat(comms)
}

func (r *Report) BiddingTimeStats() Stat {
	biddingTimes := []float64{}
	for _, result := range r.AuctionResults {
		biddingTimes = append(biddingTimes, result.BiddingDuration.Seconds())
	}

	return NewStat(biddingTimes)
}

func (r *Report) WaitTimeStats() Stat {
	waitTimes := []float64{}
	for _, result := range r.AuctionResults {
		waitTimes = append(waitTimes, result.Duration.Seconds())
	}

	return NewStat(waitTimes)
}

func FetchAndSortInstances(client TestRepPoolClient, repGuids []string) map[string][]instance.Instance {
	instancesByRepGuid := map[string][]instance.Instance{}
	for _, guid := range repGuids {
		instances := client.Instances(guid)
		sort.Sort(ByAppGuid(instances))
		instancesByRepGuid[guid] = instances
	}

	return instancesByRepGuid
}

type ByAppGuid []instance.Instance

func (a ByAppGuid) Len() int           { return len(a) }
func (a ByAppGuid) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAppGuid) Less(i, j int) bool { return a[i].AppGuid < a[j].AppGuid }
