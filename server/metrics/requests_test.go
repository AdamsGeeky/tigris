// Copyright 2022 Tigris Data, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestGRPCMetrics(t *testing.T) {
	InitializeMetrics()
	svcName := "tigrisdata.v1.Tigris"
	unaryMethodName := "TestUnaryMethod"
	streamingMethodName := "TestStreamingMethod"

	unaryMethodInfo := grpc.MethodInfo{
		Name:           unaryMethodName,
		IsServerStream: false,
		IsClientStream: false,
	}
	streamingMethodInfo := grpc.MethodInfo{
		Name:           streamingMethodName,
		IsServerStream: true,
		IsClientStream: false,
	}

	unaryEndPointMetadata := newRequestEndpointMetadata(svcName, unaryMethodInfo)
	streamingEndpointMetadata := newRequestEndpointMetadata(svcName, streamingMethodInfo)

	t.Run("Test full method names", func(t *testing.T) {
		assert.Equal(t, unaryEndPointMetadata.getFullMethod(), fmt.Sprintf("/%s/%s", svcName, unaryMethodName))
		assert.Equal(t, streamingEndpointMetadata.getFullMethod(), fmt.Sprintf("/%s/%s", svcName, streamingMethodName))
	})

	t.Run("Test unary endpoint metadata", func(t *testing.T) {
		assert.Equal(t, unaryEndPointMetadata.serviceName, svcName)
		assert.Equal(t, unaryEndPointMetadata.methodInfo.Name, unaryMethodName)
		assert.False(t, unaryEndPointMetadata.methodInfo.IsServerStream)
	})

	t.Run("Test streaming endpoint metadata", func(t *testing.T) {
		assert.Equal(t, streamingEndpointMetadata.serviceName, svcName)
		assert.Equal(t, streamingEndpointMetadata.methodInfo.Name, streamingMethodName)
		assert.True(t, streamingEndpointMetadata.methodInfo.IsServerStream)
	})

	t.Run("Test unary method preinitialized tags", func(t *testing.T) {
		tags := unaryEndPointMetadata.GetPreInitializedTags()
		for tagName, tagValue := range tags {
			switch tagName {
			case "tigris_server_request_method":
				assert.Equal(t, tagValue, "TestUnaryMethod")
			case "tigris_server_request_service_name":
				assert.Equal(t, tagValue, "tigrisdata.v1.Tigris")
			case "tigris_server_request_type":
				assert.Equal(t, tagValue, "unary")
			}
		}
	})

	t.Run("Test streaming method preinitialized tags", func(t *testing.T) {
		tags := streamingEndpointMetadata.GetPreInitializedTags()
		for tagName, tagValue := range tags {
			switch tagName {
			case "tigris_server_request_method":
				assert.Equal(t, tagValue, "TestStreamingMethod")
			case "tigris_server_request_service_name":
				assert.Equal(t, tagValue, "tigrisdata.v1.Tigris")
			case "tigris_server_request_type":
				assert.Equal(t, tagValue, "stream")
			}
		}
	})

	t.Run("Test metrics initialization", func(t *testing.T) {
		InitServerRequestMetrics(svcName, unaryMethodInfo)
		InitServerRequestMetrics(svcName, streamingMethodInfo)
	})

	t.Run("Test specific error tags", func(t *testing.T) {
		error_tags := unaryEndPointMetadata.GetSpecificErrorTags("test_source", "test_code")
		assert.Equal(t, error_tags["tigris_server_request_error_source"], "test_source")
		assert.Equal(t, error_tags["tigris_server_request_error_code"], "test_code")
	})
}