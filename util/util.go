package util

import (
	crand "crypto/rand"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var R *rand.Rand
var guidTracker map[string]int
var lock *sync.Mutex

func init() {
	R = rand.New(rand.NewSource(time.Now().UnixNano()))
	ResetGuids()
	lock = &sync.Mutex{}
}

func ResetGuids() {
	guidTracker = map[string]int{}
}

func NewGuid(prefix string) string {
	guidTracker[prefix] = guidTracker[prefix] + 1
	return fmt.Sprintf("%s-%d", prefix, guidTracker[prefix])
}

func NewGrayscaleGuid(prefix string) string {
	guidTracker[prefix] = guidTracker[prefix] + 1
	col := R.Intn(200)
	return fmt.Sprintf("%s-%d-%s", prefix, guidTracker[prefix], rgb(col, col, col))
}

func rgb(r int, g int, b int) string {
	return fmt.Sprintf("rgb(%d,%d,%d)", r, g, b)
}

func RandomIntIn(min, max int) int {
	return R.Intn(max-min+1) + min
}

func RandomGuid() string {
	b := make([]byte, 8)
	lock.Lock()
	_, err := crand.Read(b)
	lock.Unlock()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x", b[0:2], b[2:4], b[4:6], b[6:8])
}

func RandomSleep(min time.Duration, max time.Duration, timeout time.Duration) bool {
	sleepDuration := time.Duration(R.Float64()*float64(max-min) + float64(min))
	if sleepDuration <= timeout {
		time.Sleep(sleepDuration)
		return true
	} else {
		time.Sleep(timeout)
		return false
	}
}

func Flake(fraction float64) bool {
	return R.Float64() <= fraction
}

func RandomFrom(things ...string) string {
	return things[R.Intn(len(things))]
}
