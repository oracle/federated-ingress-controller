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

// Package dyndns is the implementation of pkg/dnsprovider interface for DynDNS
package dyndns

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/kubernetes-incubator/federated-ingress-controller/pkg/dnsprovider/providers/dyndns/dynect"
	"github.com/kubernetes-incubator/federated-ingress-controller/pkg/dnsprovider/providers/dyndns/stubs"
	gcfg "gopkg.in/gcfg.v1"
	"k8s.io/kubernetes/federation/pkg/dnsprovider"
)

// "dyndns" should be used to use this DNS provider
const (
	ProviderName = "dyndns"
)

// Config to override defaults
type Config struct {
	Global struct {
		DNSZones string `gcfg:"zones"`
		Customer string `gcfg:"customer"`
		User     string `gcfg:"user"`
		Password string `gcfg:"password"`
	}
}

// ConfigInfo holds information from config file
type ConfigInfo struct {
	DNSZones string
	Customer string
	User     string
	Password string
}

func init() {
	dnsprovider.RegisterDnsProvider(ProviderName, func(config io.Reader) (dnsprovider.Interface, error) {
		return newDynDNSProviderInterface(config)
	})
}

func parseConfig(config io.Reader) (*ConfigInfo, error) {
	configInfo := ConfigInfo{}

	// Possibly override defaults with config below
	if config != nil {
		var cfg Config
		if err := gcfg.ReadInto(&cfg, config); err != nil {
			glog.Errorf("Couldn't read config: %v", err)
			return nil, err
		}
		configInfo.DNSZones = cfg.Global.DNSZones
		configInfo.Customer = cfg.Global.Customer
		configInfo.User = cfg.Global.User
		configInfo.Password = cfg.Global.Password
	}

	if configInfo.DNSZones == "" {
		return nil, fmt.Errorf("Need to provide at least one DNS Zone in the config file")
	}

	if configInfo.Customer == "" {
		return nil, fmt.Errorf("Need to provide the customer in the config file")
	}

	if configInfo.User == "" {
		return nil, fmt.Errorf("Need to provide the user in the config file")
	}

	if configInfo.Password == "" {
		return nil, fmt.Errorf("Need to provide the password in the config file")
	}

	return &configInfo, nil
}

// newInterface creates a new instance of a DynDNS DNS Interface.
func newInterface(configInfo *ConfigInfo, dynDNSAPI stubs.DynDNSAPI) (*Interface, error) {
	glog.Infof("Using DynDNS DNS provider")

	glog.Infof("Signing into dyn with: %s and %s", configInfo.Customer, configInfo.User)

	err := dynDNSAPI.Login(configInfo.User, configInfo.Password)

	if err != nil {
		return nil, err
	}

	intf := newInterfaceWithStub(dynDNSAPI)
	zoneList := strings.Split(configInfo.DNSZones, ",")

	intf.zones = Zones{intf: intf}
	for index, zoneName := range zoneList {
		zone := Zone{domain: zoneName, id: strconv.Itoa(index), zones: &intf.zones}
		intf.zones.zoneList = append(intf.zones.zoneList, zone)
	}

	return intf, nil
}

// newDynDnsProviderInterface creates a new instance of a DynDNS DNS Interface.
func newDynDNSProviderInterface(config io.Reader) (*Interface, error) {
	configInfo, err := parseConfig(config)
	if err != nil {
		return nil, err
	}
	dynDNSAPI := dynect.NewDynectClient(configInfo.Customer)
	// dynDNSAPI.Verbose(true)

	return newInterface(configInfo, dynDNSAPI)

}

// newFakeInterface creates an interface for unit testing
func newFakeInterface(config io.Reader) (dnsprovider.Interface, error) {
	configInfo, err := parseConfig(config)
	if err != nil {
		return nil, err
	}
	dynDNSAPI := stubs.NewDynDNSAPIStub()
	return newInterface(configInfo, dynDNSAPI)

}
