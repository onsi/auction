package repnatsclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

func (rep *RepNatsClient) Vote(guids []string, instance instance.Instance) types.VoteResults {
	replyTo := util.RandomGuid()

	allReceived := new(sync.WaitGroup)
	responses := make(chan types.VoteResult, len(guids))

	buffer := &bytes.Buffer{}
	lock := &sync.Mutex{}
	_, err := rep.client.Subscribe(replyTo, func(msg *yagnats.Message) {
		defer func() {
			if r := recover(); r != nil {
				lock.Lock()
				fmt.Println(buffer)
				lock.Unlock()
				panic(r)
			}
		}()
		defer allReceived.Done()
		var result types.VoteResult
		err := json.Unmarshal(msg.Payload, &result)
		if err != nil {
			return
		}

		lock.Lock()
		fmt.Fprintf(buffer, "REC: %s %s %s\n", replyTo, msg.Subject, result.Rep)
		lock.Unlock()
		responses <- result
	})

	if err != nil {
		return types.VoteResults{}
	}

	payload, _ := json.Marshal(instance)

	allReceived.Add(len(guids))

	for _, guid := range guids {
		lock.Lock()
		fmt.Fprintf(buffer, "REQ: %s %s\n", guid, replyTo)
		lock.Unlock()
		rep.client.PublishWithReplyTo(guid+".vote", replyTo, payload)
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

func (rep *RepNatsClient) ReserveAndRecastVote(guid string, instance instance.Instance) (result types.VoteResult) {
	result.Rep = guid

	var score float64
	err := rep.publishWithTimeout(guid, "reserve_and_recast_vote", instance, &score)
	if err != nil {
		result.Error = err.Error()
		return
	}

	result.Score = score
	return
}

func (rep *RepNatsClient) Release(guid string, instance instance.Instance) {
	err := rep.publishWithTimeout(guid, "release", instance, nil)
	if err != nil {
		log.Println("failed to release:", err)
	}
}

func (rep *RepNatsClient) Claim(guid string, instance instance.Instance) {
	err := rep.publishWithTimeout(guid, "claim", instance, nil)
	if err != nil {
		log.Println("failed to claim:", err)
	}
}
