package auctioneer

import "github.com/onsi/auction/types"

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
		c := make(chan types.VoteResult)
		numCommunications += 1
		go func() {
			c <- client.ReserveAndRecastVote(winner.Rep, auctionRequest.Instance)
		}()

		//get everyone's score again
		secondRoundReps := firstRoundReps.Without(winner.Rep)
		numCommunications += len(secondRoundReps)
		secondRoundVotes := client.Vote(secondRoundReps, auctionRequest.Instance)

		winnerRecast := <-c

		//if the winner ran out of space: bail
		if winnerRecast.Error != "" {
			continue
		}

		// if the second place winner has a better score than the original winner: bail
		if !secondRoundVotes.AllFailed() {
			secondPlace := secondRoundVotes.FilterErrors().Shuffle().Sort()[0]
			if secondPlace.Score < winnerRecast.Score {
				client.Release(winner.Rep, auctionRequest.Instance)
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
