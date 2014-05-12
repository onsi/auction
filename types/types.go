package types

import (
	"time"

	"github.com/onsi/auction/instance"
)

type AuctionRequest struct {
	Instance instance.Instance `json:"i"`
	RepGuids RepGuids          `json:"rg"`
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
	Vote(guids []string, instance instance.Instance) VoteResults
	ReserveAndRecastVote(guids []string, instance instance.Instance) VoteResults
	Release(guids []string, instance instance.Instance)
	Claim(guid string, instance instance.Instance)
	HesitateAndClaim(guids []string, instance instance.Instance) VoteResults
}

type TestRepPoolClient interface {
	RepPoolClient

	TotalResources(guid string) int
	Instances(guid string) []instance.Instance
	SetInstances(guid string, instances []instance.Instance)
	Reset(guid string)
}
