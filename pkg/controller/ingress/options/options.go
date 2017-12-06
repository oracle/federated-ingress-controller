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

package options

import (
	"fmt"
	"github.com/spf13/pflag"
	"k8s.io/kubernetes/federation/pkg/dnsprovider"
)

// FederatedIngressControllerOptions stores the values of the command line options
type FederatedIngressControllerOptions struct {
	FkubeApiServer    string
	FkubeName         string
	DnsProvider       string
	DnsProviderConfig string
	Domain            string
	IngressDnsSuffix  string
}

// NewFICO returns an initialized FederatedIngressControllerOptions with default values
func NewFICO() *FederatedIngressControllerOptions {
	fico := FederatedIngressControllerOptions{
		DnsProvider:       "aws-route53",
		DnsProviderConfig: "",
		Domain:            "d1.example.com",
		FkubeApiServer:    "https://mkube-apiserver",
		FkubeName:         "akube",
		IngressDnsSuffix:  "ing",
	}
	return &fico
}

// AddFlags populates FederatedIngressControllerOptions with user specified command line options
func (fico *FederatedIngressControllerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&fico.DnsProvider, "dnsProvider", fico.DnsProvider,
		"DNS provider. Valid values are: "+fmt.Sprintf("%q", dnsprovider.RegisteredDnsProviders()))
	fs.StringVar(&fico.DnsProviderConfig, "dnsProviderConfig", fico.DnsProviderConfig,
		"provider config via config volume filename")
	fs.StringVar(&fico.Domain, "domain", fico.Domain, "DNS Domain")
	fs.StringVar(&fico.FkubeApiServer, "fkubeApiServer", fico.FkubeApiServer, "Federation API host")
	fs.StringVar(&fico.FkubeName, "fkubeName", fico.FkubeName, "Federation Name")
	fs.StringVar(&fico.IngressDnsSuffix, "ingressDnsSuffix", fico.IngressDnsSuffix,
		"Ingress DNS Suffix (part between domain and ingress name + clusters) defaults to ing")
}
