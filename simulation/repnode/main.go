package main

import (
	"flag"
	"strings"
	"github.com/onsi/auction/nats/repnatsserver"
	"github.com/onsi/auction/rabbit/reprabbitserver"
	"github.com/onsi/auction/representative"
)

var resources = flag.Int("resources", 100, "total available resources")
var guid = flag.String("guid", "", "guid")
var natsAddrs = flag.String("natsAddrs", "", "nats server addresses")
var rabbitAddr = flag.String("rabbitAddr", "", "rabbit server address")

func main() {
	flag.Parse()

	if *guid == "" {
		panic("need guid")
	}

	if *natsAddrs == "" && *rabbitAddr == "" {
		panic("need nats or rabbit addr")
	}

	rep := representative.New(*guid, *resources)

	if *natsAddrs != "" {
		go repnatsserver.Start(strings.Split(*natsAddrs, ","), rep)
	}

	if *rabbitAddr != "" {
		go reprabbitserver.Start(*rabbitAddr, rep)
	}

	select {}
}
