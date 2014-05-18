package auctioneer

import "github.com/onsi/auction/types"

/*

Tell the subset of reps to reserve
    Pick the winner (lowest score)
        Tell the winner to claim and the others to release

*/
func allReserveAuction(client types.RepPoolClient, auctionRequest types.AuctionRequest) (string, int, int) {
	rounds, numCommunications := 1, 0

	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		//pick a subset
		firstRoundReps := auctionRequest.RepGuids.RandomSubsetByFraction(auctionRequest.Rules.MaxBiddingPool)

		//reserve everyone
		numCommunications += len(firstRoundReps)
		votes := client.ReserveAndRecastVote(firstRoundReps, auctionRequest.Instance)

		if votes.AllFailed() {
			continue
		}

		orderedReps := votes.FilterErrors().Shuffle().Sort().Reps()

		numCommunications += len(orderedReps)
		client.Claim(orderedReps[0], auctionRequest.Instance)
		if len(orderedReps) > 1 {
			client.Release(orderedReps[1:], auctionRequest.Instance)
		}

		return orderedReps[0], rounds, numCommunications
	}

	return "", rounds, numCommunications
}
