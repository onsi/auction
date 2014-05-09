package auctioneer

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/cloudfoundry/yagnats"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
)

var AllBiddersFull = errors.New("all the bidders were full")

var DefaultRules = types.AuctionRules{
	MaxRounds:        100,
	MaxBiddingPool:   20,
	MaxConcurrent:    20,
	RepickEveryRound: true,
}

func HoldAuctionsFor(client types.RepPoolClient, instances []instance.Instance, representatives []string, rules types.AuctionRules, communicator types.AuctionCommunicator) ([]types.AuctionResult, time.Duration) {
	fmt.Printf("\nStarting Auctions\n\n")
	bar := pb.StartNew(len(instances))

	t := time.Now()
	semaphore := make(chan bool, rules.MaxConcurrent)
	c := make(chan types.AuctionResult)
	for _, inst := range instances {
		go func(inst instance.Instance) {
			semaphore <- true
			c <- communicator(types.AuctionRequest{
				Instance: inst,
				RepGuids: representatives,
				Rules:    rules,
			})
			<-semaphore
		}(inst)
	}

	results := []types.AuctionResult{}
	for _ = range instances {
		results = append(results, <-c)
		bar.Increment()
	}

	bar.Finish()

	return results, time.Since(t)
}

func RemoteAuction(client yagnats.NATSClient, auctionRequest types.AuctionRequest) types.AuctionResult {
	guid := util.RandomGuid()
	payload, _ := json.Marshal(auctionRequest)

	c := make(chan []byte)
	client.Subscribe(guid, func(msg *yagnats.Message) {
		c <- msg.Payload
	})

	client.PublishWithReplyTo("diego.auction", guid, payload)

	var responsePayload []byte
	select {
	case responsePayload = <-c:
	case <-time.After(time.Minute):
		return types.AuctionResult{}
	}

	var auctionResult types.AuctionResult
	err := json.Unmarshal(responsePayload, &auctionResult)
	if err != nil {
		panic(err)
	}

	return auctionResult
}

func Auction(client types.RepPoolClient, auctionRequest types.AuctionRequest) types.AuctionResult {
	var auctionWinner string

	var representatives []string

	if !auctionRequest.Rules.RepickEveryRound {
		representatives = randomSubset(auctionRequest.RepGuids, auctionRequest.Rules.MaxBiddingPool)
	}

	numRounds, numVotes := 0, 0
	t := time.Now()
	for round := 1; round <= auctionRequest.Rules.MaxRounds; round++ {
		if auctionRequest.Rules.RepickEveryRound {
			representatives = randomSubset(auctionRequest.RepGuids, auctionRequest.Rules.MaxBiddingPool)
		}
		numRounds++
		winner, _, err := vote(client, auctionRequest.Instance, representatives)
		numVotes += len(representatives)
		if err != nil {
			continue
		}

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

		secondRoundVoters := []string{}

		for _, rep := range representatives {
			if rep != winner {
				secondRoundVoters = append(secondRoundVoters, rep)
			}
		}

		_, secondPlaceScore, err := vote(client, auctionRequest.Instance, secondRoundVoters)

		winnerRecast := <-c
		numVotes += len(representatives)

		if winnerRecast.Error != "" {
			//winner ran out of space on the recast, retry
			continue
		}

		if err == nil && secondPlaceScore < winnerRecast.Score && round < auctionRequest.Rules.MaxRounds {
			client.Release(winner, auctionRequest.Instance)
			continue
		}

		client.Claim(winner, auctionRequest.Instance)
		auctionWinner = winner
		break
	}

	return types.AuctionResult{
		Winner:    auctionWinner,
		Instance:  auctionRequest.Instance,
		NumRounds: numRounds,
		NumVotes:  numVotes,
		Duration:  time.Since(t),
	}
}

func randomSubset(representatives []string, subsetSize int) []string {
	reps := representatives
	if len(reps) > subsetSize {
		permutation := util.R.Perm(len(representatives))
		reps = []string{}
		for _, index := range permutation[:subsetSize] {
			reps = append(reps, representatives[index])
		}
	}

	return reps
}

func vote(client types.RepPoolClient, instance instance.Instance, representatives []string) (string, float64, error) {
	results := client.Vote(representatives, instance)

	winningScore := 1e9
	winners := []string{}

	for _, result := range results {
		if result.Error != "" {
			continue
		}

		if result.Score < winningScore {
			winningScore = result.Score
			winners = []string{result.Rep}
		} else if result.Score == winningScore { // can be less strict here
			winners = append(winners, result.Rep)
		}
	}

	if len(winners) == 0 {
		return "", 0, AllBiddersFull
	}

	winner := winners[util.R.Intn(len(winners))]

	return winner, winningScore, nil
}
