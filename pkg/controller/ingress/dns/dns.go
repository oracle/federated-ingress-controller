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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/kubernetes-incubator/federated-ingress-controller/pkg/controller/util"
	"k8s.io/kubernetes/federation/pkg/dnsprovider"

	"k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	federationclientset "k8s.io/kubernetes/federation/client/clientset_generated/federation_clientset"
	futil "k8s.io/kubernetes/federation/pkg/federation-controller/util"
	corelisters "k8s.io/kubernetes/pkg/client/listers/extensions/internalversion"

	opt "github.com/kubernetes-incubator/federated-ingress-controller/pkg/controller/ingress/options"
)

const (
	// ControllerName name of the controller
	ControllerName          = "ingress-dns"
	defaultIngressSeperator = "ing"
	globalIngressLBStatus   = "kubernetes.io/ingress.global-ingress-lb-status"
	// UserAgentName name of the user agent
	UserAgentName = "federation-ingress-dns-controller"

	ingressSyncPeriod = 30 * time.Second
)

// IngressDNSController DNS controller used for routing federated ingress endpoints
type IngressDNSController struct {
	*util.AbstractDNSController
	dns dnsprovider.Interface
	// domain to put dns records
	domain string
	// Client to federation api server
	federationClient federationclientset.Interface
	federationName   string
	// ingressDNSSuffix is the DNS suffix we use when publishing ingress DNS names
	ingressDNSSuffix string
	// each federation should be configured with a single zone (e.g. "mycompany.com")
	dnsZone  dnsprovider.Zone
	dnsZones dnsprovider.Zones
	// Informer Store for federated ingresses
	ingressStore corelisters.IngressLister
	// Informer controller for federated ingresses
	ingressInformerController cache.Controller
	workQueue                 workqueue.Interface
}

// StartFederatedIngressController starts a new federated ingress controller
func StartFederatedIngressController(config *restclient.Config, options *opt.FederatedIngressControllerOptions, stopChan <-chan struct{}) {
	restclient.AddUserAgent(config, "federatedingress-controller")
	client := federationclientset.NewForConfigOrDie(config)
	controller, err := NewIngressDNSController(client,
		options.DnsProvider,
		options.DnsProviderConfig,
		options.FkubeName,
		options.IngressDnsSuffix,
		options.Domain,
	)

	if err != nil {
		glog.Fatalf("error: %v", err)
	}
	controller.dnsControllerRun(1, stopChan)
}

// NewIngressDNSController returns a new ingress dns controller to manage DNS records for federated ingresses
func NewIngressDNSController(client federationclientset.Interface,
	dnsProvider string,
	dnsProviderConfig string,
	federationName string,
	ingressDNSSuffix string,
	domain string) (*IngressDNSController, error) {

	dns, err := dnsprovider.InitDnsProvider(dnsProvider, dnsProviderConfig)
	if err != nil {
		runtime.HandleError(fmt.Errorf("DNS provider could not be initialized: %v", err))
		return nil, err
	}
	return newIngressDNSController(client, dns, dnsProviderConfig, federationName, ingressDNSSuffix, domain)
}

func newIngressDNSController(client federationclientset.Interface,
	dns dnsprovider.Interface,
	dnsProviderConfig string,
	federationName string,
	ingressDNSSuffix string,
	domain string) (*IngressDNSController, error) {

	if ingressDNSSuffix == "" {
		ingressDNSSuffix = defaultIngressSeperator
	}

	a := &util.AbstractDNSController{}
	d := &IngressDNSController{
		AbstractDNSController: a,
		federationClient:      client,
		dns:                   dns,
		federationName:        federationName,
		ingressDNSSuffix:      ingressDNSSuffix,
		domain:                domain,
		workQueue:             workqueue.New(),
	}
	if err := d.validateConfig(); err != nil {
		runtime.HandleError(fmt.Errorf("Invalid configuration passed to DNS provider: %v", err))
		return nil, err
	}
	if err := d.retrieveOrCreateDNSZone(); err != nil {
		runtime.HandleError(fmt.Errorf("Failed to retrieve DNS zone: %v", err))
		return nil, err
	}

	// Start informer in federated API servers on federated ingresses
	var ingressIndexer cache.Indexer
	ingressIndexer, d.ingressInformerController = cache.NewIndexerInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error) {
				return client.ExtensionsV1beta1().Ingresses(metav1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return client.ExtensionsV1beta1().Ingresses(metav1.NamespaceAll).Watch(options)
			},
		},
		&extensionsv1beta1.Ingress{},
		ingressSyncPeriod,
		futil.NewTriggerOnAllChanges(func(obj pkgruntime.Object) { d.workQueue.Add(obj) }),
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	d.ingressStore = corelisters.NewIngressLister(ingressIndexer)
	return d, nil
}

func (idc *IngressDNSController) dnsControllerRun(workers int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	defer idc.workQueue.ShutDown()

	glog.Infof("Starting federation ingress dns controller")
	defer glog.Infof("Stopping federation ingress dns controller")

	go idc.ingressInformerController.Run(stopCh)

	for i := 0; i < workers; i++ {
		go wait.Until(idc.worker, time.Second, stopCh)
	}

	<-stopCh
}

