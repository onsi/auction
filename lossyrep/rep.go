package lossyrep

import (
	"time"

	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/representative"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
)

var LatencyMin time.Duration
var LatencyMax time.Duration
var Timeout time.Duration
var Flakiness = 1.0

type LossyRep struct {
	reps      map[string]*representative.Representative
	FlakyReps map[string]bool
}

func New(reps map[string]*representative.Representative, flakyReps map[string]bool) *LossyRep {
	return &LossyRep{
		reps:      reps,
		FlakyReps: flakyReps,
	}
}

func (rep *LossyRep) beSlowAndFlakey(guid string) bool {
	if rep.FlakyReps[guid] {
		if util.Flake(Flakiness) {
			time.Sleep(Timeout)
			return true
		}
	}
	ok := util.RandomSleep(LatencyMin, LatencyMax, Timeout)
	if !ok {
		return true
	}

	return false
}

func (rep *LossyRep) TotalResources(guid string) int {
	return rep.reps[guid].TotalResources()
}

func (rep *LossyRep) Instances(guid string) []instance.Instance {
	return rep.reps[guid].Instances()
}

func (rep *LossyRep) SetInstances(guid string, instances []instance.Instance) {
	rep.reps[guid].SetInstances(instances)
}

func (rep *LossyRep) Reset(guid string) {
	rep.reps[guid].Reset()
}

func (rep *LossyRep) vote(guid string, instance instance.Instance, c chan types.VoteResult) {
	result := types.VoteResult{
		Rep: guid,
	}
	defer func() {
		c <- result
	}()

	if rep.beSlowAndFlakey(guid) {
		result.Error = "timeout"
		return
	}

	score, err := rep.reps[guid].Vote(instance)
	if err != nil {
		result.Error = err.Error()
		return
	}

	result.Score = score
	return
}

func (rep *LossyRep) Vote(representatives []string, instance instance.Instance) types.VoteResults {
	c := make(chan types.VoteResult)
	for _, guid := range representatives {
		go rep.vote(guid, instance, c)
	}

	results := types.VoteResults{}
	for _ = range representatives {
		results = append(results, <-c)
	}

	return results
}

func (rep *LossyRep) ReserveAndRecastVote(guid string, instance instance.Instance) (result types.VoteResult) {
	result.Rep = guid

	if rep.beSlowAndFlakey(guid) {
		result.Error = "timedout"
		return
	}

	score, err := rep.reps[guid].ReserveAndRecastVote(instance)
	if err != nil {
		result.Error = err.Error()
		return
	}

	result.Score = score
	return
}

func (rep *LossyRep) Release(guid string, instance instance.Instance) {
	rep.beSlowAndFlakey(guid)

	rep.reps[guid].Release(instance)
}

func (rep *LossyRep) Claim(guid string, instance instance.Instance) {
	rep.beSlowAndFlakey(guid)

	rep.reps[guid].Claim(instance)
}
