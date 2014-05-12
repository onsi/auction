package fakestoreadapter

import (
	"github.com/cloudfoundry/storeadapter"
	"regexp"
	"strings"
	"sync"
)

type containerNode struct {
	dir   bool
	nodes map[string]*containerNode

	storeNode storeadapter.StoreNode
}

type FakeStoreAdapterErrorInjector struct {
	KeyRegexp *regexp.Regexp
	Error     error
}

func NewFakeStoreAdapterErrorInjector(keyRegexp string, err error) *FakeStoreAdapterErrorInjector {
	return &FakeStoreAdapterErrorInjector{
		KeyRegexp: regexp.MustCompile(keyRegexp),
		Error:     err,
	}
}

type FakeStoreAdapter struct {
	DidConnect    bool
	DidDisconnect bool

	ConnectErr        error
	DisconnectErr     error
	SetErrInjector    *FakeStoreAdapterErrorInjector
	GetErrInjector    *FakeStoreAdapterErrorInjector
	ListErrInjector   *FakeStoreAdapterErrorInjector
	DeleteErrInjector *FakeStoreAdapterErrorInjector
	CreateErrInjector *FakeStoreAdapterErrorInjector

	WatchErrChannel chan error

	rootNode *containerNode

	MaintainedNodeName string
	MaintainNodeError  error
	ReleaseNodeChannel chan chan bool

	createLock *sync.Mutex

	eventChannel chan storeadapter.WatchEvent
	sendEvents   bool
}

func New() *FakeStoreAdapter {
	adapter := &FakeStoreAdapter{}
	adapter.Reset()
	return adapter
}

func (adapter *FakeStoreAdapter) Reset() {
	adapter.DidConnect = false
	adapter.DidDisconnect = false

	adapter.ConnectErr = nil
	adapter.DisconnectErr = nil
	adapter.SetErrInjector = nil
	adapter.GetErrInjector = nil
	adapter.ListErrInjector = nil
	adapter.DeleteErrInjector = nil
	adapter.CreateErrInjector = nil

	adapter.rootNode = &containerNode{
		dir:   true,
		nodes: make(map[string]*containerNode),
	}

	adapter.createLock = new(sync.Mutex)
	adapter.sendEvents = false
	adapter.eventChannel = make(chan storeadapter.WatchEvent)
}

func (adapter *FakeStoreAdapter) Connect() error {
	adapter.DidConnect = true
	return adapter.ConnectErr
}

func (adapter *FakeStoreAdapter) Disconnect() error {
	adapter.DidDisconnect = true
	return adapter.DisconnectErr
}

func (adapter *FakeStoreAdapter) sendEvent(node storeadapter.StoreNode, eventType storeadapter.EventType) {
	if adapter.sendEvents {
		go func() {
			adapter.eventChannel <- storeadapter.WatchEvent{
				Type: eventType,
				Node: node,
			}
		}()
	}
}

func (adapter *FakeStoreAdapter) SetMulti(nodes []storeadapter.StoreNode) error {
	var eventType storeadapter.EventType

	for _, node := range nodes {

		_, err := adapter.Get(node.Key)
		if err == nil {
			eventType = storeadapter.UpdateEvent
		}

		if adapter.SetErrInjector != nil && adapter.SetErrInjector.KeyRegexp.MatchString(node.Key) {
			return adapter.SetErrInjector.Error
		}
		components := adapter.keyComponents(node.Key)

		container := adapter.rootNode
		for i, component := range components {
			if i == len(components)-1 {
				existingNode, exists := container.nodes[component]
				if exists && existingNode.dir {
					return storeadapter.ErrorNodeIsDirectory
				}
				container.nodes[component] = &containerNode{storeNode: node}
			} else {
				existingNode, exists := container.nodes[component]
				if exists {
					if !existingNode.dir {
						return storeadapter.ErrorNodeIsNotDirectory
					}
					container = existingNode
				} else {
					newContainer := &containerNode{dir: true, nodes: make(map[string]*containerNode)}
					container.nodes[component] = newContainer
					container = newContainer
				}
			}
		}

		adapter.sendEvent(node, eventType)
	}

	return nil
}

func (adapter *FakeStoreAdapter) Create(node storeadapter.StoreNode) error {
	adapter.createLock.Lock()
	defer adapter.createLock.Unlock()

	if adapter.CreateErrInjector != nil && adapter.CreateErrInjector.KeyRegexp.MatchString(node.Key) {
		return adapter.CreateErrInjector.Error
	}

	_, err := adapter.Get(node.Key)
	if err == nil {
		return storeadapter.ErrorKeyExists
	}

	return adapter.SetMulti([]storeadapter.StoreNode{node})
}

