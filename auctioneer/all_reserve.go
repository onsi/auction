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
		firstRoundReps := auctionRequest.RepGuids.RandomSubset(auctionRequest.Rules.MaxBiddingPool)

		//reserve everyone
		numCommunications += len(firstRoundReps)
		c := make(chan types.VoteResult)
		for _, rep := range firstRoundReps {
			go func(rep string) {
				c <- client.ReserveAndRecastVote(rep, auctionRequest.Instance)
			}(rep)
		}

		votes := types.VoteResults{}
		for _ = range firstRoundReps {
			votes = append(votes, <-c)
		}

		if votes.AllFailed() {
			continue
		}

		orderdReps := votes.FilterErrors().Shuffle().Sort().Reps()

		done := make(chan bool)

		numCommunications += len(orderdReps)
		for i, rep := range orderdReps {
			if i == 0 {
				go func(rep string) {
					client.Claim(rep, auctionRequest.Instance)
					done <- true
				}(rep)
			} else {
				go func(rep string) {
					client.Release(rep, auctionRequest.Instance)
					done <- true
				}(rep)
			}
		}

		for _ = range orderdReps {
			<-done
		}

		return orderdReps[0], rounds, numCommunications
	}

	return "", rounds, numCommunications
}
