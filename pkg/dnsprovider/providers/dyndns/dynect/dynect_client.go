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
	"fmt"
	glog "github.com/golang/glog"
	"github.com/kubernetes-incubator/federated-ingress-controller/pkg/dnsprovider/providers/dyndns/stubs"
	godynect "github.com/nesv/go-dynect/dynect"
	"strings"
)

// DynectClient REST client of DynDNS Dynect API, with a few extra helper methods
// on top of the godynect SDK ConvenientClient
type DynectClient struct {
	*godynect.ConvenientClient
	goDynectAPI stubs.GoDynectAPI
}

// NewDynectClient Creates a new DynectClient
func NewDynectClient(customerName string) *DynectClient {
	convenientClient := godynect.NewConvenientClient(customerName)
	return &DynectClient{ConvenientClient: convenientClient, goDynectAPI: convenientClient}
}

// GetAllRecords return all records for an entry
func (d *DynectClient) GetAllRecords(record *godynect.Record, records *godynect.AllRecordsResponse) error {
	url := fmt.Sprintf("AllRecord/%s/%s", record.Zone, record.FQDN)
	err := d.goDynectAPI.Do("GET", url, nil, &records)
	return err
}

// RecordFromURL return a Record from a URL
func (d *DynectClient) RecordFromURL(url string) *godynect.Record {
	record := new(godynect.Record)
	parts := strings.Split(url, "/")
	if len(parts) != 6 {
		glog.Errorf("Failed to parse url: %v %#v", url, parts)
		return nil
	}
	record.Zone = parts[3]
	record.FQDN = parts[4]
	record.ID = parts[5]
	record.Type = strings.TrimSuffix(parts[2], "Record")
	return record
}

// GetRecordID finds the dns record ID by fetching all records for a FQDN
func (d *DynectClient) GetRecordID(record *godynect.Record) error {
	finalID := ""
	var records godynect.AllRecordsResponse
	err := d.GetAllRecords(record, &records)
	if err != nil {
		return fmt.Errorf("Failed to find Dyn record id: %s", err)
	}
	for _, recordURL := range records.Data {
		rec := d.RecordFromURL(recordURL)
		err = d.goDynectAPI.GetRecord(rec)
		if err != nil {
			return fmt.Errorf("Failed to get Dyn record %s: %v", rec.Name, err)
		}
		id := strings.TrimPrefix(recordURL, fmt.Sprintf("/REST/%sRecord/%s/%s/", record.Type, record.Zone, record.FQDN))
		if !strings.Contains(id, "/") && id != "" && rec.Value == record.Value {
			finalID = id
			glog.Infof("[INFO] Found Dyn record ID: %s", id)
			break
		}
	}
	if finalID == "" {
		return fmt.Errorf("Failed to find Dyn record id")
	}

	record.ID = finalID
	return nil
}

// GetRecordByName gets records with the specified record name
func (d *DynectClient) GetRecordByName(record *godynect.Record) ([]*godynect.Record, error) {
	var records godynect.AllRecordsResponse
	err := d.GetAllRecords(record, &records)
	if err != nil {
		return nil, fmt.Errorf("Failed to find Dyn records: %s", err)
	}
	var matchingRecords []*godynect.Record
	for _, recordURL := range records.Data {
		rec := d.RecordFromURL(recordURL)
		err = d.goDynectAPI.GetRecord(rec)
		if err != nil {
			return nil, fmt.Errorf("Failed to get Dyn record %s: %v", rec.Name, err)
		}
		if rec.Name == record.Name {
			glog.Infof("[INFO] Found matching url %s", recordURL)
			matchingRecords = append(matchingRecords, rec)
		}
	}
	return matchingRecords, nil
}

// GetAllRecordList gets all record details
func (d *DynectClient) GetAllRecordList(record *godynect.Record) ([]*godynect.Record, error) {
	var records godynect.AllRecordsResponse
	err := d.GetAllRecords(record, &records)
	if err != nil {
		return nil, fmt.Errorf("Failed to find Dyn records: %v", err)
	}
	var recordList []*godynect.Record
	for _, recordURL := range records.Data {
		glog.Infof("[INFO] Found url %s", recordURL)
		rec := d.RecordFromURL(recordURL)
		err = d.goDynectAPI.GetRecord(rec)
		if err != nil {
			return nil, fmt.Errorf("Failed to get Dyn record %s: %v", rec.Name, err)
		}
		recordList = append(recordList, rec)
	}
	return recordList, nil
}
