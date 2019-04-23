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

package kube

import (
	"fmt"
	"strings"

	envoyAdmin "github.com/envoyproxy/go-control-plane/envoy/admin/v2alpha"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"

	"istio.io/istio/pkg/test/framework/components/echo"
	"istio.io/istio/pkg/test/framework/components/echo/common"
	"istio.io/istio/pkg/test/kube"
	"istio.io/istio/pkg/test/util/retry"
)

const (
	proxyContainerName = "istio-proxy"
	proxyAdminPort     = 15000
)

var _ echo.Sidecar = &sidecar{}

type sidecar struct {
	podNamespace string
	podName      string
	accessor     *kube.Accessor
}

func (s *sidecar) Info() (*envoyAdmin.ServerInfo, error) {
	msg := &envoyAdmin.ServerInfo{}
	if err := s.adminRequest("server_info", msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *sidecar) Config() (*envoyAdmin.ConfigDump, error) {
	msg := &envoyAdmin.ConfigDump{}
	if err := s.adminRequest("config_dump", msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *sidecar) WaitForConfig(accept func(*envoyAdmin.ConfigDump) (bool, error), options ...retry.Option) error {
	return common.WaitForConfig(s.Config, accept, options...)
}

func (s *sidecar) adminRequest(path string, out proto.Message) error {
	// Exec onto the pod and make a curl request to the admin port, writing
	command := fmt.Sprintf("curl http://127.0.0.1:%d/%s", proxyAdminPort, path)
	response, err := s.accessor.Exec(s.podNamespace, s.podName, proxyContainerName, command)
	if err != nil {
		return fmt.Errorf("failed exec on pod %s/%s: %v. Command: %s. Output:\n%s",
			s.podNamespace, s.podName, err, command, response)
	}

	if err := jsonpb.Unmarshal(strings.NewReader(response), out); err != nil {
		return fmt.Errorf("failed parsing Envoy admin response from '/%s': %v\nResponse JSON: %s", path, err, response)
	}
	return nil
}
