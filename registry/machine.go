package registry

import (
	"path"
	"strings"
	"time"

	"github.com/coreos/fleet/third_party/github.com/coreos/go-etcd/etcd"

	"github.com/coreos/fleet/event"
	"github.com/coreos/fleet/machine"
)

const (
	machinePrefix = "machines"
)

// Describe all active Machines
func (r *EtcdRegistry) GetActiveMachines() (machines []machine.MachineState, err error) {
	key := path.Join(r.keyPrefix, machinePrefix)
	resp, err := r.etcd.Get(key, false, true)

	if err != nil {
		if isKeyNotFound(err) {
			err = nil
		}
		return
	}

	for _, kv := range resp.Node.Nodes {
		_, machID := path.Split(kv.Key)
		mach, _ := r.GetMachineState(machID)
		if mach != nil {
			machines = append(machines, *mach)
		}
	}

	return
}

// Get Machine object from etcd
func (r *EtcdRegistry) GetMachineState(machID string) (*machine.MachineState, error) {
	key := path.Join(r.keyPrefix, machinePrefix, machID, "object")
	resp, err := r.etcd.Get(key, false, true)

	if err != nil {
		if isKeyNotFound(err) {
			err = nil
		}
		return nil, err
	}

	var mach machine.MachineState
	if err := unmarshal(resp.Node.Value, &mach); err != nil {
		return nil, err
	}

	return &mach, nil
}

// Push Machine object to etcd
func (r *EtcdRegistry) SetMachineState(ms machine.MachineState, ttl time.Duration) (uint64, error) {
	json, err := marshal(ms)
	if err != nil {
		return uint64(0), err
	}
	key := path.Join(r.keyPrefix, machinePrefix, ms.ID, "object")

	// Assume state is already present, returning on success
	resp, err := r.etcd.Update(key, json, uint64(ttl.Seconds()))
	if err == nil {
		return resp.Node.ModifiedIndex, nil
	}

	// If state was not present, explicitly create it so the other members
	// in the cluster know this is a new member
	resp, err = r.etcd.Create(key, json, uint64(ttl.Seconds()))
	if err != nil {
		return uint64(0), err
	}

	return resp.Node.ModifiedIndex, nil
}

// Remove Machine object from etcd
func (r *EtcdRegistry) RemoveMachineState(machID string) error {
	key := path.Join(r.keyPrefix, machinePrefix, machID, "object")
	_, err := r.etcd.Delete(key, false)
	if isKeyNotFound(err) {
		err = nil
	}
	return err
}

// Attempt to acquire a lock on a given machine for a given amount of time
func (r *EtcdRegistry) LockMachine(machID, context string) *TimedResourceMutex {
	return r.lockResource("machine", machID, context)
}

func filterEventMachineCreated(resp *etcd.Response) *event.Event {
	dir, baseName := path.Split(resp.Node.Key)
	if baseName != "object" {
		return nil
	}

	dir = strings.TrimSuffix(dir, "/")
	dir = path.Dir(dir)
	prefixName := path.Base(dir)

	if prefixName != machinePrefix {
		return nil
	}

	if resp.Action != "create" {
		return nil
	}

	var m machine.MachineState
	unmarshal(resp.Node.Value, &m)
	return &event.Event{"EventMachineCreated", m, nil}
}

func filterEventMachineRemoved(resp *etcd.Response) *event.Event {
	dir, baseName := path.Split(resp.Node.Key)
	if baseName != "object" {
		return nil
	}

	dir = strings.TrimSuffix(dir, "/")
	dir = path.Dir(dir)
	prefixName := path.Base(dir)

	if prefixName != machinePrefix {
		return nil
	}

	if resp.Action != "expire" && resp.Action != "delete" {
		return nil
	}

	machID := path.Base(path.Dir(resp.Node.Key))
	return &event.Event{"EventMachineRemoved", machID, nil}
}
