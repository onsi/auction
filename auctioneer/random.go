package auctioneer

import (
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
)

func randomAuction(client types.RepPoolClient, auctionRequest types.AuctionRequest) (string, int, int) {
	rounds, numCommunications := 1, 0

	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		randomPick := auctionRequest.RepGuids[util.R.Intn(len(auctionRequest.RepGuids))]
		_, err := client.ReserveAndRecastVote(randomPick, auctionRequest.Instance)
		numCommunications += 1
		if err != nil {
			continue
		}

		client.Claim(randomPick, auctionRequest.Instance)
		numCommunications += 1

		return randomPick, rounds, numCommunications
	}

	return "", rounds, numCommunications
}
