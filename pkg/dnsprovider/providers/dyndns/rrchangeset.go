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
	"fmt"
	"k8s.io/kubernetes/federation/pkg/dnsprovider"
)

// Compile time check for interface adherence
var _ dnsprovider.ResourceRecordChangeset = &ResourceRecordChangeset{}

// ChangeSetType the type of the ResourceRecordSet change
type ChangeSetType string

const (
	// ADDITION record change type for adding ResourceRecordSet
	ADDITION = ChangeSetType("ADDITION")
	// DELETION record change type for deleting ResourceRecordSet
	DELETION = ChangeSetType("DELETION")
	// UPSERT record change type for upserting ResourceRecordSet
	UPSERT = ChangeSetType("UPSERT")
)

// ChangeSet to hold ResourceRecordSet changes
type ChangeSet struct {
	cstype ChangeSetType
	rrset  dnsprovider.ResourceRecordSet
}

// ResourceRecordChangeset provides DynDNS implementation of the dns provider interface ResourceRecordChangeset
type ResourceRecordChangeset struct {
	zone      *Zone
	rrsets    *ResourceRecordSets
	changeset []ChangeSet
}

// Add adds a ResourceRecordSet to ChangeSet
func (c *ResourceRecordChangeset) Add(rrset dnsprovider.ResourceRecordSet) dnsprovider.ResourceRecordChangeset {
	c.changeset = append(c.changeset, ChangeSet{cstype: ADDITION, rrset: rrset})
	return c
}

// Remove removes a ResourceRecordSet from ChangeSet
func (c *ResourceRecordChangeset) Remove(rrset dnsprovider.ResourceRecordSet) dnsprovider.ResourceRecordChangeset {
	c.changeset = append(c.changeset, ChangeSet{cstype: DELETION, rrset: rrset})
	return c
}

// Upsert upserts a ResourceRecordSet to ChangeSet
func (c *ResourceRecordChangeset) Upsert(rrset dnsprovider.ResourceRecordSet) dnsprovider.ResourceRecordChangeset {
	c.changeset = append(c.changeset, ChangeSet{cstype: UPSERT, rrset: rrset})
	return c
}

// IsEmpty checks whether the ChangeSet is empty
func (c *ResourceRecordChangeset) IsEmpty() bool {
	return len(c.changeset) == 0
}

// Apply publishes the changes of the ChangeSet
func (c *ResourceRecordChangeset) Apply() error {
	if c.IsEmpty() {
		return nil
	}

	for _, changeset := range c.changeset {
		switch changeset.cstype {
		case ADDITION:
			addRecords := c.rrsets.GetRecordsFromDNSRRS(changeset.rrset)
			for _, addRecord := range addRecords {
				err := c.rrsets.zone.zones.intf.dynDNSAPI.CreateRecord(addRecord)
				if err != nil {
					return fmt.Errorf("Couldn't add record %s with value %s: %v", addRecord.Name, addRecord.Value, err)
				}
			}
		case UPSERT:
			// not needed for the federation use case yet
		case DELETION:
			deleteRecords := c.rrsets.GetRecordsFromDNSRRS(changeset.rrset)
			for _, deleteRecord := range deleteRecords {
				err := c.rrsets.zone.zones.intf.dynDNSAPI.GetRecordID(deleteRecord)
				if err != nil {
					return fmt.Errorf("Couldn't get record id %s with value %s: %v", deleteRecord.Name, deleteRecord.Value, err)
				}
				err = c.rrsets.zone.zones.intf.dynDNSAPI.DeleteRecord(deleteRecord)
				if err != nil {
					return fmt.Errorf("Couldn't delete record %s with value %s: %v", deleteRecord.Name, deleteRecord.Value, err)
				}
			}
		}
	}
	err := c.rrsets.zone.zones.intf.dynDNSAPI.PublishZone(c.zone.Name())
	if err != nil {
		return fmt.Errorf("Couldn't publish the changes for the zone %s: %v", c.zone.Name(), err)
	}
	return nil
}

// ResourceRecordSets returns the parent ResourceRecordSets
func (c *ResourceRecordChangeset) ResourceRecordSets() dnsprovider.ResourceRecordSets {
	return c.rrsets
}
