package unit

import (
	"crypto/sha1"
	"fmt"
	"strings"

	"github.com/coreos/fleet/machine"
)

// RecognizedUnitType determines whether or not the given unit name represents
// a recognized unit type.
func RecognizedUnitType(name string) bool {
	types := []string{"service", "socket", "timer", "path", "device", "mount", "automount"}
	for _, t := range types {
		suffix := fmt.Sprintf(".%s", t)
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

// DefaultUnitType appends the default unit type to a given unit name, ignoring
// any file extensions that already exist.
func DefaultUnitType(name string) string {
	return fmt.Sprintf("%s.service", name)
}

// SHA1 sum
type Hash [sha1.Size]byte

func (h Hash) String() string {
	return fmt.Sprintf("%x", h[:])
}

func (h *Hash) Empty() bool {
	return *h == Hash{}
}

// A Unit represents a systemd configuration which encodes information about any of the unit
// types that fleet supports (as defined in SupportedUnitTypes()).
// Units are linked to Jobs by the Hash of their contents.
// Similar to systemd, a Unit configuration has no inherent name, but is rather
// named through the reference to it; in the case of systemd, the reference is
// the filename, and in the case of fleet, the reference is the name of the job
// that references this Unit.
type Unit struct {
	// Contents represents the parsed unit file.
	// This field must be considered readonly.
	Contents map[string]map[string][]string

	// Raw represents the entire contents of the unit file.
	Raw string
}

func (u *Unit) String() string {
	return u.Raw
}

// Hash returns the SHA1 hash of the raw contents of the Unit
func (u *Unit) Hash() Hash {
	return Hash(sha1.Sum([]byte(u.Raw)))
}

// UnitState encodes the current state of a unit loaded into systemd
type UnitState struct {
	LoadState    string                `json:"loadState"`
	ActiveState  string                `json:"activeState"`
	SubState     string                `json:"subState"`
	MachineState *machine.MachineState `json:"machineState"`
}

func NewUnitState(loadState, activeState, subState string, ms *machine.MachineState) *UnitState {
	return &UnitState{loadState, activeState, subState, ms}
}
