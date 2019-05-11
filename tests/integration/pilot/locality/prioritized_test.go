// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package locality

import (
	"fmt"
	"testing"

	"istio.io/common/pkg/log"
	"istio.io/istio/pkg/test/framework"
	"istio.io/istio/pkg/test/framework/components/environment"
	"istio.io/istio/pkg/test/framework/components/namespace"
)

// This test allows for Locality Prioritized Load Balancing testing without needing Kube nodes in multiple regions.
// We do this by overriding the calling service's locality with the istio-locality label and the serving
// service's locality with a service entry.
//
// The created Service Entry for fake-(cds|eds)-external-service-12345.com points the domain at services
// that exist internally in the mesh. In the Service Entry we set service B to be in the same locality
// as the caller (A) and service C to be in a different locality. We then verify that all request go to
// service B. For details check the Service Entry configuration at the bottom of the page.
//
//  CDS Test
//
//                                                                 +-> b (region.zone.subzone)
//                                                                 |
//                                                            100% |
//                                                                 |
// A (region.zone.subzone) -> fake-cds-external-service-12345.com -|
//                                                                 |
//                                                              0% |
//                                                                 |
//                                                                 +-> c (notregion.notzone.notsubzone)
//
//
//  EDS Test
//
//                                                                 +-> 10.28.1.138 (b -> region.zone.subzone)
//                                                                 |
//                                                            100% |
//                                                                 |
// A (region.zone.subzone) -> fake-eds-external-service-12345.com -|
//                                                                 |
//                                                              0% |
//                                                                 |
//                                                                 +-> 10.28.1.139 (c -> notregion.notzone.notsubzone)

func TestPrioritized(t *testing.T) {
	// TODO(liamawhite): Investigate why it fails in Parallel.
	//t.Parallel()

	t.Run("CDS", func(t *testing.T) {
		t.Parallel()

		framework.NewTest(t).
			RequiresEnvironment(environment.Kube).
			Run(func(ctx framework.TestContext) {

				ns := namespace.NewOrFail(t, ctx, "failover-eds", true)
				a := newEcho(t, ctx, ns, "a")
				b := newEcho(t, ctx, ns, "b")
				c := newEcho(t, ctx, ns, "c")
				a.WaitUntilReadyOrFail(t, b, c)

				fakeHostname := fmt.Sprintf("fake-cds-external-service-%v.com", r.Int())
				deploy(t, ns, serviceConfig{
					Name:             "prioritized-cds",
					Host:             fakeHostname,
					Namespace:        ns.Name(),
					Resolution:       "DNS",
					ServiceBAddress:  "b",
					ServiceBLocality: "region/zone/subzone",
					ServiceCAddress:  "c",
					ServiceCLocality: "notregion/notzone/notsubzone",
				})

				// Send traffic to service B via a service entry.
				log.Infof("Sending traffic to local service (CDS) via %v", fakeHostname)
				sendTraffic(t, a, fakeHostname)
			})
	})

	t.Run("EDS", func(t *testing.T) {
		t.Parallel()

		framework.NewTest(t).
			RequiresEnvironment(environment.Kube).
			Run(func(ctx framework.TestContext) {

				ns := namespace.NewOrFail(t, ctx, "failover-eds", true)
				a := newEcho(t, ctx, ns, "a")
				b := newEcho(t, ctx, ns, "b")
				c := newEcho(t, ctx, ns, "c")
				a.WaitUntilReadyOrFail(t, b, c)

				fakeHostname := fmt.Sprintf("fake-eds-external-service-%v.com", r.Int())
				deploy(t, ns, serviceConfig{
					Name:             "prioritized-eds",
					Host:             fakeHostname,
					Namespace:        ns.Name(),
					Resolution:       "STATIC",
					ServiceBAddress:  b.WorkloadsOrFail(t)[0].Address(),
					ServiceBLocality: "region/zone/subzone",
					ServiceCAddress:  c.WorkloadsOrFail(t)[0].Address(),
					ServiceCLocality: "notregion/notzone/notsubzone",
				})

				// Send traffic to service B via a service entry.
				log.Infof("Sending traffic to local service (EDS) via %v", fakeHostname)
				sendTraffic(t, a, fakeHostname)
			})
	})
}
