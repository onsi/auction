package repnatsclient

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/cloudfoundry/yagnats"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
)

var TimeoutError = errors.New("timeout")
var RequestFailedError = errors.New("request failed")

type RepNatsClient struct {
	client  yagnats.NATSClient
	timeout time.Duration
}

func New(client yagnats.NATSClient, timeout time.Duration) *RepNatsClient {
	return &RepNatsClient{
		client:  client,
		timeout: timeout,
	}
}

func (rep *RepNatsClient) publishWithTimeout(guid string, subject string, req interface{}, resp interface{}) (err error) {
	replyTo := util.RandomGuid()
	c := make(chan []byte, 1)

	_, err = rep.client.Subscribe(replyTo, func(msg *yagnats.Message) {
		c <- msg.Payload
	})
	if err != nil {
		return err
	}

	payload := []byte{}
	if req != nil {
		payload, err = json.Marshal(req)
		if err != nil {
			return err
		}
	}

	rep.client.PublishWithReplyTo(guid+"."+subject, replyTo, payload)

	select {
	case payload := <-c:
		if string(payload) == "error" {
			return RequestFailedError
		}

		if resp != nil {
			return json.Unmarshal(payload, resp)
		}

		return nil

	case <-time.After(rep.timeout):
		// rep.client.Unsubscribe(sid)
		return TimeoutError
	}
}

func (rep *RepNatsClient) TotalResources(guid string) int {
	var totalResources int
	err := rep.publishWithTimeout(guid, "total_resources", nil, &totalResources)
	if err != nil {
		panic(err)
	}

	return totalResources
}

func (rep *RepNatsClient) HesitateAndClaim(guids []string, instance instance.Instance) types.VoteResults {
	replyTo := util.RandomGuid()

	response := make(chan types.VoteResult, 1)

	_, err := rep.client.Subscribe(replyTo, func(msg *yagnats.Message) {
		var result types.VoteResult
		err := json.Unmarshal(msg.Payload, &result)
		if err != nil {
			return
		}

		response <- result
	})

	if err != nil {
		return types.VoteResults{}
	}

	payload, _ := json.Marshal(instance)

	for _, guid := range guids {
		rep.client.PublishWithReplyTo(guid+".hesitate_and_claim", replyTo, payload)
	}

	select {
	case res := <-response:
		return types.VoteResults{res}
	case <-time.After(rep.timeout):
		println("TIMING OUT!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	}

	return types.VoteResults{}
}

func (rep *RepNatsClient) Instances(guid string) []instance.Instance {
	var instances []instance.Instance
	err := rep.publishWithTimeout(guid, "instances", nil, &instances)
	if err != nil {
		panic(err)
	}

	return instances
}

func (rep *RepNatsClient) Reset(guid string) {
	err := rep.publishWithTimeout(guid, "reset", nil, nil)
	if err != nil {
		panic(err)
	}
}

func (rep *RepNatsClient) SetInstances(guid string, instances []instance.Instance) {
	err := rep.publishWithTimeout(guid, "set_instances", instances, nil)
	if err != nil {
		panic(err)
	}
}

func (rep *RepNatsClient) batch(subject string, guids []string, instance instance.Instance) types.VoteResults {
	replyTo := util.RandomGuid()

	allReceived := new(sync.WaitGroup)
	responses := make(chan types.VoteResult, len(guids))

	_, err := rep.client.Subscribe(replyTo, func(msg *yagnats.Message) {
		defer allReceived.Done()
		var result types.VoteResult
		err := json.Unmarshal(msg.Payload, &result)
		if err != nil {
			return
		}

		responses <- result
	})

	if err != nil {
		return types.VoteResults{}
	}

	payload, _ := json.Marshal(instance)

	allReceived.Add(len(guids))

	for _, guid := range guids {
		rep.client.PublishWithReplyTo(guid+"."+subject, replyTo, payload)
	}

	done := make(chan struct{})
	go func() {
		allReceived.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(rep.timeout):
		println("TIMING OUT!!")
	}

	results := types.VoteResults{}

	for {
		select {
		case res := <-responses:
			results = append(results, res)
		default:
			return results
		}
	}

	return results
}

func (rep *RepNatsClient) Vote(guids []string, instance instance.Instance) types.VoteResults {
	return rep.batch("vote", guids, instance)
}

func (rep *RepNatsClient) ReserveAndRecastVote(guids []string, instance instance.Instance) types.VoteResults {
	return rep.batch("reserve_and_recast_vote", guids, instance)
}

func (rep *RepNatsClient) Release(guids []string, instance instance.Instance) {
	replyTo := util.RandomGuid()

	allReceived := new(sync.WaitGroup)

	_, err := rep.client.Subscribe(replyTo, func(msg *yagnats.Message) {
		allReceived.Done()
	})

	if err != nil {
		return
	}

	payload, _ := json.Marshal(instance)

	allReceived.Add(len(guids))

	for _, guid := range guids {
		rep.client.PublishWithReplyTo(guid+".release", replyTo, payload)
	}

	done := make(chan struct{})
	go func() {
		allReceived.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(rep.timeout):
		println("TIMING OUT!!")
	}
}

func (rep *RepNatsClient) Claim(guid string, instance instance.Instance) {
	err := rep.publishWithTimeout(guid, "claim", instance, nil)
	if err != nil {
		log.Println("failed to claim:", err)
	}
}
