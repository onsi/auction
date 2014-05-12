package auctioneer

import "github.com/onsi/auction/types"

/*

Get the scores from the subset of reps
    Select the best

*/

func pickBestAuction(client types.RepPoolClient, auctionRequest types.AuctionRequest) (string, int, int) {
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

		winner := firstRoundVotes.FilterErrors().Shuffle().Sort()[0]

		result := client.ReserveAndRecastVote([]string{winner.Rep}, auctionRequest.Instance)[0]
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
