package auction_test

import (
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/util"
	"github.com/onsi/auction/visualization"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Ω

var _ = Describe("Auction", func() {
	var initialDistributions map[int][]instance.Instance

	generateUniqueInstances := func(numInstances int) []instance.Instance {
		instances := []instance.Instance{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, instance.New(util.NewGrayscaleGuid("BBB"), 1))
		}
		return instances
	}

	generateUniqueInitialInstances := func(numInstances int) []instance.Instance {
		instances := []instance.Instance{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, instance.New(util.NewGrayscaleGuid("AAA"), 1))
		}
		return instances
	}

	randomColor := func() string {
		return []string{"purple", "red", "cyan", "yellow", "gray"}[util.R.Intn(5)]
	}

	generateInstancesWithRandomColors := func(numInstances int) []instance.Instance {
		instances := []instance.Instance{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, instance.New(randomColor(), 1))
		}
		return instances
	}

	randomSVGColor := func() string {
		return []string{"purple", "red", "cyan", "teal", "gray", "blue", "pink", "green", "lime", "orange", "lightseagreen", "brown"}[util.R.Intn(12)]
	}

	generateInstancesWithRandomSVGColors := func(numInstances int) []instance.Instance {
		instances := []instance.Instance{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, instance.New(randomSVGColor(), 1))
		}
		return instances
	}

	generateInstancesForAppGuid := func(numInstances int, appGuid string) []instance.Instance {
		instances := []instance.Instance{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, instance.New(appGuid, 1))
		}
		return instances
	}

	generateNewColorInstances := func(newInstances map[string]int) []instance.Instance {
		instances := []instance.Instance{}
		for color, num := range newInstances {
			instances = append(instances, generateInstancesForAppGuid(num, color)...)
		}
		return instances
	}

	BeforeEach(func() {
		util.ResetGuids()
		initialDistributions = map[int][]instance.Instance{}
	})

	JustBeforeEach(func() {
		for index, instances := range initialDistributions {
			client.SetInstances(guids[index], instances)
		}
	})

	FDescribe("Experiments", func() {
		Context("Cold start scenario", func() {
			nexec := []int{25, 100}
			napps := []int{2000, 8000}
			for i := range nexec {
				i := i
				Context("with single-instance and multi-instance apps apps", func() {
					It("should distribute evenly", func() {
						instances := generateUniqueInstances(napps[i] / 2)
						instances = append(instances, generateInstancesWithRandomSVGColors(napps[i]/2)...)

						report := auctioneer.HoldAuctionsFor(client, instances, guids[:nexec[i]], rules, communicator)

						visualization.PrintReport(client, report.AuctionResults, guids[:nexec[i]], report.AuctionDuration, rules)

						svgReport.DrawReportCard(i, 0, report)
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
							initialDistributions[j] = generateUniqueInitialInstances(50)
						}
					})

					It("should distribute evenly", func() {
						instances := generateUniqueInstances(napps[i])

						report := auctioneer.HoldAuctionsFor(client, instances, guids[:nexec[i]], rules, communicator)

						visualization.PrintReport(client, report.AuctionResults, guids[:nexec[i]], report.AuctionDuration, rules)

						svgReport.DrawReportCard(i, 1, report)
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
							initialDistributions[j] = generateUniqueInitialInstances(util.RandomIntIn(78, 80))
						}
					})

					It("should distribute evenly", func() {
						instances := generateInstancesForAppGuid(napps[i], "red")

						report := auctioneer.HoldAuctionsFor(client, instances, guids[:nexec[i]], rules, communicator)

						visualization.PrintReport(client, report.AuctionResults, guids[:nexec[i]], report.AuctionDuration, rules)

						svgReport.DrawReportCard(i, 2, report)
					})
				})
			}
		})
	})

	Context("with non-empty representatives (and single-instance apps)", func() {
		var numApps int
		BeforeEach(func() {
			numApps = 1000

			for i := 0; i < numReps; i++ {
				initialDistributions[i] = generateUniqueInstances(util.R.Intn(60))
			}
		})

		It("should distribute evenly", func() {
			instances := generateUniqueInstances(numApps)

			report := auctioneer.HoldAuctionsFor(client, instances, guids, rules, communicator)

			visualization.PrintReport(client, report.AuctionResults, guids, report.AuctionDuration, rules)
		})
	})

	Context("something that looks like prod", func() {
		var numExistingApps int
		var numReps int
		var numDemoInstances int
		BeforeEach(func() {
			numExistingApps = 1337
			numReps = 26
			appsPerRep := numExistingApps / numReps
			numDemoInstances = 100

			for i := 0; i < numReps; i++ {
				initialDistributions[i] = generateUniqueInstances(util.R.Intn(appsPerRep))
			}
		})

		It("should distribute evenly when watters does a demo", func() {
			instances := generateInstancesForAppGuid(numDemoInstances, "red")

			report := auctioneer.HoldAuctionsFor(client, instances, guids, rules, communicator)

			visualization.PrintReport(client, report.AuctionResults, guids, report.AuctionDuration, rules)

			svgReport.DrawReportCard(visualization.ReportCardWidth, 0, report)
		})
	})

	Context("something very imbalanced", func() {
		var numReps int
		var numDemoInstances int
		BeforeEach(func() {
			numReps = 20
			numDemoInstances = 200

			for i := 0; i < numReps-1; i++ {
				initialDistributions[i] = generateUniqueInstances(50)
			}
		})

		It("should distribute evenly", func() {
			instances := generateUniqueInstances(numDemoInstances)

			report := auctioneer.HoldAuctionsFor(client, instances, guids, rules, communicator)

			visualization.PrintReport(client, report.AuctionResults, guids, report.AuctionDuration, rules)
		})
	})

	Context("apps with multiple instances", func() {
		var newInstances map[string]int

		Context("when starting from a (terrible) initial distribution", func() {
			BeforeEach(func() {
				newInstances = map[string]int{
					"red":    570,
					"purple": 420,
					"cyan":   500,
					"yellow": 720,
					"gray":   129,
				}

				for i := 0; i < numReps; i++ {
					initialDistributions[i] = generateInstancesWithRandomColors(util.R.Intn(60))
				}
			})

			It("should distribute evenly", func() {
				instances := generateNewColorInstances(newInstances)

				report := auctioneer.HoldAuctionsFor(client, instances, guids, rules, communicator)
				visualization.PrintReport(client, report.AuctionResults, guids, report.AuctionDuration, rules)
			})
		})

		Context("when starting from empty", func() {
			BeforeEach(func() {
				newInstances = map[string]int{
					"red":    1000,
					"purple": 750,
					"cyan":   500,
					"yellow": 250,
					"gray":   100,
				}
			})

			It("should distribute evently", func() {
				instances := generateNewColorInstances(newInstances)
				instances = append(instances, generateUniqueInstances(2000)...)

				report := auctioneer.HoldAuctionsFor(client, instances, guids, rules, communicator)
				visualization.PrintReport(client, report.AuctionResults, guids, report.AuctionDuration, rules)
			})
		})
	})
})
