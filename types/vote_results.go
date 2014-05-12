package types

import (
	"sort"

	"github.com/onsi/auction/util"
)

type VoteResult struct {
	Rep   string  `json:"r"`
	Score float64 `json:"s"`
	Error string  `json:"e"`
}

type VoteResults []VoteResult

func (a VoteResults) Len() int           { return len(a) }
func (a VoteResults) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a VoteResults) Less(i, j int) bool { return a[i].Score < a[j].Score }

func (v VoteResults) AllFailed() bool {
	return len(v.FilterErrors()) == 0
}

func (v VoteResults) Reps() RepGuids {
	out := RepGuids{}
	for _, r := range v {
		out = append(out, r.Rep)
	}

	return out
}

func (v VoteResults) FilterErrors() VoteResults {
	out := VoteResults{}
	for _, r := range v {
		if r.Error == "" {
			out = append(out, r)
		}
	}

	return out
}

func (v VoteResults) Shuffle() VoteResults {
	out := make(VoteResults, len(v))

	perm := util.R.Perm(len(v))
	for i, index := range perm {
		out[i] = v[index]
	}

	return out
}

func (v VoteResults) Sort() VoteResults {
	sort.Sort(v)
	return v
}
