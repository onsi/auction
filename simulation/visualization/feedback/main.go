package main

import (
	"encoding/json"
	"io/ioutil"
	"os/exec"

	"github.com/onsi/auction/types"
	"github.com/onsi/auction/visualization"
)

func main() {
	var report types.Report
	data, _ := ioutil.ReadFile("./test.json")
	json.Unmarshal(data, &report)

	svgReport := visualization.StartSVGReport("./test.svg", visualization.ReportCardWidth, visualization.ReportCardHeight)

	svgReport.DrawReportCard(0, 0, &report)

	svgReport.Done()

	exec.Command("open", "-a", "safari", "./test.svg").Run()
}
