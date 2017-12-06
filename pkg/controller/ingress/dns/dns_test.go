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

package dns

import (
	"fmt"
	"net"
	"reflect"
	"sort"
	"testing"

	ic "github.com/kubernetes-incubator/federated-ingress-controller/pkg/controller/ingress"
	"github.com/kubernetes-incubator/federated-ingress-controller/pkg/controller/util"

	"k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/kubernetes/federation/apis/federation/v1beta1"

	fakefedclientset "k8s.io/kubernetes/federation/client/clientset_generated/federation_clientset/fake"
	"k8s.io/kubernetes/federation/pkg/dnsprovider"                           // Only for unit testing purposes.
	"k8s.io/kubernetes/federation/pkg/dnsprovider/providers/google/clouddns" // Only for unit testing purposes.

	apiv1 "k8s.io/api/core/v1"
	. "k8s.io/kubernetes/federation/pkg/federation-controller/util/test"
	"time"
)

const (
	ingresses              string = "ingresses"
	firstClusterAnnotation        = "ingress.federation.kubernetes.io/first-cluster"
)

// NewClusterWithRegionZone builds a new cluster object with given region and zone attributes.
func NewClusterWithRegionZone(name string, readyStatus v1.ConditionStatus, region, zone string) *v1beta1.Cluster {
	cluster := NewCluster(name, readyStatus)
	cluster.Status.Zones = []string{zone}
	cluster.Status.Region = region
	return cluster
}

func TestServiceController_ensureDnsRecords(t *testing.T) {
	_, _ = dnsprovider.InitDnsProvider("coredns", "")

	cluster1Name := "c1"
	cluster2Name := "c2"
	cluster1 := NewClusterWithRegionZone(cluster1Name, v1.ConditionTrue, "fooregion", "foozone")
	cluster2 := NewClusterWithRegionZone(cluster2Name, v1.ConditionTrue, "barregion", "barzone")
	globalDNSName := "ingname.ingns.myfederation.ing.federation.example.com"
	fooRegionDNSName := "ingname.ingns.myfederation.ing.fooregion.federation.example.com"
	fooZoneDNSName := "ingname.ingns.myfederation.ing.foozone.fooregion.federation.example.com"
	barRegionDNSName := "ingname.ingns.myfederation.ing.barregion.federation.example.com"
	barZoneDNSName := "ingname.ingns.myfederation.ing.barzone.barregion.federation.example.com"

	tests := []struct {
		name     string
		ingress  extensionsv1beta1.Ingress
		expected []string
	}{
		{
			name: "ServiceWithSingleLBIngress",
			ingress: extensionsv1beta1.Ingress{

				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
					globalIngressLBStatus: buildAnnotation(map[string][]string{
						cluster1Name: {"198.51.100.1", "198.51.100.2"},
						cluster2Name: {},
					})},
				},
			},
			expected: []string{
				"example.com:" + globalDNSName + ":A:180:[198.51.100.1 198.51.100.2]",
				"example.com:" + fooRegionDNSName + ":A:180:[198.51.100.1 198.51.100.2]",
				"example.com:" + fooZoneDNSName + ":A:180:[198.51.100.1 198.51.100.2]",
				"example.com:" + barRegionDNSName + ":CNAME:180:[" + globalDNSName + "]",
				"example.com:" + barZoneDNSName + ":CNAME:180:[" + barRegionDNSName + "]",
			},
		},
	}

	for _, test := range tests {
		fakedns, _ := clouddns.NewFakeInterface()
		fakednsZones, ok := fakedns.Zones()
		if !ok {
			t.Error("Unable to fetch zones")
		}

		fakeClient := &fakefedclientset.Clientset{}
		RegisterFakeClusterGet(&fakeClient.Fake, &v1beta1.ClusterList{Items: []v1beta1.Cluster{*cluster1, *cluster2}})
		a := &util.AbstractDNSController{}
		d := IngressDNSController{
			AbstractDNSController: a,
			federationClient:      fakeClient,
			dns:                   fakedns,
			dnsZones:              fakednsZones,
			domain:                "federation.example.com",
			ingressDNSSuffix:      "ing",
			federationName:        "myfederation",
		}

		dnsZone, err := d.GetDNSZone(d.domain, d.dnsZones)
		if err != nil || dnsZone == nil {
			t.Errorf("Test failed for %s, Get DNS Zones failed: %v", test.name, err)
		}
		d.dnsZone = dnsZone
		test.ingress.Name = "ingname"
		test.ingress.Namespace = "ingns"

		d.ensureDNSRecords(cluster1Name, &test.ingress)
		d.ensureDNSRecords(cluster2Name, &test.ingress)

		verifyDNSRecords(fakednsZones, test.expected, test.name, t)
	}
}

