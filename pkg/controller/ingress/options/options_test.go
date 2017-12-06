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
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	fmt.Printf("Running tests...\n")
	os.Exit(m.Run())
}

func TestNewFICO(t *testing.T) {
	fico := NewFICO()
	fico.AddFlags(pflag.CommandLine)
	// testing a default value
	if fico.DnsProvider != "aws-route53" {
		t.Errorf("Wrong dns provider value %s, should be %s instead", fico.DnsProvider, "aws-route53")
	}
}
