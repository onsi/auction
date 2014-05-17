package reprabbitclient

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/onsi/auction/rabbitclient"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
)

var TimeoutError = errors.New("timeout")
var RequestFailedError = errors.New("request failed")

type RepRabbitClient struct {
	client  rabbitclient.RabbitClientInterface
	timeout time.Duration
}

func New(rabbitUrl string, timeout time.Duration) *RepRabbitClient {
	guid := util.RandomGuid()
	client := rabbitclient.NewClient(guid, rabbitUrl)
	err := client.ConnectAndEstablish()
	if err != nil {
		panic(err)
	}

	return &RepRabbitClient{
		client:  client,
		timeout: timeout,
	}
}

func (rep *RepRabbitClient) request(guid string, subject string, req interface{}, resp interface{}) (err error) {
	payload := []byte{}
	if req != nil {
		payload, err = json.Marshal(req)
		if err != nil {
			return err
		}
	}

	response, err := rep.client.Request(guid, subject, payload, rep.timeout)

	if err != nil {
		return err
	}

	if string(response) == "error" {
		return RequestFailedError
	}

	if resp != nil {
		return json.Unmarshal(response, resp)
	}

	return nil
}

func (rep *RepRabbitClient) TotalResources(guid string) int {
	var totalResources int
	err := rep.request(guid, "total_resources", []byte{}, &totalResources)
	if err != nil {
		panic(err)
	}
	return totalResources
}

func (rep *RepRabbitClient) Instances(guid string) []types.Instance {
	var instances []types.Instance
	err := rep.request(guid, "instances", nil, &instances)
	if err != nil {
		panic(err)
	}

	return instances
}

func (rep *RepRabbitClient) Reset(guid string) {
	err := rep.request(guid, "reset", nil, nil)
	if err != nil {
		panic(err)
	}
}

func (rep *RepRabbitClient) SetInstances(guid string, instances []types.Instance) {
	err := rep.request(guid, "set_instances", instances, nil)
	if err != nil {
		panic(err)
	}
}

func (rep *RepRabbitClient) batch(subject string, guids []string, instance types.Instance) types.VoteResults {
	c := make(chan types.VoteResult)
	for _, guid := range guids {
		go func(guid string) {
			var response types.VoteResult
			err := rep.request(guid, subject, instance, &response)
			if err != nil {
				c <- types.VoteResult{
					Error: err.Error(),
				}
			}
			c <- response
		}(guid)
	}

	votes := types.VoteResults{}
	for _ = range guids {
		votes = append(votes, <-c)
	}

	return votes
}

func (rep *RepRabbitClient) Vote(guids []string, instance types.Instance) types.VoteResults {
	return rep.batch("vote", guids, instance)
}

func (rep *RepRabbitClient) ReserveAndRecastVote(guids []string, instance types.Instance) types.VoteResults {
	return rep.batch("reserve_and_recast_vote", guids, instance)
}

func (rep *RepRabbitClient) Release(guids []string, instance types.Instance) {
	allReceived := new(sync.WaitGroup)
	allReceived.Add(len(guids))
	for _, guid := range guids {
		go func(guid string) {
			rep.request(guid, "release", instance, nil)
			allReceived.Done()
		}(guid)
	}

	allReceived.Wait()
}

func (rep *RepRabbitClient) Claim(guid string, instance types.Instance) {
	err := rep.request(guid, "claim", instance, nil)
	if err != nil {
		log.Println("failed to claim:", err)
	}
}
