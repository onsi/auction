set -e -x
mkdir -p ./runs

ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_revote -maxBiddingPool=20 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=20 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=20 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_best -maxBiddingPool=20 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=20 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=20 -maxConcurrent=20

ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_revote -maxBiddingPool=100 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=100 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=100 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_best -maxBiddingPool=100 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=100 -maxConcurrent=20
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=100 -maxConcurrent=20

ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_revote -maxBiddingPool=20 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=20 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=20 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_best -maxBiddingPool=20 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=20 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=20 -maxConcurrent=100

ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_revote -maxBiddingPool=20 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=20 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=20 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_best -maxBiddingPool=20 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=20 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=20 -maxConcurrent=1000

ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_revote -maxBiddingPool=100 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=100 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=100 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_best -maxBiddingPool=100 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=100 -maxConcurrent=100
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=100 -maxConcurrent=100

ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_revote -maxBiddingPool=100 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=all_reserve -maxBiddingPool=100 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=reserve_n_best -maxBiddingPool=100 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_best -maxBiddingPool=100 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=random -maxBiddingPool=100 -maxConcurrent=1000
ginkgo -- -communicationMode=ketchup-nats -auctioneerMode=remote -algorithm=pick_among_best -maxBiddingPool=100 -maxConcurrent=1000
