package types

import (
	"time"
)

type AuctionRequest struct {
	Instance Instance     `json:"i"`
	RepGuids RepGuids     `json:"rg"`
	Rules    AuctionRules `json:"r"`
}

type AuctionResult struct {
	Instance          Instance      `json:"i"`
	Winner            string        `json:"w"`
	NumRounds         int           `json:"nr"`
	NumCommunications int           `json:"nc"`
	BiddingDuration   time.Duration `json:"bd"`
	Duration          time.Duration `json:"d"`
}

type AuctionRules struct {
	Algorithm      string `json:"alg"`
	MaxRounds      int    `json:"mr"`
	MaxBiddingPool int    `json:"mb"`
	MaxConcurrent  int    `json:"mc"`
}

type AuctionCommunicator func(AuctionRequest) AuctionResult

type RepPoolClient interface {
	Vote(guids []string, instance Instance) VoteResults
	ReserveAndRecastVote(guids []string, instance Instance) VoteResults
	Release(guids []string, instance Instance)
	Claim(guid string, instance Instance)
}

type TestRepPoolClient interface {
	RepPoolClient

	TotalResources(guid string) int
	Instances(guid string) []Instance
	SetInstances(guid string, instances []Instance)
	Reset(guid string)
}
