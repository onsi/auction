package repnatsserver

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/cloudfoundry/yagnats"
	"github.com/onsi/auction/auctionrep"
	"github.com/onsi/auction/types"
)

var errorResponse = []byte("error")
var successResponse = []byte("ok")

func Start(natsAddrs []string, rep auctionrep.AuctionRep) {
	client := yagnats.NewClient()

	clusterInfo := &yagnats.ConnectionCluster{}

	for _, addr := range natsAddrs {
		clusterInfo.Members = append(clusterInfo.Members, &yagnats.ConnectionInfo{
			Addr: addr,
		})
	}

	err := client.Connect(clusterInfo)
	if err != nil {
		log.Fatalln("no nats:", err)
	}

	guid := rep.Guid()

	testAuctionRep := func() auctionrep.TestAuctionRep {
		tar, ok := rep.(auctionrep.TestAuctionRep)
		if !ok {
			panic("attempting to do a test-like thing with a non-test-like rep")
		}

		return tar
	}

	client.Subscribe(guid+".total_resources", func(msg *yagnats.Message) {
		jresources, _ := json.Marshal(testAuctionRep().TotalResources())
		client.Publish(msg.ReplyTo, jresources)
	})

	client.Subscribe(guid+".reset", func(msg *yagnats.Message) {
		testAuctionRep().Reset()
		client.Publish(msg.ReplyTo, successResponse)
	})

	client.Subscribe(guid+".set_instances", func(msg *yagnats.Message) {
		var instances []types.Instance

		err := json.Unmarshal(msg.Payload, &instances)
		if err != nil {
			client.Publish(msg.ReplyTo, errorResponse)
		}

		testAuctionRep().SetInstances(instances)
		client.Publish(msg.ReplyTo, successResponse)
	})

	client.Subscribe(guid+".instances", func(msg *yagnats.Message) {
		jinstances, _ := json.Marshal(testAuctionRep().Instances())
		client.Publish(msg.ReplyTo, jinstances)
	})

	client.Subscribe(guid+".vote", func(msg *yagnats.Message) {
		var inst types.Instance

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			panic(err)
		}

		response := types.VoteResult{
			Rep: guid,
		}

		defer func() {
			payload, _ := json.Marshal(response)
			client.Publish(msg.ReplyTo, payload)
		}()

		score, err := rep.Score(inst)
		if err != nil {
			response.Error = err.Error()
			return
		}

		response.Score = score
	})

	client.Subscribe(guid+".reserve_and_recast_vote", func(msg *yagnats.Message) {
		var inst types.Instance

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			panic(err)
		}

		response := types.VoteResult{
			Rep: guid,
		}

		defer func() {
			payload, _ := json.Marshal(response)
			client.Publish(msg.ReplyTo, payload)
		}()

		score, err := rep.ScoreThenTentativelyReserve(inst)
		if err != nil {
			response.Error = err.Error()
			return
		}

		response.Score = score
	})

	client.Subscribe(guid+".release", func(msg *yagnats.Message) {
		var inst types.Instance

		responsePayload := errorResponse
		defer func() {
			client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			log.Println(guid, "invalid reserve_and_recast_vote request:", err)
			return
		}

		rep.ReleaseReservation(inst) //need to handle error

		responsePayload = successResponse
	})

	client.Subscribe(guid+".claim", func(msg *yagnats.Message) {
		var inst types.Instance

		responsePayload := errorResponse
		defer func() {
			client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			log.Println(guid, "invalid reserve_and_recast_vote request:", err)
			return
		}

		rep.Claim(inst) //need to handle error

		responsePayload = successResponse
	})

	fmt.Printf("[%s] listening for nats\n", guid)

	select {}
}
