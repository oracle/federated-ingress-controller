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
var _ GoDynectAPI = &GoDynectAPIStub{}

// GoDynectAPI captures the APIs will be invoked on the dynect client
type GoDynectAPI interface {
	GetRecord(record *godynect.Record) error
	Do(method, endpoint string, requestData, responseData interface{}) error
}

// GoDynectAPIStub is a stubbed implementation of the GoDynectAPI interface for testing
type GoDynectAPIStub struct {
	recordMap map[string]*godynect.Record
}

// NewGoDynectAPIStub returns an initialized NewGoDynectAPIStub
func NewGoDynectAPIStub() *GoDynectAPIStub {
	return &GoDynectAPIStub{make(map[string]*godynect.Record)}
}

// Do provides a stub implementation of sending a REST request
func (stub *GoDynectAPIStub) Do(method, endpoint string, requestData, responseData interface{}) error {
	// currently it only supports the request for getting all records
	if ps, ok := responseData.(**godynect.AllRecordsResponse); ok {
		var allRecordsData []string
		for _, record := range stub.recordMap {
			url := fmt.Sprintf("/REST/%sRecord/%s/%s/%s", record.Type, record.Zone, record.FQDN, hashKey(record))
			allRecordsData = append(allRecordsData, url)
		}
		(*ps).Data = allRecordsData
		return nil
	}
	return fmt.Errorf("OperationNotSupported")
}

// GetRecord provides a stub implementation of populating a record
func (stub *GoDynectAPIStub) GetRecord(record *godynect.Record) error {
	if record.ID == "" {
		return fmt.Errorf("Cannot call GetRecord with an empty ID")
	}
	rec := stub.recordMap[record.ID]
	record.Name = rec.Name
	record.Value = rec.Value
	record.TTL = rec.TTL
	return nil
}

// AddRecord provides a stub implementation of adding a record
func (stub *GoDynectAPIStub) AddRecord(record *godynect.Record) {
	key := stub.HashKey(record)
	stub.recordMap[key] = record
}

// HashKey returns a unique identifier for the record
func (stub *GoDynectAPIStub) HashKey(record *godynect.Record) string {
	return fmt.Sprintf("%s-%s-%s-%s", record.Name, record.TTL, record.Type, record.Value)
}
