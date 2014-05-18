package simulationrep

import (
	"errors"
	"fmt"
	"sync"

	"github.com/onsi/auction/auctionrep"
	"github.com/onsi/auction/types"
)

var InsufficientResources = errors.New("insufficient resources for instance")

type SimulationRep struct {
	guid           string
	lock           *sync.Mutex
	instances      map[string]types.Instance
	totalResources int
}

func New(guid string, totalResources int) auctionrep.TestAuctionRep {
	return &SimulationRep{
		guid:           guid,
		totalResources: totalResources,

		lock:      &sync.Mutex{},
		instances: map[string]types.Instance{},
	}
}

func (rep *SimulationRep) Guid() string {
	return rep.guid
}

func (rep *SimulationRep) Score(instance types.Instance) (float64, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	if !rep.hasRoomFor(instance) {
		return 0, InsufficientResources
	}
	return rep.score(instance), nil
}

func (rep *SimulationRep) ScoreThenTentativelyReserve(instance types.Instance) (float64, error) {
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

func (rep *SimulationRep) ReleaseReservation(instance types.Instance) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	reservedInstance, ok := rep.instances[instance.InstanceGuid]
	if !(ok && reservedInstance.Tentative) {
		return errors.New(fmt.Sprintf("no reservation for instance %s", reservedInstance.InstanceGuid))
	}

	delete(rep.instances, instance.InstanceGuid)

	return nil
}

func (rep *SimulationRep) Claim(instance types.Instance) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	_, ok := rep.instances[instance.InstanceGuid]
	if !ok {
		return errors.New(fmt.Sprintf("no reservation for instance %s", instance.InstanceGuid))
	}

	instance.Tentative = false
	rep.instances[instance.InstanceGuid] = instance

	return nil
}

// internals -- no locks here the operations above should be atomic

func (rep *SimulationRep) hasRoomFor(instance types.Instance) bool {
	return rep.usedResources()+instance.RequiredResources <= rep.totalResources
}

func (rep *SimulationRep) score(instance types.Instance) float64 {
	fResources := float64(rep.usedResources()) / float64(rep.totalResources)
	nInstances := rep.numberOfInstancesForAppGuid(instance.AppGuid)

	return fResources + float64(nInstances)
}

func (rep *SimulationRep) usedResources() int {
	usedResources := 0
	for _, instance := range rep.instances {
		usedResources += instance.RequiredResources
	}

	return usedResources
}

func (rep *SimulationRep) numberOfInstancesForAppGuid(guid string) int {
	n := 0
	for _, instance := range rep.instances {
		if instance.AppGuid == guid {
			n += 1
		}
	}
	return n
}

// simulation only

func (rep *SimulationRep) TotalResources() int {
	return rep.totalResources
}

func (rep *SimulationRep) Reset() {
	rep.lock.Lock()
	defer rep.lock.Unlock()
	rep.instances = map[string]types.Instance{}
}

func (rep *SimulationRep) SetInstances(instances []types.Instance) {
	rep.lock.Lock()
	defer rep.lock.Unlock()
	instancesMap := map[string]types.Instance{}
	for _, instance := range instances {
		instancesMap[instance.InstanceGuid] = instance
	}

	rep.instances = instancesMap
}

func (rep *SimulationRep) Instances() []types.Instance {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	result := []types.Instance{}
	for _, instance := range rep.instances {
		result = append(result, instance)
	}
	return result
}