func verifyDNSRecords(fakednsZones dnsprovider.Zones, expected []string, testname string, t *testing.T) {
	zones, err := fakednsZones.List()
	if err != nil {
		t.Errorf("error querying zones: %v", err)
	}

	// Dump every record to a testable-by-string-comparison form
	records := []string{}
	for _, z := range zones {
		zoneName := z.Name()

		rrs, ok := z.ResourceRecordSets()
		if !ok {
			t.Errorf("cannot get rrs for zone %q", zoneName)
		}

		rrList, err := rrs.List()
		if err != nil {
			t.Errorf("error querying rr for zone %q: %v", zoneName, err)
		}
		for _, rr := range rrList {
			rrdatas := rr.Rrdatas()

			// Put in consistent (testable-by-string-comparison) order
			sort.Strings(rrdatas)
			records = append(records, fmt.Sprintf("%s:%s:%s:%d:%s", zoneName, rr.Name(), rr.Type(), rr.Ttl(), rrdatas))
		}
	}

	// Ignore order of records
	sort.Strings(records)
	sort.Strings(expected)

	if !reflect.DeepEqual(records, expected) {
		t.Errorf("Test %q failed.  Actual=%v, Expected=%v", testname, records, expected)
	} else {
		t.Logf("Test %q passed.  Actual=%v, Expected=%v", testname, records, expected)
	}
}

func TestParseGlobalLbStatusAnnotation(t *testing.T) {
	cluster1Name := "c1"
	cluster2Name := "c2"
	cluster1 := NewClusterWithRegionZone(cluster1Name, v1.ConditionTrue, "fooregion", "foozone")
	cluster2 := NewClusterWithRegionZone(cluster2Name, v1.ConditionTrue, "barregion", "barzone")

	fakedns, _ := clouddns.NewFakeInterface()
	fakednsZones, ok := fakedns.Zones()
	if !ok {
		t.Error("Unable to fetch zones")
	}

	fakeClient := &fakefedclientset.Clientset{}
	RegisterFakeClusterGet(&fakeClient.Fake, &v1beta1.ClusterList{Items: []v1beta1.Cluster{*cluster1, *cluster2}})
	a := &util.AbstractDNSController{}
	idc := IngressDNSController{
		AbstractDNSController: a,
		federationClient:      fakeClient,
		dns:                   fakedns,
		dnsZones:              fakednsZones,
		ingressDNSSuffix:      "ing",
		domain:                "federation.example.com",
		federationName:        "myfederation",
	}
	fedIngress := extensionsv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "mynamespace",
			SelfLink:  "/api/v1/namespaces/mynamespace/ingress/test-ingress",
		},
	}
	map1, err := idc.parseGlobalLbStatusAnnotation(&fedIngress)
	if err != nil {
		t.Errorf("TestParseGlobalLbStatusAnnotation failed: %v", err)
	}
	if map1 != nil {
		t.Errorf("Did not get the expected nil map")
	}

	fedIngress2 := extensionsv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "mynamespace",
			SelfLink:  "/api/v1/namespaces/mynamespace/ingress/test-ingress",
			Annotations: map[string]string{
				firstClusterAnnotation: cluster1.Name,
			},
		},
	}
	map2, err := idc.parseGlobalLbStatusAnnotation(&fedIngress2)
	if err != nil {
		t.Errorf("TestParseGlobalLbStatusAnnotation failed: %v", err)
	}
	if map2 != nil {
		t.Errorf("Did not get the expected nil map")
	}

	fedIngress3 := extensionsv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "mynamespace",
			SelfLink:  "/api/v1/namespaces/mynamespace/ingress/test-ingress",
			Annotations: map[string]string{
				firstClusterAnnotation: cluster1.Name,
				globalIngressLBStatus: buildAnnotation(map[string][]string{
					cluster1Name: {"198.51.100.1"},
					cluster2Name: {"test.example.com"},
				}),
			},
		},
	}
	map3, err := idc.parseGlobalLbStatusAnnotation(&fedIngress3)
	if err != nil {
		t.Errorf("TestParseGlobalLbStatusAnnotation failed: %v", err)
	}
	for lbClusterName, lbStatuses := range *map3 {
		if err != nil {
			t.Errorf("TestParseGlobalLbStatusAnnotation failed: %v", err)
		}
		for _, lbStatus := range lbStatuses {
			for _, lbIngress := range lbStatus.Ingress {
				if lbClusterName == "c1" {
					if lbIngress.IP != "198.51.100.1" {
						t.Errorf("Did not  get expected IP for c1, got %s instead", lbIngress.IP)
					}
				}
				if lbClusterName == "c2" {
					if lbIngress.Hostname != "test.example.com" {
						t.Errorf("Did not  get expected host name for c2, got %s instead", lbIngress.Hostname)
					}
				}
			}
		}
	}
}

