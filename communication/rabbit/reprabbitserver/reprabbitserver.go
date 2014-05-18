package reprabbitserver

import (
	"encoding/json"
	"fmt"

	"github.com/onsi/auction/auctionrep"
	"github.com/onsi/auction/communication/rabbit/rabbitclient"
	"github.com/onsi/auction/types"
)

var errorResponse = []byte("error")
var successResponse = []byte("ok")

func Start(rabbitUrl string, rep auctionrep.AuctionRep) {
	println("RABBIT", rabbitUrl)
	server := rabbitclient.NewServer(rep.Guid(), rabbitUrl)
	err := server.ConnectAndEstablish()
	if err != nil {
		panic(err)
	}

	testAuctionRep := func() auctionrep.TestAuctionRep {
		tar, ok := rep.(auctionrep.TestAuctionRep)
		if !ok {
			panic("attempting to do a test-like thing with a non-test-like rep")
		}

		return tar
	}

	server.Handle("total_resources", func(_ []byte) []byte {
		out, _ := json.Marshal(testAuctionRep().TotalResources())
		return out
	})

	server.Handle("reset", func(_ []byte) []byte {
		testAuctionRep().Reset()
		return successResponse
	})

	server.Handle("set_instances", func(req []byte) []byte {
		var instances []types.Instance

		err := json.Unmarshal(req, &instances)
		if err != nil {
			return errorResponse
		}

		testAuctionRep().SetInstances(instances)
		return successResponse
	})

	server.Handle("instances", func(_ []byte) []byte {
		out, _ := json.Marshal(testAuctionRep().Instances())
		return out
	})

	server.Handle("vote", func(req []byte) []byte {
		var inst types.Instance

		err := json.Unmarshal(req, &inst)
		if err != nil {
			return errorResponse
		}

		response := types.VoteResult{
			Rep: rep.Guid(),
		}

		score, err := rep.Score(inst)
		if err != nil {
			response.Error = err.Error()
		} else {
			response.Score = score
		}

		out, _ := json.Marshal(response)
		return out
	})

	server.Handle("reserve_and_recast_vote", func(req []byte) []byte {
		var inst types.Instance

		err := json.Unmarshal(req, &inst)
		if err != nil {
			return errorResponse
		}

		response := types.VoteResult{
			Rep: rep.Guid(),
		}

		score, err := rep.ScoreThenTentativelyReserve(inst)
		if err != nil {
			response.Error = err.Error()
		} else {
			response.Score = score
		}

		out, _ := json.Marshal(response)
		return out
	})

	server.Handle("release", func(req []byte) []byte {
		var instance types.Instance

		err := json.Unmarshal(req, &instance)
		if err != nil {
			return errorResponse
		}

		rep.ReleaseReservation(instance)

		return successResponse
	})

	server.Handle("claim", func(req []byte) []byte {
		var instance types.Instance

		err := json.Unmarshal(req, &instance)
		if err != nil {
			return errorResponse
		}

		rep.Claim(instance)

		return successResponse
	})

	fmt.Printf("[%s] listening for rabbit\n", rep.Guid())

	select {}
}
