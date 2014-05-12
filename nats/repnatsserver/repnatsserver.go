package repnatsserver

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/cloudfoundry/yagnats"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/representative"
	"github.com/onsi/auction/types"
)

var errorResponse = []byte("error")
var successResponse = []byte("ok")

func Start(natsAddrs []string, rep *representative.Representative) {
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

	client.Subscribe(guid+".guid", func(msg *yagnats.Message) {
		jguid, _ := json.Marshal(rep.Guid())
		client.Publish(msg.ReplyTo, jguid)
	})

	client.Subscribe(guid+".total_resources", func(msg *yagnats.Message) {
		jresources, _ := json.Marshal(rep.TotalResources())
		client.Publish(msg.ReplyTo, jresources)
	})

	client.Subscribe(guid+".reset", func(msg *yagnats.Message) {
		rep.Reset()
		client.Publish(msg.ReplyTo, successResponse)
	})

	client.Subscribe(guid+".set_instances", func(msg *yagnats.Message) {
		var instances []instance.Instance

		err := json.Unmarshal(msg.Payload, &instances)
		if err != nil {
			client.Publish(msg.ReplyTo, errorResponse)
		}

		rep.SetInstances(instances)
		client.Publish(msg.ReplyTo, successResponse)
	})

	client.Subscribe(guid+".instances", func(msg *yagnats.Message) {
		jinstances, _ := json.Marshal(rep.Instances())
		client.Publish(msg.ReplyTo, jinstances)
	})

	client.Subscribe(guid+".vote", func(msg *yagnats.Message) {
		var inst instance.Instance

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

		score, err := rep.Vote(inst)
		if err != nil {
			response.Error = err.Error()
			return
		}

		response.Score = score
	})

	client.Subscribe(guid+".reserve_and_recast_vote", func(msg *yagnats.Message) {
		var inst instance.Instance

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

		score, err := rep.ReserveAndRecastVote(inst)
		if err != nil {
			response.Error = err.Error()
			return
		}

		response.Score = score
	})

	client.Subscribe(guid+".release", func(msg *yagnats.Message) {
		var inst instance.Instance

		responsePayload := errorResponse
		defer func() {
			client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			log.Println(guid, "invalid reserve_and_recast_vote request:", err)
			return
		}

		rep.Release(inst)

		responsePayload = successResponse
	})

	client.Subscribe(guid+".claim", func(msg *yagnats.Message) {
		var inst instance.Instance

		responsePayload := errorResponse
		defer func() {
			client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			log.Println(guid, "invalid reserve_and_recast_vote request:", err)
			return
		}

		rep.Claim(inst)

		responsePayload = successResponse
	})

	fmt.Printf("[%s] listening for nats\n", guid)

	select {}
}