func TestGetHealthEndpoints(t *testing.T) {
	cluster1Name := "c1"
	cluster2Name := "c2"
	cluster1 := NewClusterWithRegionZone(cluster1Name, v1.ConditionTrue, "fooregion", "foozone")
	cluster2 := NewClusterWithRegionZone(cluster2Name, v1.ConditionTrue, "barregion", "")

	fakedns, _ := clouddns.NewFakeInterface()
	fakednsZones, ok := fakedns.Zones()
	if !ok {
		t.Error("Unable to fetch zones")
	}

	fakeClient := &fakefedclientset.Clientset{}
	RegisterFakeClusterGet(&fakeClient.Fake, &v1beta1.ClusterList{Items: []v1beta1.Cluster{*cluster1, *cluster2}})
	a := &util.AbstractDNSController{}
	idc := IngressDNSController{
		AbstractDNSController: a,
		federationClient:      fakeClient,
		dns:                   fakedns,
		dnsZones:              fakednsZones,
		ingressDNSSuffix:      "ing",
		domain:                "federation.example.com",
		federationName:        "myfederation",
	}
	fedIngress := extensionsv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "mynamespace",
			SelfLink:  "/api/v1/namespaces/mynamespace/ingress/test-ingress",
			Annotations: map[string]string{
				firstClusterAnnotation: cluster1.Name,
				globalIngressLBStatus: buildAnnotation(map[string][]string{
					cluster1Name: {"198.51.100.1"},
					cluster2Name: {"test.example.com"},
				}),
			},
		},
		Status: extensionsv1beta1.IngressStatus{
			LoadBalancer: apiv1.LoadBalancerStatus{
				Ingress: []apiv1.LoadBalancerIngress{
					{IP: "198.51.100.1"}},
			},
		}}

	zoneEndpoints, regionEndpoints, globalEndpoints, err := idc.getHealthyEndpoints(cluster1Name, &fedIngress)
	if err != nil {
		t.Errorf("TestGetHealthEndpoints failed %v", err)
	}
	if zoneEndpoints == nil || len(zoneEndpoints) != 1 || zoneEndpoints[0] != "198.51.100.1" {
		t.Errorf("Got unexpected zoneEndpoints for c1 %v", zoneEndpoints)
	}
	if regionEndpoints == nil || len(regionEndpoints) != 1 || regionEndpoints[0] != "198.51.100.1" {
		t.Errorf("Got unexpected regionEndpoints for c1 %v", regionEndpoints)
	}
	if globalEndpoints == nil || len(globalEndpoints) != 2 {
		t.Errorf("Got unexpected globalEndpoints for c1 %v", globalEndpoints)
	}

	zoneEndpoints, regionEndpoints, globalEndpoints, err = idc.getHealthyEndpoints(cluster2Name, &fedIngress)
	if err != nil {
		t.Errorf("TestGetHealthEndpoints failed %v", err)
	}
	if zoneEndpoints == nil || len(zoneEndpoints) != 1 || zoneEndpoints[0] != "test.example.com" {
		t.Errorf("Got unexpected zoneEndpoints for c1 %v", zoneEndpoints)
	}
	if regionEndpoints == nil || len(regionEndpoints) != 1 || regionEndpoints[0] != "test.example.com" {
		t.Errorf("Got unexpected regionEndpoints for c1 %v", regionEndpoints)
	}
	if globalEndpoints == nil || len(globalEndpoints) != 2 {
		t.Errorf("Got unexpected globalEndpoints for c1 %v", globalEndpoints)
	}

	fedIngress.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	zoneEndpoints, regionEndpoints, globalEndpoints, err = idc.getHealthyEndpoints(cluster1Name, &fedIngress)
	if err != nil {
		t.Errorf("TestGetHealthEndpoints failed %v", err)
	}
	if zoneEndpoints != nil || regionEndpoints != nil || globalEndpoints != nil {
		t.Errorf("Did not expected nil endpoints for deleted ingress")
	}
}