func (adapter *FakeStoreAdapter) Get(key string) (storeadapter.StoreNode, error) {
	if adapter.GetErrInjector != nil && adapter.GetErrInjector.KeyRegexp.MatchString(key) {
		return storeadapter.StoreNode{}, adapter.GetErrInjector.Error
	}

	components := adapter.keyComponents(key)
	container := adapter.rootNode
	for _, component := range components {
		var exists bool
		container, exists = container.nodes[component]
		if !exists {
			return storeadapter.StoreNode{}, storeadapter.ErrorKeyNotFound
		}
	}

	if container.dir {
		return storeadapter.StoreNode{}, storeadapter.ErrorNodeIsDirectory
	} else {
		return container.storeNode, nil
	}
}

func (adapter *FakeStoreAdapter) ListRecursively(key string) (storeadapter.StoreNode, error) {
	if adapter.ListErrInjector != nil && adapter.ListErrInjector.KeyRegexp.MatchString(key) {
		return storeadapter.StoreNode{}, adapter.ListErrInjector.Error
	}

	container := adapter.rootNode

	components := adapter.keyComponents(key)
	for _, component := range components {
		var exists bool
		container, exists = container.nodes[component]
		if !exists {
			return storeadapter.StoreNode{}, storeadapter.ErrorKeyNotFound
		}
	}

	if !container.dir {
		return storeadapter.StoreNode{}, storeadapter.ErrorNodeIsNotDirectory
	}

	return adapter.listContainerNode(key, container), nil
}

func (adapter *FakeStoreAdapter) listContainerNode(key string, container *containerNode) storeadapter.StoreNode {
	childNodes := []storeadapter.StoreNode{}

	for nodeKey, node := range container.nodes {
		if node.dir {
			if key == "/" {
				nodeKey = "/" + nodeKey
			} else {
				nodeKey = key + "/" + nodeKey
			}
			childNodes = append(childNodes, adapter.listContainerNode(nodeKey, node))
		} else {
			childNodes = append(childNodes, node.storeNode)
		}
	}

	return storeadapter.StoreNode{
		Key:        key,
		Dir:        true,
		ChildNodes: childNodes,
	}
}

func (adapter *FakeStoreAdapter) Delete(keys ...string) error {
	for _, key := range keys {
		node, _ := adapter.Get(key)

		if adapter.DeleteErrInjector != nil && adapter.DeleteErrInjector.KeyRegexp.MatchString(key) {
			return adapter.DeleteErrInjector.Error
		}

		components := adapter.keyComponents(key)
		container := adapter.rootNode
		parentNode := adapter.rootNode
		for _, component := range components {
			var exists bool
			parentNode = container
			container, exists = container.nodes[component]
			if !exists {
				return storeadapter.ErrorKeyNotFound
			}
		}

		delete(parentNode.nodes, components[len(components)-1])
		adapter.sendEvent(node, storeadapter.DeleteEvent)
	}

	return nil
}

func (adapter *FakeStoreAdapter) UpdateDirTTL(key string, ttl uint64) error {
	panic("not implemented")
}

func (adapter *FakeStoreAdapter) Update(node storeadapter.StoreNode) error {
	panic("not implemented")
}

func (adapter *FakeStoreAdapter) CompareAndSwap(oldNode storeadapter.StoreNode, newNode storeadapter.StoreNode) error {
	panic("not implemented")
}

func (adapter *FakeStoreAdapter) Watch(key string) (events <-chan storeadapter.WatchEvent, stop chan<- bool, errors <-chan error) {
	adapter.sendEvents = true
	adapter.WatchErrChannel = make(chan error, 1)

	// We haven't implemented stop yet

	return adapter.eventChannel, nil, adapter.WatchErrChannel
}

func (adapter *FakeStoreAdapter) keyComponents(key string) (components []string) {
	for _, s := range strings.Split(key, "/") {
		if s != "" {
			components = append(components, s)
		}
	}

	return components
}

func (adapter *FakeStoreAdapter) MaintainNode(storeNode storeadapter.StoreNode) (lostNodeChannel <-chan bool, releaseNode chan chan bool, err error) {
	adapter.MaintainedNodeName = storeNode.Key
	adapter.ReleaseNodeChannel = make(chan chan bool, 1)

	return nil, adapter.ReleaseNodeChannel, adapter.MaintainNodeError
}
