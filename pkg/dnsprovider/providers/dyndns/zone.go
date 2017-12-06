/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dyndns

import (
	"k8s.io/kubernetes/federation/pkg/dnsprovider"
)

// Compile time check for interface adherence
var _ dnsprovider.Zone = &Zone{}

// Zone provides DynDNS implementation of the dns provider interface Zone
type Zone struct {
	domain string
	id     string
	zones  *Zones
}

// Name returns the name of the zone
func (zone Zone) Name() string {
	return zone.domain
}

// ID returns the id of the zone
func (zone Zone) ID() string {
	return zone.id
}

// ResourceRecordSets returns the ResourceRecordSets of the zone
func (zone Zone) ResourceRecordSets() (dnsprovider.ResourceRecordSets, bool) {
	return &ResourceRecordSets{zone: &zone}, true
}
