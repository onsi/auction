package auctioneer

import "github.com/onsi/auction/types"

/*

Get the scores from the subset of reps
    Pick the winner (lowest score)
        Tell the winner to reserve and the others to revote
        	If the winner still has the lowest score we are done, otherwise, repeat

*/

func allRevoteAuction(client types.RepPoolClient, auctionRequest types.AuctionRequest) (string, int, int) {
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

		// tell the winner to reserve
		numCommunications += 1
		winnerRecast := client.ReserveAndRecastVote([]string{winner.Rep}, auctionRequest.Instance)[0]

		//get everyone's score again
		secondRoundReps := firstRoundReps.Without(winner.Rep)
		numCommunications += len(secondRoundReps)
		secondRoundVotes := client.Vote(secondRoundReps, auctionRequest.Instance)

		//if the winner ran out of space: bail
		if winnerRecast.Error != "" {
			continue
		}

		// if the second place winner has a better score than the original winner: bail
		if !secondRoundVotes.AllFailed() {
			secondPlace := secondRoundVotes.FilterErrors().Shuffle().Sort()[0]
			if secondPlace.Score < winnerRecast.Score {
				client.Release([]string{winner.Rep}, auctionRequest.Instance)
				numCommunications += 1
				continue
			}
		}

		client.Claim(winner.Rep, auctionRequest.Instance)
		numCommunications += 1
		return winner.Rep, rounds, numCommunications
	}

	return "", rounds, numCommunications
}
