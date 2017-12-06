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
	"strconv"
	"strings"

	"github.com/golang/glog"
	godynect "github.com/nesv/go-dynect/dynect"
	"k8s.io/kubernetes/federation/pkg/dnsprovider"
	"k8s.io/kubernetes/federation/pkg/dnsprovider/rrstype"
)

// Compile time check for interface adherence
var _ dnsprovider.ResourceRecordSet = ResourceRecordSet{}

// ResourceRecordSet provides DynDNS implementation of the dns provider interface ResourceRecordSet
type ResourceRecordSet struct {
	impl   *godynect.Record
	rrsets *ResourceRecordSets
}

// Name returns the name of the ResourceRecordSet
func (rrset ResourceRecordSet) Name() string {
	return rrset.impl.Name
}

// Rrdatas returns the data of the ResourceRecordSet
func (rrset ResourceRecordSet) Rrdatas() []string {
	return strings.Split(rrset.impl.Value, ",")

}

// Ttl returns the time to live of the ResourceRecordSet
func (rrset ResourceRecordSet) Ttl() int64 {
	ttl, err := strconv.ParseInt(rrset.impl.TTL, 10, 64)
	if err != nil || ttl < 0 {
		glog.Errorf("Invalid ttl %v and setting it to default 60 seconds", rrset.impl.TTL)
		return 60
	}
	return ttl
}

// Type returns the type of the ResourceRecordSet
func (rrset ResourceRecordSet) Type() rrstype.RrsType {
	return rrstype.RrsType(rrset.impl.Type)
}
