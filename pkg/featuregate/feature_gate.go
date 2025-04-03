// Copyright 2025 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package featuregate

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// Feature represents a feature gate name
type Feature string

// FeatureStage represents the maturity level of a feature
type FeatureStage string

const (
	// Alpha means the feature is experimental and disabled by default
	Alpha FeatureStage = "ALPHA"
	// Beta means the feature is more stable but still might change and is disabled by default
	Beta FeatureStage = "BETA"
	// GA means the feature is generally available and enabled by default
	GA FeatureStage = ""
)

// FeatureSpec describes a feature and its properties
type FeatureSpec struct {
	// Default is the default enablement state for the feature
	Default bool
	// LockToDefault indicates the feature cannot be changed from its default
	LockToDefault bool
	// Stage indicates the maturity level of the feature
	Stage FeatureStage
}

// Define all available features here
var (
	VirtualNet = Feature("VirtualNet")
)

// defaultFeatures defines default features with their specifications
var defaultFeatures = map[Feature]FeatureSpec{
	// Actual features
	VirtualNet: {Default: false, Stage: Alpha},
}

// FeatureGate indicates whether a given feature is enabled or not
type FeatureGate interface {
	// Enabled returns true if the key is enabled
	Enabled(key Feature) bool
	// KnownFeatures returns a slice of strings describing the known features
	KnownFeatures() []string
}

// MutableFeatureGate allows for dynamic feature gate configuration
type MutableFeatureGate interface {
	FeatureGate

	// SetFromMap sets feature gate values from a map[string]bool
	SetFromMap(m map[string]bool) error
	// Add adds features to the feature gate
	Add(features map[Feature]FeatureSpec) error
	// String returns a string representing the feature gate configuration
	String() string
}

// featureGate implements the FeatureGate and MutableFeatureGate interfaces
type featureGate struct {
	// lock guards writes to known, enabled, and reads/writes of closed
	lock sync.Mutex
	// known holds a map[Feature]FeatureSpec
	known atomic.Value
	// enabled holds a map[Feature]bool
	enabled atomic.Value
	// closed is set to true once the feature gates are considered immutable
	closed bool
}

// NewFeatureGate creates a new feature gate with the default features
func NewFeatureGate() MutableFeatureGate {
	known := map[Feature]FeatureSpec{}
	for k, v := range defaultFeatures {
		known[k] = v
	}

	f := &featureGate{}
	f.known.Store(known)
	f.enabled.Store(map[Feature]bool{})
	return f
}

// SetFromMap sets feature gate values from a map[string]bool
func (f *featureGate) SetFromMap(m map[string]bool) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	// Copy existing state
	known := map[Feature]FeatureSpec{}
	for k, v := range f.known.Load().(map[Feature]FeatureSpec) {
		known[k] = v
	}
	enabled := map[Feature]bool{}
	for k, v := range f.enabled.Load().(map[Feature]bool) {
		enabled[k] = v
	}

	// Apply the new settings
	for k, v := range m {
		k := Feature(k)
		featureSpec, ok := known[k]
		if !ok {
			return fmt.Errorf("unrecognized feature gate: %s", k)
		}
		if featureSpec.LockToDefault && featureSpec.Default != v {
			return fmt.Errorf("cannot set feature gate %v to %v, feature is locked to %v", k, v, featureSpec.Default)
		}
		enabled[k] = v
	}

	// Persist the changes
	f.known.Store(known)
	f.enabled.Store(enabled)
	return nil
}

// Add adds features to the feature gate
func (f *featureGate) Add(features map[Feature]FeatureSpec) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.closed {
		return fmt.Errorf("cannot add feature gates after the feature gate is closed")
	}

	// Copy existing state
	known := map[Feature]FeatureSpec{}
	for k, v := range f.known.Load().(map[Feature]FeatureSpec) {
		known[k] = v
	}

	// Add new features
	for name, spec := range features {
		if existingSpec, found := known[name]; found {
			if existingSpec == spec {
				continue
			}
			return fmt.Errorf("feature gate %q with different spec already exists: %v", name, existingSpec)
		}
		known[name] = spec
	}

	// Persist changes
	f.known.Store(known)

	return nil
}

// String returns a string containing all enabled feature gates, formatted as "key1=value1,key2=value2,..."
func (f *featureGate) String() string {
	pairs := []string{}
	for k, v := range f.enabled.Load().(map[Feature]bool) {
		pairs = append(pairs, fmt.Sprintf("%s=%t", k, v))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ",")
}

// Enabled returns true if the key is enabled
func (f *featureGate) Enabled(key Feature) bool {
	if v, ok := f.enabled.Load().(map[Feature]bool)[key]; ok {
		return v
	}
	if v, ok := f.known.Load().(map[Feature]FeatureSpec)[key]; ok {
		return v.Default
	}
	return false
}

// KnownFeatures returns a slice of strings describing the FeatureGate's known features
// GA features are hidden from the list
func (f *featureGate) KnownFeatures() []string {
	knownFeatures := f.known.Load().(map[Feature]FeatureSpec)
	known := make([]string, 0, len(knownFeatures))
	for k, v := range knownFeatures {
		if v.Stage == GA {
			continue
		}
		known = append(known, fmt.Sprintf("%s=true|false (%s - default=%t)", k, v.Stage, v.Default))
	}
	sort.Strings(known)
	return known
}

// Default feature gates instance
var DefaultFeatureGates = NewFeatureGate()

// Enabled checks if a feature is enabled in the default feature gates
func Enabled(name Feature) bool {
	return DefaultFeatureGates.Enabled(name)
}

// SetFromMap sets feature gate values from a map in the default feature gates
func SetFromMap(featureMap map[string]bool) error {
	return DefaultFeatureGates.SetFromMap(featureMap)
}