func wantsDNSRecords(ingress *extensionsv1beta1.Ingress) bool {
	return len(ingress.Status.LoadBalancer.Ingress) > 0
}

func (idc *IngressDNSController) workerFunction() bool {
	item, quit := idc.workQueue.Get()
	if quit {
		return true
	}
	defer idc.workQueue.Done(item)

	ingress := item.(*extensionsv1beta1.Ingress)

	if !wantsDNSRecords(ingress) {
		glog.V(4).Infof("Got Ingess event %s in cluster %s but Status.LoadBalancer.Ingress is empty ", ingress, ingress.ClusterName)
		return false
	}
	glog.V(4).Infof("Got Ingess event %s in cluster %s", ingress, ingress.ClusterName)

	globalIngLbStatuses, err := idc.parseGlobalLbStatusAnnotation(ingress)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Error in parsing Global Ingress LB Status for  %s/%s: %v", ingress.Namespace, ingress.Name, err))
		return false
	}
	if globalIngLbStatuses != nil {
		for clusterName := range *globalIngLbStatuses {
			idc.ensureDNSRecords(clusterName, ingress)
		}
	}
	return false
}

func (idc *IngressDNSController) worker() {
	for {
		if quit := idc.workerFunction(); quit {
			glog.Infof("ingress dns controller worker queue shutting down")
			return
		}
	}
}

func (idc *IngressDNSController) validateConfig() error {
	if idc.federationName == "" {
		return fmt.Errorf("DNSController should not be run without federationName")
	}
	if idc.domain == "" {
		return fmt.Errorf("DNSController must be run with a domain")
	}
	if idc.dns == nil {
		return fmt.Errorf("DNSController should not be run without a dnsprovider")
	}
	zones, ok := idc.dns.Zones()
	if !ok {
		return fmt.Errorf("the dns provider does not support zone enumeration, which is required for creating dns records")
	}
	idc.dnsZones = zones
	return nil
}

func (idc *IngressDNSController) retrieveOrCreateDNSZone() error {
	zone, err := idc.GetDNSZone(idc.domain, idc.dnsZones)
	if err != nil {
		return fmt.Errorf("error querying for DNS zones: %v", err)
	}
	if zone == nil {
		if idc.domain == "" {
			return fmt.Errorf("DNSController must be run with domain to create zone automatically")
		}
		glog.Infof("DNS zone %q not found.  Creating DNS zone %q.", idc.domain, idc.domain)
		newZone, err := idc.dnsZones.New(idc.domain)
		if err != nil {
			return err
		}
		idc.dnsZone = newZone
		glog.Infof("DNS zone %q successfully created.  Note that DNS resolution will not work until you have registered this name with "+
			"a DNS registrar and they have changed the authoritative name servers for your domain to point to your DNS provider", newZone.Name())
	} else {
		idc.dnsZone = zone
	}

	return nil
}

// getHealthyEndpoints returns the hostnames and/or IP addresses of healthy endpoints for the ingress, at a zone, region and global level (or an error)
func (idc *IngressDNSController) getHealthyEndpoints(clusterName string, ingress *extensionsv1beta1.Ingress) (zoneEndpoints, regionEndpoints, globalEndpoints []string, err error) {

	var (
		zoneNames  []string
		regionName string
	)

	if zoneNames, regionName, err = idc.getClusterZoneNames(clusterName); err != nil {
		return nil, nil, nil, err
	}

	// If federated ingress is deleted, return empty endpoints, so that DNS records are removed
	if ingress.DeletionTimestamp != nil {
		return zoneEndpoints, regionEndpoints, globalEndpoints, nil
	}

	globalIngLbStatuses, err := idc.parseGlobalLbStatusAnnotation(ingress)
	if err != nil {
		return nil, nil, nil, err
	}

	for lbClusterName, lbStatuses := range *globalIngLbStatuses {
		lbZoneNames, lbRegionName, err := idc.getClusterZoneNames(lbClusterName)
		if err != nil {
			return nil, nil, nil, err
		}
		for _, lbStatus := range lbStatuses {
			for _, lbIngress := range lbStatus.Ingress {
				var address string
				// We should get either an IP address or a hostname - use whichever one we get
				if lbIngress.IP != "" {
					address = lbIngress.IP
				} else if lbIngress.Hostname != "" {
					address = lbIngress.Hostname
				}
				if len(address) <= 0 {
					return nil, nil, nil, fmt.Errorf("Ingress %s/%s in cluster %s has neither LoadBalancerStatus.ingress.ip nor LoadBalancerStatus.ingress.hostname. Cannot use it as endpoint for federated Ingress",
						ingress.Name, ingress.Namespace, clusterName)
				}
				for _, lbZoneName := range lbZoneNames {
					for _, zoneName := range zoneNames {
						if lbZoneName == zoneName {
							zoneEndpoints = append(zoneEndpoints, address)
						}
					}
				}
				if lbRegionName == regionName {
					regionEndpoints = append(regionEndpoints, address)
				}
				globalEndpoints = append(globalEndpoints, address)
			}
		}
	}

	return zoneEndpoints, regionEndpoints, globalEndpoints, nil
}

