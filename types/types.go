package types

import (
	"time"

	"github.com/onsi/auction/util"
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
	Algorithm      string  `json:"alg"`
	MaxRounds      int     `json:"mr"`
	MaxBiddingPool float64 `json:"mb"`
}

type RepGuids []string

type VoteResult struct {
	Rep   string  `json:"r"`
	Score float64 `json:"s"`
	Error string  `json:"e"`
}

type VoteResults []VoteResult

type Instance struct {
	AppGuid           string
	InstanceGuid      string
	RequiredResources int
	Tentative         bool
}

func NewInstance(appGuid string, requiredResources int) Instance {
	return Instance{
		AppGuid:           appGuid,
		InstanceGuid:      util.NewGuid("INS"),
		RequiredResources: requiredResources,
		Tentative:         false,
	}
}

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
