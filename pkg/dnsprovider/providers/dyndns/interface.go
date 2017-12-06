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
	"github.com/kubernetes-incubator/federated-ingress-controller/pkg/dnsprovider/providers/dyndns/stubs"
	"k8s.io/kubernetes/federation/pkg/dnsprovider"
)

// Compile time check for interface adherence
var _ dnsprovider.Interface = Interface{}

// Interface provides DynDNS implementation of the dns provider interface Interface
type Interface struct {
	dynDNSAPI stubs.DynDNSAPI
	zones     Zones
}

// newInterfaceWithStub facilitates stubbing out the underlying etcd
// library for testing purposes.  It returns an provider-independent interface.
func newInterfaceWithStub(dynDNSAPI stubs.DynDNSAPI) *Interface {
	return &Interface{dynDNSAPI: dynDNSAPI}
}

// Zones returns the hosted zones of the dns provider
func (i Interface) Zones() (dnsprovider.Zones, bool) {
	return i.zones, true
}
