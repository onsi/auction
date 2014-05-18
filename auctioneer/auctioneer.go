package auctioneer

import (
	"errors"
	"time"

	"github.com/onsi/auction/types"
)

var AllBiddersFull = errors.New("all the bidders were full")

var DefaultRules = types.AuctionRules{
	Algorithm:      "reserve_n_best",
	MaxRounds:      100,
	MaxBiddingPool: 0.2,
}

func Auction(client types.RepPoolClient, auctionRequest types.AuctionRequest) types.AuctionResult {
	result := types.AuctionResult{
		Instance: auctionRequest.Instance,
	}

	t := time.Now()
	switch auctionRequest.Rules.Algorithm {
	case "all_revote":
		result.Winner, result.NumRounds, result.NumCommunications = allRevoteAuction(client, auctionRequest)
	case "all_reserve":
		result.Winner, result.NumRounds, result.NumCommunications = allReserveAuction(client, auctionRequest)
	case "pick_among_best":
		result.Winner, result.NumRounds, result.NumCommunications = pickAmongBestAuction(client, auctionRequest)
	case "pick_best":
		result.Winner, result.NumRounds, result.NumCommunications = pickBestAuction(client, auctionRequest)
	case "reserve_n_best":
		result.Winner, result.NumRounds, result.NumCommunications = reserveNBestAuction(client, auctionRequest)
	case "random":
		result.Winner, result.NumRounds, result.NumCommunications = randomAuction(client, auctionRequest)
	default:
		panic("unkown algorithm " + auctionRequest.Rules.Algorithm)
	}
	result.BiddingDuration = time.Since(t)

	return result
}
