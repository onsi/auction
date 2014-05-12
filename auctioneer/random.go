package auctioneer

import "github.com/onsi/auction/types"

func randomAuction(client types.RepPoolClient, auctionRequest types.AuctionRequest) (string, int, int) {
	rounds, numCommunications := 1, 0

	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		randomPick := auctionRequest.RepGuids.RandomSubset(1)[0]
		result := client.ReserveAndRecastVote(randomPick, auctionRequest.Instance)
		numCommunications += 1
		if result.Error != "" {
			continue
		}

		client.Claim(randomPick, auctionRequest.Instance)
		numCommunications += 1

		return randomPick, rounds, numCommunications
	}

	return "", rounds, numCommunications
}
