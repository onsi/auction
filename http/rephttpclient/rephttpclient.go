package rephttpclient

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/types"
)

var semaphore chan bool
var MaxConcurrentConnections = 10

func init() {
	semaphore = make(chan bool, MaxConcurrentConnections)
}

type RepHTTPClient struct {
	endpoints map[string]string
	client    *http.Client
}

func New(endpoints map[string]string, timeout time.Duration) *RepHTTPClient {
	http.DefaultClient.Transport = &http.Transport{
		ResponseHeaderTimeout: timeout,
		DisableKeepAlives:     true,
	}

	return &RepHTTPClient{
		endpoints: endpoints,
		client:    http.DefaultClient,
	}
}

func (rep *RepHTTPClient) enter() {
	semaphore <- true
}

func (rep *RepHTTPClient) exit() {
	<-semaphore
}

func (rep *RepHTTPClient) TotalResources(guid string) int {
	rep.enter()
	defer rep.exit()

	resp, err := rep.client.Get(rep.endpoints[guid] + "/total_resources")
	if err != nil {
		panic("failed to get total resources!")
	}

	defer resp.Body.Close()

	var totalResources int
	err = json.NewDecoder(resp.Body).Decode(&totalResources)
	if err != nil {
		panic("invalid total resources: " + err.Error())
	}

	return totalResources
}

func (rep *RepHTTPClient) Instances(guid string) []instance.Instance {
	rep.enter()
	defer rep.exit()

	resp, err := rep.client.Get(rep.endpoints[guid] + "/instances")
	if err != nil {
		panic("failed to get instances!")
	}

	defer resp.Body.Close()

	var instances []instance.Instance
	err = json.NewDecoder(resp.Body).Decode(&instances)
	if err != nil {
		panic("invalid instances: " + err.Error())
	}

	return instances
}

func (rep *RepHTTPClient) Reset(guid string) {
	rep.enter()
	defer rep.exit()
	rep.client.Get(rep.endpoints[guid] + "/reset")
}

func (rep *RepHTTPClient) SetInstances(guid string, instances []instance.Instance) {
	rep.enter()
	defer rep.exit()

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(instances)
	if err != nil {
		println(err.Error())
		return
	}

	resp, err := rep.client.Post(rep.endpoints[guid]+"/set_instances", "application/json", body)
	if err != nil {
		println(err.Error())
		return
	}

	resp.Body.Close()
}

func (rep *RepHTTPClient) vote(guid string, instance instance.Instance, c chan types.VoteResult) {
	rep.enter()
	defer rep.exit()
	result := types.VoteResult{
		Rep: guid,
	}
	defer func() {
		c <- result
	}()

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(instance)
	if err != nil {
		result.Error = err.Error()
		return
	}

	resp, err := rep.client.Post(rep.endpoints[guid]+"/vote", "application/json", body)
	if err != nil {
		println(err.Error())
		result.Error = err.Error()
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Error = "failed"
		return
	}

	var score float64
	err = json.NewDecoder(resp.Body).Decode(&score)
	if err != nil {
		result.Error = err.Error()
		return
	}
	result.Score = score

	return
}

func (rep *RepHTTPClient) Vote(guids []string, instance instance.Instance) types.VoteResults {
	c := make(chan types.VoteResult)
	for _, guid := range guids {
		go rep.vote(guid, instance, c)
	}

	results := types.VoteResults{}
	for _ = range guids {
		results = append(results, <-c)
	}

	return results
}

func (rep *RepHTTPClient) reserveandRecastVote(guid string, instance instance.Instance, c chan types.VoteResult) {
	rep.enter()
	defer rep.exit()
	result := types.VoteResult{
		Rep: guid,
	}
	defer func() {
		c <- result
	}()

	body := new(bytes.Buffer)

	err := json.NewEncoder(body).Encode(instance)
	if err != nil {
		result.Error = err.Error()
		return
	}

	resp, err := rep.client.Post(rep.endpoints[guid]+"/reserve_and_recast_vote", "application/json", body)
	if err != nil {
		result.Error = err.Error()
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Error = "failed"
		return
	}

	var score float64
	err = json.NewDecoder(resp.Body).Decode(&score)
	if err != nil {
		result.Error = err.Error()
		return
	}

	result.Score = score
	return
}

func (rep *RepHTTPClient) ReserveAndRecastVote(guids []string, instance instance.Instance) types.VoteResults {
	c := make(chan types.VoteResult)
	for _, guid := range guids {
		go rep.reserveandRecastVote(guid, instance, c)
	}

	results := types.VoteResults{}
	for _ = range guids {
		results = append(results, <-c)
	}

	return results
}

func (rep *RepHTTPClient) release(guid string, instance instance.Instance) {
	rep.enter()
	defer rep.exit()

	body := new(bytes.Buffer)

	err := json.NewEncoder(body).Encode(instance)
	if err != nil {
		panic("failed to encode instance: " + err.Error())
	}

	resp, err := rep.client.Post(rep.endpoints[guid]+"/release", "application/json", body)
	if err != nil {
		return
	}

	resp.Body.Close()
}

func (rep *RepHTTPClient) Release(guids []string, instance instance.Instance) {
	c := make(chan bool)
	for _, guid := range guids {
		go func(guid string) {
			rep.release(guid, instance)
			c <- true
		}(guid)
	}
	for _ = range guids {
		<-c
	}
}

func (rep *RepHTTPClient) Claim(guid string, instance instance.Instance) {
	rep.enter()
	defer rep.exit()

	body := new(bytes.Buffer)

	err := json.NewEncoder(body).Encode(instance)
	if err != nil {
		panic("failed to encode instance: " + err.Error())
	}

	resp, err := rep.client.Post(rep.endpoints[guid]+"/claim", "application/json", body)
	if err != nil {
		return
	}

	resp.Body.Close()
}
