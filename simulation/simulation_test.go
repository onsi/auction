package simulation_test

import (
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/simulation/visualization"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Ω

var _ = Describe("Auction", func() {
	var initialDistributions map[int][]types.Instance

	newInstance := func(appGuid string, memoryMB float64) types.Instance {
		return types.Instance{
			AppGuid:      appGuid,
			InstanceGuid: util.NewGuid("INS"),
			Resources: types.Resources{
				MemoryMB: memoryMB,
				DiskMB:   1,
			},
		}
	}

	generateUniqueInstances := func(numInstances int, memoryMB float64) []types.Instance {
		instances := []types.Instance{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, newInstance(util.NewGrayscaleGuid("BBB"), memoryMB))
		}
		return instances
	}

	generateUniqueInitialInstances := func(numInstances int, memoryMB float64) []types.Instance {
		instances := []types.Instance{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, newInstance(util.NewGrayscaleGuid("AAA"), memoryMB))
		}
		return instances
	}

	randomSVGColor := func() string {
		return []string{"purple", "red", "cyan", "teal", "gray", "blue", "pink", "green", "lime", "orange", "lightseagreen", "brown"}[util.R.Intn(12)]
	}

	generateInstancesWithRandomSVGColors := func(numInstances int, memoryMB float64) []types.Instance {
		instances := []types.Instance{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, newInstance(randomSVGColor(), memoryMB))
		}
		return instances
	}

	generateInstancesForAppGuid := func(numInstances int, appGuid string, memoryMB float64) []types.Instance {
		instances := []types.Instance{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, newInstance(appGuid, memoryMB))
		}
		return instances
	}

	BeforeEach(func() {
		util.ResetGuids()
		initialDistributions = map[int][]types.Instance{}
	})

	JustBeforeEach(func() {
		for index, instances := range initialDistributions {
			client.SetInstances(guids[index], instances)
		}
	})

	Describe("Experiments", func() {
		Context("Cold start scenario", func() {
			nexec := []int{25, 100}
			n1apps := []int{1800, 7000}
			n2apps := []int{200, 1000}
			n4apps := []int{50, 200}
			for i := range nexec {
				i := i
				Context("with single-instance and multi-instance apps apps", func() {
					It("should distribute evenly", func() {
						instances := []types.Instance{}

						instances = append(instances, generateUniqueInstances(n1apps[i]/2, 1)...)
						instances = append(instances, generateInstancesWithRandomSVGColors(n1apps[i]/2, 1)...)
						instances = append(instances, generateUniqueInstances(n2apps[i]/2, 2)...)
						instances = append(instances, generateInstancesWithRandomSVGColors(n2apps[i]/2, 2)...)
						instances = append(instances, generateUniqueInstances(n4apps[i]/2, 4)...)
						instances = append(instances, generateInstancesWithRandomSVGColors(n4apps[i]/2, 4)...)

						permutedInstances := make([]types.Instance, len(instances))
						for i, index := range util.R.Perm(len(instances)) {
							permutedInstances[i] = instances[index]
						}

						report := auctionDistributor.HoldAuctionsFor(instances, guids[:nexec[i]], auctioneer.DefaultRules)

						visualization.PrintReport(client, report.AuctionResults, guids[:nexec[i]], report.AuctionDuration, auctioneer.DefaultRules)

						svgReport.DrawReportCard(i, 0, report)
						reports = append(reports, report)
					})
				})
			}

		})

		Context("Imbalanced scenario (e.g. a deploy)", func() {
			nexec := []int{100, 100}
			nempty := []int{5, 1}
			napps := []int{500, 100}

			for i := range nexec {
				i := i
				Context("scenario", func() {
					BeforeEach(func() {
						for j := 0; j < nexec[i]-nempty[i]; j++ {
							initialDistributions[j] = generateUniqueInitialInstances(50, 1)
						}
					})

					It("should distribute evenly", func() {
						instances := generateUniqueInstances(napps[i], 1)

						report := auctionDistributor.HoldAuctionsFor(instances, guids[:nexec[i]], auctioneer.DefaultRules)

						visualization.PrintReport(client, report.AuctionResults, guids[:nexec[i]], report.AuctionDuration, auctioneer.DefaultRules)

						svgReport.DrawReportCard(i, 1, report)
						reports = append(reports, report)
					})
				})
			}
		})

		Context("The Watters demo", func() {
			nexec := []int{30, 100}
			napps := []int{200, 400}

			for i := range nexec {
				i := i

				Context("scenario", func() {
					BeforeEach(func() {
						for j := 0; j < nexec[i]; j++ {
							initialDistributions[j] = generateUniqueInitialInstances(util.RandomIntIn(78, 80), 1)
						}
					})

					It("should distribute evenly", func() {
						instances := generateInstancesForAppGuid(napps[i], "red", 1)

						report := auctionDistributor.HoldAuctionsFor(instances, guids[:nexec[i]], auctioneer.DefaultRules)

						visualization.PrintReport(client, report.AuctionResults, guids[:nexec[i]], report.AuctionDuration, auctioneer.DefaultRules)

						svgReport.DrawReportCard(i, 2, report)
						reports = append(reports, report)
					})
				})
			}
		})
	})
})
