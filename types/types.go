package types

import (
	"time"

	"github.com/onsi/auction/instance"
)

type VoteResult struct {
	Rep   string
	Score float64
	Error string
}

type AuctionResult struct {
	Instance  instance.Instance
	Winner    string
	NumRounds int
	NumVotes  int
	Duration  time.Duration
}

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
