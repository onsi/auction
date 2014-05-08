package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/cloudfoundry/yagnats"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/representative"
)

var errorResponse = []byte("error")
var successResponse = []byte("ok")

var resources = flag.Int("resources", 100, "total available resources")
var guid = flag.String("guid", "", "guid")
var natsAddr = flag.String("natsAddr", "127.0.0.1:4222", "nats server address")

func main() {
	flag.Parse()
	if *guid == "" {
		panic("can haz guid")
	}

	if *natsAddr == "" {
		panic("can haz nats addr")
	}

	client := yagnats.NewClient()

	err := client.Connect(&yagnats.ConnectionInfo{
		Addr: *natsAddr,
	})
	if err != nil {
		log.Fatalln("no nats:", err)
	}

	rep := representative.New(*guid, *resources, nil)

	client.Subscribe(*guid+".guid", func(msg *yagnats.Message) {
		jguid, _ := json.Marshal(rep.Guid())
		client.Publish(msg.ReplyTo, jguid)
	})

	client.Subscribe(*guid+".total_resources", func(msg *yagnats.Message) {
		jresources, _ := json.Marshal(rep.TotalResources())
		client.Publish(msg.ReplyTo, jresources)
	})

	client.Subscribe(*guid+".instances", func(msg *yagnats.Message) {
		jinstances, _ := json.Marshal(rep.Instances())
		client.Publish(msg.ReplyTo, jinstances)
	})

	client.Subscribe(*guid+".vote", func(msg *yagnats.Message) {
		var inst instance.Instance

		responsePayload := errorResponse
		defer func() {
			client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			log.Println("invalid vote request:", err)
			return
		}

		score, err := rep.Vote(inst)
		if err != nil {
			log.Println("failed to vote:", err)
			return
		}

		responsePayload, _ = json.Marshal(score)
	})

	client.Subscribe(*guid+".reserve_and_recast_vote", func(msg *yagnats.Message) {
		var inst instance.Instance

		responsePayload := errorResponse
		defer func() {
			client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			log.Println("invalid reserve_and_recast_vote request:", err)
			return
		}

		score, err := rep.ReserveAndRecastVote(inst)
		if err != nil {
			log.Println("failed to reserve_and_recast_vote:", err)
			return
		}

		responsePayload, _ = json.Marshal(score)
	})

	client.Subscribe(*guid+".release", func(msg *yagnats.Message) {
		var inst instance.Instance

		responsePayload := errorResponse
		defer func() {
			client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			log.Println("invalid reserve_and_recast_vote request:", err)
			return
		}

		rep.Release(inst)

		responsePayload = successResponse
	})

	client.Subscribe(*guid+".claim", func(msg *yagnats.Message) {
		var inst instance.Instance

		responsePayload := errorResponse
		defer func() {
			client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			log.Println("invalid reserve_and_recast_vote request:", err)
			return
		}

		rep.Claim(inst)

		responsePayload = successResponse
	})

	fmt.Printf("[%s] listening\n", *guid)

	select {}
}