func TestValidateConfig(t *testing.T) {
	_, _ = dnsprovider.InitDnsProvider("coredns", "")

	cluster1Name := "c1"
	cluster2Name := "c2"
	cluster1 := NewClusterWithRegionZone(cluster1Name, v1.ConditionTrue, "fooregion", "foozone")
	cluster2 := NewClusterWithRegionZone(cluster2Name, v1.ConditionTrue, "barregion", "barzone")

	fakedns, _ := clouddns.NewFakeInterface()
	fakednsZones, ok := fakedns.Zones()
	if !ok {
		t.Error("Unable to fetch zones")
	}

	fakeClient := &fakefedclientset.Clientset{}
	RegisterFakeClusterGet(&fakeClient.Fake, &v1beta1.ClusterList{Items: []v1beta1.Cluster{*cluster1, *cluster2}})
	a := &util.AbstractDNSController{}
	idc1 := IngressDNSController{
		AbstractDNSController: a,
		federationClient:      fakeClient,
		dns:                   fakedns,
		dnsZones:              fakednsZones,
		ingressDNSSuffix:      "ing",
		domain:                "federation.example.com",
		federationName:        "myfederation",
	}
	err := idc1.validateConfig()
	if err != nil {
		t.Errorf("TestValidateConfig failed :%v", err)
	}

	idc2 := IngressDNSController{
		AbstractDNSController: a,
		federationClient:      fakeClient,
		dns:                   fakedns,
		dnsZones:              fakednsZones,
		ingressDNSSuffix:      "ing",
		federationName:        "myfederation",
	}
	err = idc2.validateConfig()
	if err != nil {
		t.Logf("Got expected validation error on nil domain :%v", err)
	}

	idc3 := IngressDNSController{
		AbstractDNSController: a,
		federationClient:      fakeClient,
		dnsZones:              fakednsZones,
		domain:                "federation.example.com",
		ingressDNSSuffix:      "ing",
		federationName:        "myfederation",
	}
	err = idc3.validateConfig()
	if err != nil {
		t.Logf("Got expected validation error on nil dns :%v", err)
	}

	idc4 := IngressDNSController{
		AbstractDNSController: a,
		federationClient:      fakeClient,
		dns:                   fakedns,
		dnsZones:              fakednsZones,
		domain:                "federation.example.com",
		ingressDNSSuffix:      "ing",
	}
	err = idc4.validateConfig()
	if err != nil {
		t.Logf("Got expected validation error on nil federation name :%v", err)
	}
}

