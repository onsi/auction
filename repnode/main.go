package main

import (
	"flag"
	"strings"

	"github.com/cloudfoundry/storeadapter/etcdstoreadapter"
	"github.com/cloudfoundry/storeadapter/workerpool"
	"github.com/onsi/auction/nats/repnatsserver"
	"github.com/onsi/auction/representative"
)

var resources = flag.Int("resources", 100, "total available resources")
var guid = flag.String("guid", "", "guid")
var natsAddrs = flag.String("natsAddrs", "", "nats server addresses")
var etcdCluster = flag.String(
	"etcdCluster",
	"http://127.0.0.1:4001",
	"comma-separated list of etcd addresses (http://ip:port)",
)

func main() {
	flag.Parse()

	if *guid == "" {
		panic("need guid")
	}

	if *natsAddrs == "" {
		panic("need nats addr")
	}

	etcdAdapter := etcdstoreadapter.NewETCDStoreAdapter(
		strings.Split(*etcdCluster, ","),
		workerpool.NewWorkerPool(30),
	)
	err := etcdAdapter.Connect()
	if err != nil {
		panic(err)
	}

	rep := representative.New(etcdAdapter, *guid, *resources)

	if *natsAddrs != "" {
		go repnatsserver.Start(strings.Split(*natsAddrs, ","), rep)
	}

	select {}
}
