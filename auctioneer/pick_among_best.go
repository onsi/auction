package auctioneer

import "github.com/onsi/auction/types"

/*

Get the scores from the subset of reps
	Select the best 5
		Pick a winner randomly from that set

*/

func pickAmongBestAuction(client types.RepPoolClient, auctionRequest types.AuctionRequest) (string, int, int) {
	rounds, numCommunications := 1, 0

	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		//pick a subset
		firstRoundReps := auctionRequest.RepGuids.RandomSubset(auctionRequest.Rules.MaxBiddingPool)

		//get everyone's score, if they're all full: bail
		numCommunications += len(firstRoundReps)
		firstRoundVotes := client.Vote(firstRoundReps, auctionRequest.Instance)
		if firstRoundVotes.AllFailed() {
			continue
		}

		top5Winners := firstRoundVotes.FilterErrors().Shuffle().Sort()[:5]

		winner := top5Winners.Shuffle()[0]

		result := client.ReserveAndRecastVote(winner.Rep, auctionRequest.Instance)
		numCommunications += 1
		if result.Error != "" {
			continue
		}

		client.Claim(winner.Rep, auctionRequest.Instance)
		numCommunications += 1

		return winner.Rep, rounds, numCommunications
	}

	return "", rounds, numCommunications
}