func TestRetrieveDNSZone(t *testing.T) {
	_, _ = dnsprovider.InitDnsProvider("coredns", "")

	cluster1Name := "c1"
	cluster2Name := "c2"
	cluster1 := NewClusterWithRegionZone(cluster1Name, v1.ConditionTrue, "fooregion", "foozone")
	cluster2 := NewClusterWithRegionZone(cluster2Name, v1.ConditionTrue, "barregion", "barzone")

	fakedns, _ := clouddns.NewFakeInterface()
	fakednsZones, ok := fakedns.Zones()
	if !ok {
		t.Error("Unable to fetch zones")
	}

	fakeClient := &fakefedclientset.Clientset{}
	RegisterFakeClusterGet(&fakeClient.Fake, &v1beta1.ClusterList{Items: []v1beta1.Cluster{*cluster1, *cluster2}})
	a := &util.AbstractDNSController{}
	idc := IngressDNSController{
		AbstractDNSController: a,
		federationClient:      fakeClient,
		dns:                   fakedns,
		dnsZones:              fakednsZones,
		domain:                "federation.example.com",
		ingressDNSSuffix:      "ing",
		federationName:        "myfederation",
	}
	err := idc.retrieveOrCreateDNSZone()
	if err != nil {
		t.Errorf("TestRetrieveDNSZone failed :%v", err)
	}
	if idc.dnsZone.Name() != "example.com" {
		t.Errorf("Got unexpected zone :%s", idc.dnsZone.Name())
	}
}

func TestCreateDNSZone(t *testing.T) {
	_, _ = dnsprovider.InitDnsProvider("coredns", "")

	cluster1Name := "c1"
	cluster2Name := "c2"
	cluster1 := NewClusterWithRegionZone(cluster1Name, v1.ConditionTrue, "fooregion", "foozone")
	cluster2 := NewClusterWithRegionZone(cluster2Name, v1.ConditionTrue, "barregion", "barzone")

	fakedns, _ := clouddns.NewFakeInterface()
	fakednsZones, ok := fakedns.Zones()
	if !ok {
		t.Error("Unable to fetch zones")
	}
	zones, err := fakednsZones.List()
	for _, dnsZone := range zones {
		fakednsZones.Remove(dnsZone)
	}

	fakeClient := &fakefedclientset.Clientset{}
	RegisterFakeClusterGet(&fakeClient.Fake, &v1beta1.ClusterList{Items: []v1beta1.Cluster{*cluster1, *cluster2}})
	a := &util.AbstractDNSController{}
	idc := IngressDNSController{
		AbstractDNSController: a,
		federationClient:      fakeClient,
		dns:                   fakedns,
		dnsZones:              fakednsZones,
		domain:                "test.example.com",
		ingressDNSSuffix:      "ing",
		federationName:        "myfederation",
	}
	err = idc.retrieveOrCreateDNSZone()
	if err != nil {
		t.Errorf("TestRetrieveOrCreateDNSZone failed :%v", err)
	}
	if idc.dnsZone.Name() != "test.example.com" {
		t.Errorf("Got unexpected zone :%s", idc.dnsZone.Name())
	}

	idc2 := IngressDNSController{
		AbstractDNSController: a,
		federationClient:      fakeClient,
		dns:                   fakedns,
		dnsZones:              fakednsZones,
		ingressDNSSuffix:      "ing",
		federationName:        "myfederation",
	}

	err = idc2.retrieveOrCreateDNSZone()
	if err != nil {
		t.Logf("Got expected error for creating zone with empty domain :%v", err)
	} else {
		t.Errorf("Did not get expected error for creating zone with empty domain")
	}
}

