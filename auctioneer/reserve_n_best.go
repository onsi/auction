package auctioneer

import "github.com/onsi/auction/types"

/*

Get the scores from the subset of reps
	Tell the top 5 to reserve
		Pick the best from that set and release the others

*/

func reserveNBestAuction(client types.RepPoolClient, auctionRequest types.AuctionRequest) (string, int, int) {
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

		// pick the top 5 winners
		winners := firstRoundVotes.FilterErrors().Shuffle().Sort()[:5]

		//ask them to reserve
		c := make(chan types.VoteResult)
		numCommunications += len(winners)
		for _, winner := range winners {
			go func(rep string) {
				c <- client.ReserveAndRecastVote(rep, auctionRequest.Instance)
			}(winner.Rep)
		}

		winners = make(types.VoteResults, 5)
		for i := range winners {
			winners[i] = <-c
		}

		//if they're all out of space, try again
		if winners.AllFailed() {
			continue
		}

		//order by score: the first is the winner, all others release
		winners = winners.FilterErrors().Shuffle().Sort()
		done := make(chan bool)

		numCommunications += len(winners)
		for i, winner := range winners {
			if i == 0 {
				go func(rep string) {
					client.Claim(rep, auctionRequest.Instance)
					done <- true
				}(winner.Rep)
			} else {
				go func(rep string) {
					client.Release(rep, auctionRequest.Instance)
					done <- true
				}(winner.Rep)
			}
		}

		for _ = range winners {
			<-done
		}

		return winners[0].Rep, rounds, numCommunications
	}

	return "", rounds, numCommunications
}
