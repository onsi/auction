package representative

import (
	"errors"
	"sync"
	"time"

	"github.com/cloudfoundry/storeadapter"
	"github.com/onsi/auction/instance"
)

var InsufficientResources = errors.New("insufficient resources for instance")

type Representative struct {
	guid           string
	lock           *sync.Mutex
	instances      map[string]instance.Instance
	totalResources int

	store storeadapter.StoreAdapter
}

func New(store storeadapter.StoreAdapter, guid string, totalResources int) *Representative {
	return &Representative{
		guid:           guid,
		totalResources: totalResources,

		store: store,

		lock:      &sync.Mutex{},
		instances: map[string]instance.Instance{},
	}
}

func (rep *Representative) Guid() string {
	return rep.guid
}

func (rep *Representative) TotalResources() int {
	return rep.totalResources
}

func (rep *Representative) Reset() {
	rep.lock.Lock()
	defer rep.lock.Unlock()
	rep.instances = map[string]instance.Instance{}
}

func (rep *Representative) SetInstances(instances []instance.Instance) {
	rep.lock.Lock()
	defer rep.lock.Unlock()
	instancesMap := map[string]instance.Instance{}
	for _, instance := range instances {
		instancesMap[instance.InstanceGuid] = instance
	}

	rep.instances = instancesMap
}

func (rep *Representative) Instances() []instance.Instance {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	result := []instance.Instance{}
	for _, instance := range rep.instances {
		result = append(result, instance)
	}
	return result
}

func (rep *Representative) Vote(instance instance.Instance) (float64, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	if !rep.hasRoomFor(instance) {
		return 0, InsufficientResources
	}
	return rep.score(instance), nil
}

func (rep *Representative) HesitateAndClaim(instance instance.Instance) error {
	rep.lock.Lock()
	if !rep.hasRoomFor(instance) {
		rep.lock.Unlock()
		return InsufficientResources
	}

	resources := int(float64(rep.usedResources()) / float64(rep.totalResources) * 10)
	nInstances := rep.numberOfInstancesForAppGuid(instance.AppGuid)
	// resources := rep.usedResources()
	instance.Tentative = true
	rep.instances[instance.InstanceGuid] = instance
	rep.lock.Unlock()

	sleep(time.Duration(nInstances) * time.Millisecond)
	sleep(time.Duration(resources) * time.Millisecond)
	// time.Sleep(time.Duration(rep.usedResources()) * time.Millisecond)

	if !rep.claim(instance) {
		rep.lock.Lock()
		delete(rep.instances, instance.InstanceGuid)
		rep.lock.Unlock()
		return errors.New("failed to claim")
	}

	rep.lock.Lock()
	instance.Tentative = false
	rep.instances[instance.InstanceGuid] = instance
	rep.lock.Unlock()

	return nil
}

func sleep(duration time.Duration) {
	if duration == 0 {
		// just to prevent any scheduling trickery...
		return
	}

	// log.Println("hestitating", duration)
	time.Sleep(duration)
}

func (rep *Representative) claim(instance instance.Instance) bool {
	err := rep.store.CompareAndSwap(storeadapter.StoreNode{
		Key:   "/apps/" + instance.AppGuid + "/" + instance.InstanceGuid,
		Value: []byte("marco"),
	}, storeadapter.StoreNode{
		Key:   "/apps/" + instance.AppGuid + "/" + instance.InstanceGuid,
		Value: []byte("polo"),
	})
	return err == nil
}

func (rep *Representative) ReserveAndRecastVote(instance instance.Instance) (float64, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	if !rep.hasRoomFor(instance) {
		return 0, InsufficientResources
	}

	score := rep.score(instance) //recompute score *first*
	instance.Tentative = true
	rep.instances[instance.InstanceGuid] = instance //*then* make reservation

	return score, nil
}

func (rep *Representative) Release(instance instance.Instance) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	reservedInstance, ok := rep.instances[instance.InstanceGuid]
	if !(ok && reservedInstance.Tentative) {
		panic("wat?")
	}

	delete(rep.instances, instance.InstanceGuid)
}

func (rep *Representative) Claim(instance instance.Instance) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	_, ok := rep.instances[instance.InstanceGuid]
	if !ok {
		panic("wat?")
	}

	instance.Tentative = false
	rep.instances[instance.InstanceGuid] = instance
}

// internals -- no locks here the operations above should be atomic

func (rep *Representative) hasRoomFor(instance instance.Instance) bool {
	return rep.usedResources()+instance.RequiredResources <= rep.totalResources
}

func (rep *Representative) score(instance instance.Instance) float64 {
	fResources := float64(rep.usedResources()) / float64(rep.totalResources)
	nInstances := rep.numberOfInstancesForAppGuid(instance.AppGuid)

	return fResources + float64(nInstances)
}

func (rep *Representative) usedResources() int {
	usedResources := 0
	for _, instance := range rep.instances {
		usedResources += instance.RequiredResources
	}

	return usedResources
}

func (rep *Representative) numberOfInstancesForAppGuid(guid string) int {
	n := 0
	for _, instance := range rep.instances {
		if instance.AppGuid == guid {
			n += 1
		}
	}
	return n
}

//
