package representative

import (
	"errors"
	"sync"

	"github.com/onsi/auction/types"
)

var InsufficientResources = errors.New("insufficient resources for instance")

type Representative struct {
	guid           string
	lock           *sync.Mutex
	instances      map[string]types.Instance
	totalResources int
}

func New(guid string, totalResources int) *Representative {
	return &Representative{
		guid:           guid,
		totalResources: totalResources,

		lock:      &sync.Mutex{},
		instances: map[string]types.Instance{},
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
	rep.instances = map[string]types.Instance{}
}

func (rep *Representative) SetInstances(instances []types.Instance) {
	rep.lock.Lock()
	defer rep.lock.Unlock()
	instancesMap := map[string]types.Instance{}
	for _, instance := range instances {
		instancesMap[instance.InstanceGuid] = instance
	}

	rep.instances = instancesMap
}

func (rep *Representative) Instances() []types.Instance {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	result := []types.Instance{}
	for _, instance := range rep.instances {
		result = append(result, instance)
	}
	return result
}

func (rep *Representative) Vote(instance types.Instance) (float64, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	if !rep.hasRoomFor(instance) {
		return 0, InsufficientResources
	}
	return rep.score(instance), nil
}

func (rep *Representative) ReserveAndRecastVote(instance types.Instance) (float64, error) {
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

func (rep *Representative) Release(instance types.Instance) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	reservedInstance, ok := rep.instances[instance.InstanceGuid]
	if !(ok && reservedInstance.Tentative) {
		panic("wat?")
	}

	delete(rep.instances, instance.InstanceGuid)
}

func (rep *Representative) Claim(instance types.Instance) {
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

func (rep *Representative) hasRoomFor(instance types.Instance) bool {
	return rep.usedResources()+instance.RequiredResources <= rep.totalResources
}

func (rep *Representative) score(instance types.Instance) float64 {
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
