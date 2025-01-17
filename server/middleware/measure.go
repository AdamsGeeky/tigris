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

package middleware

import (
	"context"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	api "github.com/tigrisdata/tigris/api/server/v1"
	"github.com/tigrisdata/tigris/server/metrics"
	"github.com/tigrisdata/tigris/server/request"
	"github.com/tigrisdata/tigris/util"
	ulog "github.com/tigrisdata/tigris/util/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

const (
	TigrisStreamSpan string = "rpcstream"
)

type wrappedStream struct {
	*middleware.WrappedServerStream
	measurement *metrics.Measurement
}

func getNoMeasurementMethods() []string {
	return []string{
		api.HealthMethodName,
	}
}

func measureMethod(fullMethod string) bool {
	for _, method := range getNoMeasurementMethods() {
		if method == fullMethod {
			return false
		}
	}
	return true
}

func measureUnary() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !measureMethod(info.FullMethod) {
			resp, err := handler(ctx, req)
			return resp, err
		}
		reqMetadata, err := request.GetRequestMetadataFromContext(ctx)
		ulog.E(err)
		tags := reqMetadata.GetInitialTags()
		measurement := metrics.NewMeasurement(util.Service, info.FullMethod, metrics.GrpcSpanType, tags)
		measurement.AddTags(metrics.GetDbCollTagsForReq(req))
		ctx = measurement.StartTracing(ctx, false)
		resp, err := handler(ctx, req)
		if err != nil {
			// Request had an error
			measurement.CountErrorForScope(metrics.RequestsErrorCount, measurement.GetRequestErrorTags(err))
			_ = measurement.FinishWithError(ctx, "request", err)
			measurement.RecordDuration(metrics.RequestsErrorRespTime, measurement.GetRequestErrorTags(err))
			return nil, err
		}
		// Request was ok
		measurement.CountOkForScope(metrics.RequestsOkCount, measurement.GetRequestOkTags())
		measurement.CountReceivedBytes(metrics.BytesReceived, measurement.GetNetworkTags(), proto.Size(req.(proto.Message)))
		measurement.CountSentBytes(metrics.BytesSent, measurement.GetNetworkTags(), proto.Size(resp.(proto.Message)))
		_ = measurement.FinishTracing(ctx)
		measurement.RecordDuration(metrics.RequestsRespTime, measurement.GetRequestOkTags())
		return resp, err
	}
}

func measureStream() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapped := &wrappedStream{WrappedServerStream: middleware.WrapServerStream(stream)}
		wrapped.WrappedContext = stream.Context()
		if !measureMethod(info.FullMethod) {
			err := handler(srv, wrapped)
			return err
		}
		reqMetadata, err := request.GetRequestMetadataFromContext(wrapped.WrappedContext)
		if err != nil {
			ulog.E(err)
		}
		tags := reqMetadata.GetInitialTags()
		measurement := metrics.NewMeasurement(util.Service, info.FullMethod, metrics.GrpcSpanType, tags)
		wrapped.measurement = measurement
		wrapped.WrappedContext = measurement.StartTracing(wrapped.WrappedContext, false)
		err = handler(srv, wrapped)
		if err != nil {
			measurement.CountErrorForScope(metrics.RequestsErrorCount, measurement.GetRequestErrorTags(err))
			_ = measurement.FinishWithError(wrapped.WrappedContext, "request", err)
			measurement.RecordDuration(metrics.RequestsErrorRespTime, measurement.GetRequestErrorTags(err))
			ulog.E(err)
			return err
		}
		measurement.CountOkForScope(metrics.RequestsOkCount, measurement.GetRequestOkTags())
		_ = measurement.FinishTracing(wrapped.WrappedContext)
		measurement.RecordDuration(metrics.RequestsRespTime, measurement.GetRequestOkTags())
		return nil
	}
}

func (w *wrappedStream) RecvMsg(m interface{}) error {
	parentMeasurement := w.measurement
	if parentMeasurement == nil {
		err := w.ServerStream.RecvMsg(m)
		return err
	}
	childMeasurement := metrics.NewMeasurement(TigrisStreamSpan, "RecvMsg", metrics.GrpcSpanType, parentMeasurement.GetRequestOkTags())
	w.WrappedContext = childMeasurement.StartTracing(w.WrappedContext, true)
	err := w.ServerStream.RecvMsg(m)
	parentMeasurement.RecursiveAddTags(metrics.GetDbCollTagsForReq(m))
	childMeasurement.RecursiveAddTags(metrics.GetDbCollTagsForReq(m))
	parentMeasurement.CountReceivedBytes(metrics.BytesReceived, parentMeasurement.GetNetworkTags(), proto.Size(m.(proto.Message)))
	w.WrappedContext = childMeasurement.FinishTracing(w.WrappedContext)
	return err
}

func (w *wrappedStream) SendMsg(m interface{}) error {
	parentMeasurement := w.measurement
	if parentMeasurement == nil {
		err := w.ServerStream.SendMsg(m)
		return err
	}
	childMeasurement := metrics.NewMeasurement(TigrisStreamSpan, "SendMsg", metrics.GrpcSpanType, parentMeasurement.GetRequestOkTags())
	w.WrappedContext = childMeasurement.StartTracing(w.WrappedContext, true)
	err := w.ServerStream.SendMsg(m)
	parentMeasurement.RecursiveAddTags(metrics.GetDbCollTagsForReq(m))
	childMeasurement.RecursiveAddTags(metrics.GetDbCollTagsForReq(m))
	parentMeasurement.CountSentBytes(metrics.BytesSent, parentMeasurement.GetNetworkTags(), proto.Size(m.(proto.Message)))
	w.WrappedContext = childMeasurement.FinishTracing(w.WrappedContext)
	return err
}
