package visualization

import (
	"fmt"
	"strings"
	"time"

	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/lossyrep"
	"github.com/onsi/auction/types"
)

const defaultStyle = "\x1b[0m"
const boldStyle = "\x1b[1m"
const redColor = "\x1b[91m"
const greenColor = "\x1b[32m"
const yellowColor = "\x1b[33m"
const cyanColor = "\x1b[36m"
const grayColor = "\x1b[90m"
const lightGrayColor = "\x1b[37m"

func PrintReport(client types.RepPoolClient, results []types.AuctionResult, representatives []string, duration time.Duration) {
	roundsDistribution := map[int]int{}
	auctionedInstances := map[string]bool{}

	///
	maxRounds := 0
	fmt.Println("Rounds")
	for _, result := range results {
		roundsDistribution[result.NumRounds] += 1
		auctionedInstances[result.Instance.InstanceGuid] = true
		if result.NumRounds > maxRounds {
			maxRounds = result.NumRounds
		}
	}

	for i := 1; i <= maxRounds; i++ {
		if roundsDistribution[i] > 0 {
			fmt.Printf("  %2d: %s\n", i, strings.Repeat("■", roundsDistribution[i]))
		}
	}

	///

	minRounds, maxRounds, totalRounds, meanRounds := 100000000, 0, 0, float64(0)
	for _, result := range results {
		if result.NumRounds < minRounds {
			minRounds = result.NumRounds
		}
		if result.NumRounds > maxRounds {
			maxRounds = result.NumRounds
		}
		totalRounds += result.NumRounds
		meanRounds += float64(result.NumRounds)
	}

	meanRounds = meanRounds / float64(len(results))
	fmt.Printf("  Min: %d | Max: %d | Total: %d | Mean: %.2f\n", minRounds, maxRounds, totalRounds, meanRounds)

	///

	fmt.Println("Votes")
	minVotes, maxVotes, totalVotes, meanVotes := 100000000, 0, 0, float64(0)
	for _, result := range results {
		if result.NumVotes < minVotes {
			minVotes = result.NumVotes
		}
		if result.NumVotes > maxVotes {
			maxVotes = result.NumVotes
		}
		totalVotes += result.NumVotes
		meanVotes += float64(result.NumVotes)
	}

	meanVotes = meanVotes / float64(len(results))
	fmt.Printf("  Min: %d | Max: %d | Total: %d | Mean: %.2f\n", minVotes, maxVotes, totalVotes, meanVotes)

	///

	fmt.Println("Distribution")
	maxGuidLength := 0
	for _, guid := range representatives {
		if len(guid) > maxGuidLength {
			maxGuidLength = len(guid)
		}
	}
	guidFormat := fmt.Sprintf("%%%ds", maxGuidLength)

	numNew := 0
	for _, guid := range representatives {
		repString := fmt.Sprintf(guidFormat, guid)
		lossyRep, ok := client.(*lossyrep.LossyRep)
		if ok && lossyRep.FlakyReps[guid] {
			repString = fmt.Sprintf("%s"+guidFormat+"%s", redColor, repString, defaultStyle)
		}

		instanceString := ""
		instances := client.Instances(guid)

		availableColors := []string{"green", "red", "cyan", "yellow", "gray"}
		colorLookup := map[string]string{"red": redColor, "green": greenColor, "cyan": cyanColor, "yellow": yellowColor, "gray": lightGrayColor}

		originalCounts := map[string]int{}
		newCounts := map[string]int{}
		for _, instance := range instances {
			key := "green"
			if _, ok := colorLookup[instance.AppGuid]; ok {
				key = instance.AppGuid
			}
			if auctionedInstances[instance.InstanceGuid] {
				newCounts[key] += 1
				numNew += 1
			} else {
				originalCounts[key] += 1
			}
		}
		for _, col := range availableColors {
			instanceString += strings.Repeat(colorLookup[col]+"○"+defaultStyle, originalCounts[col])
			instanceString += strings.Repeat(colorLookup[col]+"●"+defaultStyle, newCounts[col])
		}
		instanceString += strings.Repeat(grayColor+"○"+defaultStyle, client.TotalResources(guid)-len(instances))

		fmt.Printf("  %s: %s\n", repString, instanceString)
	}

	fmt.Printf("Finished %d Auctions among %d Representatives in %s\n", len(results), len(representatives), duration)
	if numNew < len(auctionedInstances) {
		expected := len(auctionedInstances)
		fmt.Printf("  %s!!!!MISSING INSTANCES!!!!  Expected %d, got %d (%.3f %% failure rate)%s", redColor, expected, numNew, float64(expected-numNew)/float64(expected), defaultStyle)
	}
	fmt.Printf("  MaxConcurrent: %d, MaxBiddingBool:%d\n", auctioneer.MaxConcurrent, auctioneer.MaxBiddingPool)
	if _, ok := client.(*lossyrep.LossyRep); ok {
		fmt.Printf("  Latency Range: %s < %s, Timeout: %s, Flakiness: %.2f\n", lossyrep.LatencyMin, lossyrep.LatencyMax, lossyrep.Timeout, lossyrep.Flakiness)
	}

	///

	minTime, maxTime, totalTime, meanTime := time.Hour, time.Duration(0), time.Duration(0), time.Duration(0)
	for _, result := range results {
		if result.Duration < minTime {
			minTime = result.Duration
		}
		if result.Duration > maxTime {
			maxTime = result.Duration
		}
		totalTime += result.Duration
		meanTime += result.Duration
	}

	meanTime = meanTime / time.Duration(len(results))
	fmt.Printf("  Min: %s | Max: %s | Total: %s | Mean: %s\n", minTime, maxTime, totalTime, meanTime)

}
