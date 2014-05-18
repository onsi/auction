package auctionrep

import "github.com/onsi/auction/types"

/*
AuctionRep is the interface that all auction representatives must satisfy.

The simulation rep provides an example of a correct *in-memory* representation.

In particular, it is important that each of these operations be one atomic unit.
*/
type AuctionRep interface {
	Guid() string
	Score(instance types.Instance) (float64, error)
	ScoreThenTentativelyReserve(instance types.Instance) (float64, error)
	ReleaseReservation(instance types.Instance) error
	Claim(instance types.Instance) error
}

//Used in simulation
type TestAuctionRep interface {
	AuctionRep
	TotalResources() int
	Reset()
	SetInstances(instances []types.Instance)
	Instances() []types.Instance
}
