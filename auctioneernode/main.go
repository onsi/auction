package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
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
var httpAddr = flag.String("httpAddr", "0.0.0.0:48710", "http address to listen on")

var errorResponse = []byte("error")

func main() {
	flag.Parse()

	if *natsAddrs == "" {
		panic("need nats addr")
	}

	if *httpAddr == "" {
		panic("need http addr")
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

	http.HandleFunc("/auction", func(w http.ResponseWriter, r *http.Request) {
		semaphore <- true
		defer func() {
			<-semaphore
		}()

		var auctionRequest types.AuctionRequest
		err := json.NewDecoder(r.Body).Decode(&auctionRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		auctionResult := auctioneer.Auction(etcdAdapter, repclient, auctionRequest)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(auctionResult)
	})

	fmt.Println("auctioneering")

	panic(http.ListenAndServe(*httpAddr, nil))
}
