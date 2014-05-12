package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cloudfoundry/storeadapter/etcdstoreadapter"
	"github.com/cloudfoundry/storeadapter/workerpool"
	"github.com/cloudfoundry/yagnats"
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/nats/repnatsclient"
	"github.com/onsi/auction/types"
)

var natsAddrs = flag.String("natsAddrs", "", "nats server addresses")
var timeout = flag.Duration("timeout", 500*time.Millisecond, "timeout for entire auction")
var maxConcurrent = flag.Int("maxConcurrent", 1000, "number of concurrent auctions to hold")
var etcdCluster = flag.String(
	"etcdCluster",
	"http://127.0.0.1:4001",
	"comma-separated list of etcd addresses (http://ip:port)",
)

var errorResponse = []byte("error")

func main() {
	flag.Parse()

	if *natsAddrs == "" {
		panic("need either nats addr")
	}

	client := yagnats.NewClient()

	clusterInfo := &yagnats.ConnectionCluster{}

	for _, addr := range strings.Split(*natsAddrs, ",") {
		clusterInfo.Members = append(clusterInfo.Members, &yagnats.ConnectionInfo{
			Addr: addr,
		})
	}

	err := client.Connect(clusterInfo)

	if err != nil {
		log.Fatalln("no nats:", err)
	}

	semaphore := make(chan bool, *maxConcurrent)

	etcdAdapter := etcdstoreadapter.NewETCDStoreAdapter(
		strings.Split(*etcdCluster, ","),
		workerpool.NewWorkerPool(30),
	)
	err = etcdAdapter.Connect()
	if err != nil {
		panic(err)
	}

	repclient := repnatsclient.New(client, *timeout)

	client.SubscribeWithQueue("diego.auction", "auction-channel", func(msg *yagnats.Message) {
		semaphore <- true
		defer func() {
			<-semaphore
		}()

		var auctionRequest types.AuctionRequest
		err := json.Unmarshal(msg.Payload, &auctionRequest)
		if err != nil {
			client.Publish(msg.ReplyTo, errorResponse)
			return
		}

		auctionResult := auctioneer.Auction(etcdAdapter, repclient, auctionRequest)
		payload, _ := json.Marshal(auctionResult)

		client.Publish(msg.ReplyTo, payload)
	})

	fmt.Println("auctioneering")

	select {}
}
