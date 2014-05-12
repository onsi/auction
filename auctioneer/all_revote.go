package auctioneer

import "github.com/onsi/auction/types"

func allRevoteAuction(client types.RepPoolClient, auctionRequest types.AuctionRequest) (string, int, int) {
	rounds, numCommunications := 1, 0

	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		representatives := randomSubset(auctionRequest.RepGuids, auctionRequest.Rules.MaxBiddingPool)

		winner, _, err := vote(client, auctionRequest.Instance, representatives)
		numCommunications += len(representatives)
		if err != nil {
			continue
		}

		//make this a one-liner
		c := make(chan types.VoteResult)
		go func() {
			winnerScore, err := client.ReserveAndRecastVote(winner, auctionRequest.Instance)
			result := types.VoteResult{
				Rep: winner,
			}
			if err != nil {
				result.Error = err.Error()
				c <- result
				return
			}
			result.Score = winnerScore
			c <- result
		}()

		//and make filtering a oneliner
		secondRoundVoters := []string{}

		for _, rep := range representatives {
			if rep != winner {
				secondRoundVoters = append(secondRoundVoters, rep)
			}
		}

		_, secondPlaceScore, err := vote(client, auctionRequest.Instance, secondRoundVoters)

		winnerRecast := <-c
		numCommunications += len(representatives)

		if winnerRecast.Error != "" {
			//winner ran out of space on the recast, retry
			continue
		}

		if err == nil && secondPlaceScore < winnerRecast.Score && rounds < auctionRequest.Rules.MaxRounds {
			client.Release(winner, auctionRequest.Instance)
			numCommunications += 1
			continue
		}

		client.Claim(winner, auctionRequest.Instance)
		numCommunications += 1
		return winner, rounds, numCommunications
	}

	return "", rounds, numCommunications
}
