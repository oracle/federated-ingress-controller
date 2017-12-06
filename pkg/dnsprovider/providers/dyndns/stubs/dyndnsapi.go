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

// Package stubs implements a stub for the service, used primarily for unit testing purposes
package stubs

import (
	"fmt"

	godynect "github.com/nesv/go-dynect/dynect"
)

// Compile time check for interface conformance
var _ DynDNSAPI = &DynDNSAPIStub{}

// DynDNSAPI captures the APIs will be invoked on the dyndns service
type DynDNSAPI interface {
	Login(user string, password string) error
	CreateRecord(record *godynect.Record) error
	DeleteRecord(record *godynect.Record) error
	GetRecordByName(record *godynect.Record) ([]*godynect.Record, error)
	GetRecordID(record *godynect.Record) error
	GetAllRecordList(record *godynect.Record) ([]*godynect.Record, error)
	PublishZone(zone string) error
}

// DynDNSAPIStub is a stubbed implementation of the DynDNSAPI interface for testing
type DynDNSAPIStub struct {
	rrsets map[string]*godynect.Record
}

// NewDynDNSAPIStub returns an initialized DynDNSAPIStub
func NewDynDNSAPIStub() *DynDNSAPIStub {
	return &DynDNSAPIStub{make(map[string]*godynect.Record)}
}

// Login provides a stub implementation of authenticating
func (stub *DynDNSAPIStub) Login(user string, password string) error {
	if user != "foo" || password != "bar" {
		return fmt.Errorf("Failed to login with user name %s and password %s", user, password)
	}
	return nil
}

// CreateRecord provides a stub implementation of creating a record
func (stub *DynDNSAPIStub) CreateRecord(record *godynect.Record) error {
	key := hashKey(record)
	if _, found := stub.rrsets[key]; found {
		return fmt.Errorf("Attempt to create duplicate record %s", record.Name)
	}

	stub.rrsets[key] = record
	return nil
}

// DeleteRecord provides a stub implementation of deleting a record
func (stub *DynDNSAPIStub) DeleteRecord(record *godynect.Record) error {
	key := hashKey(record)
	if _, found := stub.rrsets[key]; !found {
		return fmt.Errorf("Attempt to delete non-existent record %s", record.Name)
	}
	delete(stub.rrsets, key)
	return nil
}

// GetRecordByName provides a stub implementation of getting records matching the specified name
func (stub *DynDNSAPIStub) GetRecordByName(record *godynect.Record) ([]*godynect.Record, error) {
	var recList []*godynect.Record
	for key := range stub.rrsets {
		rec := stub.rrsets[key]
		if rec.Name == record.Name {
			recList = append(recList, rec)
		}
	}
	return recList, nil
}

// GetRecordID provides a stub implementation of getting id for the record
func (stub *DynDNSAPIStub) GetRecordID(record *godynect.Record) error {
	return nil
}

// GetAllRecordList provides a stub implementation of getting all records
func (stub *DynDNSAPIStub) GetAllRecordList(record *godynect.Record) ([]*godynect.Record, error) {
	var records []*godynect.Record
	for k := range stub.rrsets {
		records = append(records, stub.rrsets[k])
	}
	return records, nil
}

// PublishZone provides a stub implementation of publishing the changes
func (stub *DynDNSAPIStub) PublishZone(zone string) error {
	// no op for the stub
	return nil
}

func hashKey(record *godynect.Record) string {
	return fmt.Sprintf("%s-%s-%s-%s", record.Name, record.TTL, record.Type, record.Value)
}
