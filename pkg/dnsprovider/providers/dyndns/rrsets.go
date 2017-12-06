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
	"strconv"
	"strings"

	"github.com/golang/glog"

	godynect "github.com/nesv/go-dynect/dynect"
	"k8s.io/kubernetes/federation/pkg/dnsprovider"
	"k8s.io/kubernetes/federation/pkg/dnsprovider/rrstype"
)

// Compile time check for interface adherence
var _ dnsprovider.ResourceRecordSets = ResourceRecordSets{}

// ResourceRecordSets provides DynDNS implementation of the dns provider interface ResourceRecordSets
type ResourceRecordSets struct {
	zone *Zone
}

// List lists the ResourceRecordSet in its hosted zone
func (rrsets ResourceRecordSets) List() ([]dnsprovider.ResourceRecordSet, error) {
	var list []dnsprovider.ResourceRecordSet
	record := new(godynect.Record)
	record.Zone = rrsets.zone.Name()
	record.FQDN = record.Zone
	recordList, err := rrsets.zone.zones.intf.dynDNSAPI.GetAllRecordList(record)
	if err != nil {
		return list, fmt.Errorf("Failed to list records %v", err)
	}
	for _, rec := range recordList {
		rrs := &ResourceRecordSet{impl: rec, rrsets: &rrsets}
		list = append(list, rrs)
	}
	return list, nil
}

// Get gets the list of ResourceRecordSets that match the specified record name
func (rrsets ResourceRecordSets) Get(name string) ([]dnsprovider.ResourceRecordSet, error) {
	var list []dnsprovider.ResourceRecordSet

	record := new(godynect.Record)
	record.Zone = rrsets.zone.Name()
	rrsets.populateRecordWithName(name, record)

	records, err := rrsets.zone.zones.intf.dynDNSAPI.GetRecordByName(record)

	if err != nil {
		glog.Errorf("Failed to get record with name %s, %v", name, err)
		return list, nil
	}

	for _, rec := range records {
		r := ResourceRecordSet{rec, &rrsets}
		list = append(list, r)
	}

	return list, nil
}

// StartChangeset returns a new ResourceRecordChangeset
func (rrsets ResourceRecordSets) StartChangeset() dnsprovider.ResourceRecordChangeset {
	return &ResourceRecordChangeset{
		zone:   rrsets.zone,
		rrsets: &rrsets,
	}
}

// New returns a new ResourceRecordSet
func (rrsets ResourceRecordSets) New(name string, rrdatas []string, ttl int64, rrsType rrstype.RrsType) dnsprovider.ResourceRecordSet {
	record := rrsets.newRecord(name, rrdatas, ttl, rrsType)

	rrset := &ResourceRecordSet{
		impl:   record,
		rrsets: &rrsets,
	}

	return rrset
}

// GetRecordsFromDNSRRS gets records which map to single IP address so they can be used for REST API to DynDNS server
func (rrsets ResourceRecordSets) GetRecordsFromDNSRRS(rrs dnsprovider.ResourceRecordSet) []*godynect.Record {

	var impl *godynect.Record
	var records []*godynect.Record

	if dynrrs, ok := rrs.(*ResourceRecordSet); ok {
		impl = dynrrs.impl
	} else {
		impl = rrsets.newRecord(rrs.Name(), rrs.Rrdatas(), rrs.Ttl(), rrs.Type())
	}

	rdatas := strings.Split(impl.Value, ",")
	if len(rdatas) == 1 {
		return append(records, impl)
	}

	for _, rdata := range rdatas {
		record := rrsets.newRecord(rrs.Name(), []string{rdata}, rrs.Ttl(), rrs.Type())
		records = append(records, record)
	}

	return records
}

func (rrsets ResourceRecordSets) newRecord(name string, rrdatas []string, ttl int64, rrsType rrstype.RrsType) *godynect.Record {
	record := new(godynect.Record)
	record.Zone = rrsets.zone.Name()
	rrsets.populateRecordWithName(name, record)
	record.Type = string(rrsType)
	record.TTL = strconv.FormatInt(ttl, 10)
	record.Value = strings.Join(rrdatas, ",")
	return record
}

// Zone returns the parent zone
func (rrsets ResourceRecordSets) Zone() dnsprovider.Zone {
	return rrsets.zone
}

func (rrsets ResourceRecordSets) populateRecordWithName(name string, record *godynect.Record) {
	name = strings.TrimSuffix(name, ".")
	if isFQDN := strings.HasSuffix(name, "."+record.Zone); isFQDN {
		record.FQDN = name
		record.Name = strings.TrimSuffix(record.FQDN, "."+record.Zone)
	} else {
		record.Name = name
		record.FQDN = fmt.Sprintf("%s.%s", record.Name, record.Zone)
	}
}
