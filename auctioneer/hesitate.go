package auctioneer

import "github.com/onsi/auction/types"

/*

Pick a subset of reps
  Tell them to hesitate and claim
    If this succeeds -- we have a winner

*/

func hesitateAuction(client types.RepPoolClient, auctionRequest types.AuctionRequest) (string, int, int) {
	rounds, numCommunications := 1, 0

	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		subset := auctionRequest.RepGuids.RandomSubset(auctionRequest.Rules.MaxBiddingPool)
		votes := client.HesitateAndClaim(subset, auctionRequest.Instance)
		numCommunications += len(subset)
		for _, vote := range votes {
			if vote.Score == 1 {
				return vote.Rep, rounds, numCommunications
			}
		}
	}

	return "", rounds, numCommunications
}