// getClusterZoneNames returns the name of the zones (and the region) where the specified cluster exists (e.g. zones "us-east1-c" on GCE, or "us-east-1b" on AWS)
func (idc *IngressDNSController) getClusterZoneNames(clusterName string) ([]string, string, error) {
	cluster, err := idc.federationClient.Federation().Clusters().Get(clusterName, metav1.GetOptions{})
	if err != nil {
		return nil, "", err
	}
	return cluster.Status.Zones, cluster.Status.Region, nil
}

/* ensureDNSRecords ensures (idempotently, and with minimum mutations) that all of the DNS records for a ingress in a given cluster are correct,
given the current state of that ingress in that cluster.  This should be called every time the state of a ingress might have changed
(either w.r.t. its loadbalancer address, or if the number of healthy backend endpoints for that ingress transitioned from zero to non-zero
(or vice versa).  Only shards of the ingress which have both a loadbalancer ingress IP address or hostname AND at least one healthy backend endpoint
are included in DNS records for that ingress (at all of zone, region and global levels). All other addresses are removed.  Also, if no shards exist
in the zone or region of the cluster, a CNAME reference to the next higher level is ensured to exist. */
func (idc *IngressDNSController) ensureDNSRecords(clusterName string, ingress *extensionsv1beta1.Ingress) error {
	// Quinton: Pseudocode....
	// See https://github.com/kubernetes/kubernetes/pull/25107#issuecomment-218026648
	// For each ingress we need the following DNS names:
	// mying.myns.myfed.ing.z1.r1.mydomain.com  (for zone z1 in region r1)
	//         - an A record to IP address of specific shard in that zone (if that shard exists and has healthy endpoints)
	//         - OR a CNAME record to the next level up, i.e. mying.myns.myfed.ing.r1.mydomain.com  (if a healthy shard does not exist in zone z1)
	// mying.myns.myfed.ing.r1.mydomain.com
	//         - a set of A records to IP addresses of all healthy shards in region r1, if one or more of these exist
	//         - OR a CNAME record to the next level up, i.e. mying.myns.myfed.svc.mydomain.com (if no healthy shards exist in region r1)
	// mysvc.myns.myfed.svc.mydomain.com
	//         - a set of A records to IP addresses of all healthy shards in all regions, if one or more of these exist.
	//         - no record (NXRECORD response) if no healthy shards exist in any regions
	//
	// Each ingress has the current known state of loadbalancer ingress for the federated cluster stored in annotations.
	// So generate the DNS records based on the current state and ensure those desired DNS records match the
	// actual DNS records (add new records, remove deleted records, and update changed records).
	//
	ingressName := ingress.Name
	namespaceName := ingress.Namespace

	zoneNames, regionName, err := idc.getClusterZoneNames(clusterName)
	if err != nil {
		return err
	}
	if zoneNames == nil {
		return fmt.Errorf("failed to get cluster zone names")
	}
	zoneEndpoints, regionEndpoints, globalEndpoints, err := idc.getHealthyEndpoints(clusterName, ingress)
	if err != nil {
		return err
	}

	commonPrefix := ingressName + "." + namespaceName + "." + idc.federationName + "." + idc.ingressDNSSuffix
	// dnsNames is the path up the DNS search tree, starting at the leaf
	dnsNames := []string{
		strings.Join([]string{commonPrefix, zoneNames[0], regionName, idc.domain}, "."), // zone level - TODO might need other zone names for multi-zone clusters
		strings.Join([]string{commonPrefix, regionName, idc.domain}, "."),               // region level, one up from zone level
		strings.Join([]string{commonPrefix, idc.domain}, "."),                           // global level, one up from region level
		"", // nowhere to go up from global level
	}

	glog.V(4).Infof("Going to create a set of dns names for Ingres %s -> %v", ingress.Name, dnsNames)

	endpoints := [][]string{zoneEndpoints, regionEndpoints, globalEndpoints}

	glog.V(4).Infof("Got Ingres %s endpoints %v", ingress.Name, endpoints)

	for i, endpoint := range endpoints {
		if err = idc.EnsureDNSRrsets(idc.dnsZone, dnsNames[i], endpoint, dnsNames[i+1]); err != nil {
			return err
		}
	}
	return nil
}

func (idc *IngressDNSController) parseGlobalLbStatusAnnotation(ingress *extensionsv1beta1.Ingress) (*map[string][]v1.LoadBalancerStatus, error) {

	globalLbStat := make(map[string][]v1.LoadBalancerStatus)
	if ingress.Annotations == nil {
		return nil, nil
	}
	globalLbStatString, found := ingress.Annotations[globalIngressLBStatus]
	if !found {
		return nil, nil
	}
	if err := json.Unmarshal([]byte(globalLbStatString), &globalLbStat); err != nil {
		return &globalLbStat, err
	}
	return &globalLbStat, nil
}