func TestNewIngressDNSController(t *testing.T) {
	_, _ = dnsprovider.InitDnsProvider("coredns", "")

	cluster1Name := "c1"
	cluster2Name := "c2"
	cluster1 := NewClusterWithRegionZone(cluster1Name, v1.ConditionTrue, "fooregion", "foozone")
	cluster2 := NewClusterWithRegionZone(cluster2Name, v1.ConditionTrue, "barregion", "barzone")

	fakedns, _ := clouddns.NewFakeInterface()
	fakeClient := &fakefedclientset.Clientset{}
	RegisterFakeClusterGet(&fakeClient.Fake, &v1beta1.ClusterList{Items: []v1beta1.Cluster{*cluster1, *cluster2}})
	idc, err := newIngressDNSController(fakeClient, fakedns, "",
		"myfederation", "ing", "federation.example.com")
	if err != nil {
		t.Errorf("TestNewIngressDNSController failed :%v", err)
	}
	err = idc.retrieveOrCreateDNSZone()
	if err != nil {
		t.Errorf("TestNewIngressDNSController failed :%v", err)
	}
	idc1, err := newIngressDNSController(fakeClient, fakedns, "",
		"myfederation", "", "federation.example.com")
	if err != nil {
		t.Errorf("TestNewIngressDNSController failed :%v", err)
	}
	if idc1.ingressDNSSuffix != defaultIngressSeperator {
		t.Errorf("Wrong ingressDNSSuffix value %s, failed to set with the default ingress separator :%v", idc1.ingressDNSSuffix, err)
	}
	_, err = newIngressDNSController(fakeClient, fakedns, "",
		"", "ing", "federation.example.com")
	if err != nil {
		t.Logf("Got expected validation error with empty federation name:%v", err)
	} else {
		t.Errorf("Did not get expected validation error with empty federation name:%v", err)
	}
}

func TestDNSControllerRun(t *testing.T) {
	_, _ = dnsprovider.InitDnsProvider("coredns", "")

	cluster1Name := "c1"
	cluster2Name := "c2"
	cluster1 := NewClusterWithRegionZone(cluster1Name, v1.ConditionTrue, "fooregion", "foozone")
	cluster2 := NewClusterWithRegionZone(cluster2Name, v1.ConditionTrue, "barregion", "barzone")

	fakedns, _ := clouddns.NewFakeInterface()
	fakednsZones, ok := fakedns.Zones()
	if !ok {
		t.Error("Unable to fetch zones")
	}

	fedClient := &fakefedclientset.Clientset{}
	RegisterFakeClusterGet(&fedClient.Fake, &v1beta1.ClusterList{Items: []v1beta1.Cluster{*cluster1, *cluster2}})
	RegisterFakeList(ingresses, &fedClient.Fake, &extensionsv1beta1.IngressList{Items: []extensionsv1beta1.Ingress{}})
	fedIngressWatch := RegisterFakeWatch(ingresses, &fedClient.Fake)
	RegisterFakeCopyOnUpdate(ingresses, &fedClient.Fake, fedIngressWatch)

	idc, err := newIngressDNSController(fedClient, fakedns, "",
		"myfederation", "ing", "federation.example.com")
	if err != nil {
		t.Errorf("TestDNSControllerRun failed :%v", err)
	}

	dnsZone, err := idc.GetDNSZone(idc.domain, idc.dnsZones)
	if err != nil || dnsZone == nil {
		t.Errorf("Test failed, Get DNS Zones failed: %v", err)
	}
	idc.dnsZone = dnsZone
	idc.workQueue = NewTestWorkQueue()

	// ingress without Status.LoadBalancer.Ingress field will not trigger
	// DNS records creation
	fedIngress := extensionsv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ingname2",
			Namespace: "ingns",
			Annotations: map[string]string{
				globalIngressLBStatus: buildAnnotation(map[string][]string{
					cluster1Name: {"123.78.10.12"},
					cluster2Name: {"test.net"},
				})},
		}}
	idc.workQueue.Add(&fedIngress)

	// ingress Status.LoadBalancer.Ingress field will trigger
	// DNS records creation
	fedIngress2 := extensionsv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ingname",
			Namespace: "ingns",
			Annotations: map[string]string{
				firstClusterAnnotation: cluster1.Name,
				globalIngressLBStatus: buildAnnotation(map[string][]string{
					cluster1Name: {"198.51.100.1", "198.51.100.2"},
					cluster2Name: {},
				}),
			},
		},
		Status: extensionsv1beta1.IngressStatus{
			LoadBalancer: apiv1.LoadBalancerStatus{
				Ingress: []apiv1.LoadBalancerIngress{
					{IP: "198.51.100.1", Hostname: "198.51.100.2"}},
			},
		}}
	idc.workQueue.Add(&fedIngress2)

	stop := make(chan struct{})
	go idc.dnsControllerRun(1, stop)
	time.Sleep(time.Second * 2)
	close(stop)

	// verify that ingress (fedIngress) without Status.LoadBalancer.Ingress field
	// will not trigger DNS records creation, and ingress (fedIngress2) with
	// Status.LoadBalancer.Ingress field triggers DNS records creation as expected.
	globalDNSName := "ingname.ingns.myfederation.ing.federation.example.com"
	fooRegionDNSName := "ingname.ingns.myfederation.ing.fooregion.federation.example.com"
	fooZoneDNSName := "ingname.ingns.myfederation.ing.foozone.fooregion.federation.example.com"
	barRegionDNSName := "ingname.ingns.myfederation.ing.barregion.federation.example.com"
	barZoneDNSName := "ingname.ingns.myfederation.ing.barzone.barregion.federation.example.com"

	expected := []string{
		"example.com:" + globalDNSName + ":A:180:[198.51.100.1 198.51.100.2]",
		"example.com:" + fooRegionDNSName + ":A:180:[198.51.100.1 198.51.100.2]",
		"example.com:" + fooZoneDNSName + ":A:180:[198.51.100.1 198.51.100.2]",
		"example.com:" + barRegionDNSName + ":CNAME:180:[" + globalDNSName + "]",
		"example.com:" + barZoneDNSName + ":CNAME:180:[" + barRegionDNSName + "]"}

	verifyDNSRecords(fakednsZones, expected, "TestDNSControllerRun", t)
}

