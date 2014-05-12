package types

import "github.com/onsi/auction/util"

type RepGuids []string

func (r RepGuids) RandomSubset(n int) RepGuids {
	if len(r) < n {
		return r
	}

	permutation := util.R.Perm(len(r))
	subset := make(RepGuids, n)
	for i, index := range permutation[:n] {
		subset[i] = r[index]
	}

	return subset
}

func (r RepGuids) Without(guids ...string) RepGuids {
	lookup := map[string]bool{}
	for _, guid := range guids {
		lookup[guid] = true
	}

	out := RepGuids{}
	for _, guid := range r {
		if !lookup[guid] {
			out = append(out, guid)
		}
	}

	return out
}
