package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/onsi/auction/types"
)

func main() {
	algorithms := []string{
		"random",
		"pick_best",
		// "pick_among_best",
		"reserve_n_best",
		// "all_reserve",
		"all_revote",
	}
	out := "<html><head></head><body><table>"
	// for _, comm := range []string{"inprocess", "nats", "ketchup"} {
	for _, comm := range []string{"ketchup"} {
		out += "<tr>"
		out += "<td></td>"
		for _, alg := range algorithms {
			out += "<th>" + alg + "</th>"
		}
		out += "</tr>"
		// for _, poolConc := range [][]int{{20, 20}, {100, 20}, {20, 100}, {100, 100}, {20, 1000}, {100, 1000}} {
		for _, poolConc := range [][]int{{20, 20}, {100, 20}, {20, 100}, {100, 100}} {
			out += "<tr>"
			out += fmt.Sprintf("<th>%s<br>%d Bidders<br>%d Concurrently</th>", comm, poolConc[0], poolConc[1])
			for _, alg := range algorithms {
				fmt.Println(comm, alg, poolConc)
				fname := fmt.Sprintf("../imac/%s_%s_pool%d_conc%d", alg, comm, poolConc[0], poolConc[1])
				_, err := os.Stat(fname + ".json")
				if err != nil {
					out += "<td>"
					out += "</td>"
					continue
				}
				data, _ := ioutil.ReadFile(fname + ".json")
				reports := []*types.Report{}
				json.Unmarshal(data, &reports)
				scores := 0.0
				communication := 0.0
				waitTimes := 0.0
				for _, report := range reports {
					waitTimes += report.AuctionDuration.Seconds()
					communication += report.CommStats().Total
					scores += report.DistributionScore()
				}

				out += "<td>"
				out += fmt.Sprintf(`<a href="../imac/%s.svg">`, fname)
				out += fmt.Sprintf(`<div style="background-color:%s;">%.3f</div>`, scoreColor(scores), scores)
				out += fmt.Sprintf(`<div style="background-color:%s;">%.2f</div>`, waitColor(waitTimes), waitTimes)
				out += fmt.Sprintf("<div>%d</div>", int(communication))
				out += "</a>"
				out += "</td>"
			}
			out += "</tr>"
		}
		out += "<tr><td></td></tr>"
	}
	out += "</table></body></html>"
	ioutil.WriteFile("./present.html", []byte(out), 0777)
}

func scoreColor(score float64) string {
	scaled := 1 - score/0.3 //0 is great (white), 0.3 is worst (red)
	rg := 80 + scaled*(255-80)
	if rg < 0 {
		rg = 0
	}
	return fmt.Sprintf("rgb(255, %d, %d)", int(rg), int(rg))
}

func waitColor(waitTime float64) string {
	scaled := 1 - waitTime/120.0 //0 is great (white), 60s is worst (red)
	rg := 80 + scaled*(255-80)
	if rg < 0 {
		rg = 0
	}
	return fmt.Sprintf("rgb(255, %d, %d)", int(rg), int(rg))
}