func NewTestWorkQueue() *TestWorkQueue {
	return NewNamed("")
}

func NewNamed(name string) *TestWorkQueue {
	return &TestWorkQueue{}
}

type TestWorkQueue struct {
	queue        []t
	shuttingDown bool
}

type t interface{}

func (q *TestWorkQueue) Add(item interface{}) {
	q.queue = append(q.queue, item)
}

func (q *TestWorkQueue) Len() int {
	return len(q.queue)
}

func (q *TestWorkQueue) Get() (item interface{}, shutdown bool) {
	if len(q.queue) == 0 {
		// We must be shutting down.
		return nil, true
	}
	item, q.queue = q.queue[0], q.queue[1:]
	return item, false
}

func (q *TestWorkQueue) Done(item interface{}) {
}

func (q *TestWorkQueue) ShutDown() {
	q.shuttingDown = true
}

func (q *TestWorkQueue) ShuttingDown() bool {
	return q.shuttingDown
}

func buildAnnotation(clusters map[string][]string) string {
	globalLbStat := make(map[string][]v1.LoadBalancerStatus)
	for clusterName, hosts := range clusters {
		lbStatus := v1.LoadBalancerStatus{Ingress: make([]v1.LoadBalancerIngress, len(hosts))}
		for i := 0; i < len(hosts); i++ {
			addr := net.ParseIP(hosts[i])
			if addr == nil {
				lbStatus.Ingress[i] = v1.LoadBalancerIngress{Hostname: hosts[i]}
			} else {
				lbStatus.Ingress[i] = v1.LoadBalancerIngress{IP: hosts[i]}
			}
		}
		globalLbStat[clusterName] = append(globalLbStat[clusterName], lbStatus)
	}

	return ic.BuildGlobalLbStatusAnnotation(globalLbStat)
}
