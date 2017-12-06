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

package dynect

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/kubernetes-incubator/federated-ingress-controller/pkg/dnsprovider/providers/dyndns/stubs"
	godynect "github.com/nesv/go-dynect/dynect"
	"k8s.io/kubernetes/federation/pkg/dnsprovider/rrstype"
	"strconv"
)

func newTestInterface() (*DynectClient, error) {
	return NewDynectTestClient(), nil
}

var dynectClient *DynectClient

var goDynectAPIStub *stubs.GoDynectAPIStub

func TestMain(m *testing.M) {
	fmt.Printf("Parsing flags.\n")
	flag.Parse()
	var err error
	fmt.Printf("Getting new test interface.\n")
	dynectClient, err = newTestInterface()
	if err != nil {
		fmt.Printf("Error creating interface: %v", err)
		os.Exit(1)
	}
	populateRecords()
	fmt.Printf("Running tests...\n")
	os.Exit(m.Run())
}

func NewDynectTestClient() *DynectClient {
	goDynectAPIStub = stubs.NewGoDynectAPIStub()
	return &DynectClient{goDynectAPI: goDynectAPIStub}
}

func createNewRecord(name string, rdata string) *godynect.Record {
	record := new(godynect.Record)
	record.Zone = "example.com"
	record.Name = name
	record.Type = string(rrstype.A)
	record.FQDN = record.Name + "." + record.Zone
	record.TTL = strconv.FormatInt(180, 10)
	record.Value = rdata
	return record
}

func populateRecords() {
	record := createNewRecord("test", "10.26.10.6")
	goDynectAPIStub.AddRecord(record)
	record = createNewRecord("test", "10.26.10.25")
	goDynectAPIStub.AddRecord(record)
	record = createNewRecord("test2", "20.10.7.23")
	goDynectAPIStub.AddRecord(record)
}

func TestGetAllRecords(t *testing.T) {
	var allRecords godynect.AllRecordsResponse
	record := createNewRecord("test", "10.26.10.6")
	err := dynectClient.GetAllRecords(record, &allRecords)
	if err != nil {
		t.Errorf("Failed to get all records %v", err)
	}
	if len(allRecords.Data) != 3 {
		t.Errorf("Expected to get records size as 3, but got %d instead", len(allRecords.Data))
	}
}

func TestRecordFromURL(t *testing.T) {
	record := createNewRecord("test", "10.26.10.6")
	url := fmt.Sprintf("/REST/%sRecord/%s/%s/%s", record.Type, record.Zone, record.FQDN, goDynectAPIStub.HashKey(record))
	rec := dynectClient.RecordFromURL(url)
	if rec.Type != record.Type {
		t.Errorf("Expected to get record type as %s, but got %s instead", record.Type, rec.Type)
	}
	if rec.Zone != record.Zone {
		t.Errorf("Expected to get record zone as %s, but got %s instead", record.Zone, rec.Zone)
	}
}

func TestGetRecordID(t *testing.T) {
	record := createNewRecord("test", "10.26.10.6")
	err := dynectClient.GetRecordID(record)
	if err != nil {
		t.Errorf("Failed to get record id for record %s: %v", record.FQDN, err)
	}
	if record.ID != goDynectAPIStub.HashKey(record) {
		t.Errorf("Expected to get record id as %s, but got %s instead", goDynectAPIStub.HashKey(record), record.ID)
	}
}

func TestGetRecordByName(t *testing.T) {
	record := createNewRecord("test", "10.26.10.6")
	records, err := dynectClient.GetRecordByName(record)
	if err != nil {
		t.Errorf("Failed to get record by name %s: %v", record.Name, err)
	}
	if len(records) != 2 {
		t.Errorf("Expected to get records size as 2, but got %d instead", len(records))
	}
	for _, rec := range records {
		if rec.Name != record.Name {
			t.Errorf("Got the wrong record with name %s, type %s, value %s",
				rec.Name, rec.Type, rec.Value)

		} else {
			t.Logf("Found record with name %s, type %s, value %s",
				rec.Name, rec.Type, rec.Value)
		}
	}
}

func TestGetAllRecordList(t *testing.T) {
	record := createNewRecord("test", "10.26.10.6")
	records, err := dynectClient.GetAllRecordList(record)
	if err != nil {
		t.Errorf("Failed to get all record list %v", err)
	}
	if len(records) != 3 {
		t.Errorf("Expected to get records size as 3, but got %d instead", len(records))
	}
}
