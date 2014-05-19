# Auction

- [X] Auctioneer
- [X] Representative
- [X] Instance
- [X] In-process algorithm simulation
- [X] Textual visualization
- [X] Naive HTTP (too many open files)
- [X] Naive NATS topology (pub/sub per rep per score)
- [X] Collated NATS topology (one pub/sub per score across all reps)
- [X] Pull out common client interface
- [] Organize by communication medium
- [] One rep binary for all communication
- [] All client methods should return an error
- [] Refactor tests
- [] Separate Auctioneer (server) from AuctionDistributer (client)
- [] Move suites into communication medium and extract formatting into a visualization package
- [] Text visualization can be written without reference to reps?
- [] Implement connection limiting (semaphore) in http client and server
- [] RPC ?
- [] Bosh deploy representatives that support all three protocols
- [] Run trials on AWS
- [] Improved visualizations (animations?)
- [] Test suite that runs different scenarios and actually fails if the distribution is invalid.
- [] Identify entry points for using the Auctioneer package and Representative package in actual components.
    - Ideally the algorithm is safely locked up and tested in the Auction repo.
    - Consumers simply provide methods that the Auction repo guarantees are called correctly (order really matters here!)

Explorations

- Optimizations
    - [X] Limit Rep bidding pool (value of choice: 20)
    - [X] Limit max number of concurrenct auctions (value of choice: 20)
    - [] Repick bidding pool between rounds
    - [] Limit second-round score to top-3 (?) bidders
- Scoring functions
    - [X] Memory
    - [X] App Distribution
    - [] Memory & Disk
- Scenarios
    [X] Empty reps => large swarm of starts of 1-instance apps
    [X] Non-empty reps (poor distribution) => large swarm of starts of 1-instance apps
    [X] Empty reps => N multi-instance apps
    [X] Seed reps with N multi-instance apps, then deploy M more
    [X] Seed subset of reps with N multi-instance apps, then deploy M more across all reps
    [] Many very full reps and few empty ones - how does this play with MaxConcurrent.  Is it improved by repicking between rounds?