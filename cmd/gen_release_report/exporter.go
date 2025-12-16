package main

import (
	"context"
	"encoding/json"
	"io"
	"strconv"
	"sync"

	"go.opentelemetry.io/otel/sdk/trace"
)

type traceAttributeValue struct {
	StringValue string `json:"stringValue"`
}

type traceAttribute struct {
	Key   string              `json:"key"`
	Value traceAttributeValue `json:"value"`
}

type traceResource struct {
	Attributes []traceAttribute `json:"attributes"`
}

type traceSpan struct {
	TraceID           string           `json:"traceId"`
	SpanID            string           `json:"spanId"`
	ParentSpanID      string           `json:"parentSpanId"`
	Name              string           `json:"name"`
	Kind              string           `json:"kind"`
	StartTimeUnixNano string           `json:"startTimeUnixNano"`
	EndTimeUnixNano   string           `json:"endTimeUnixNano"`
	Attributes        []traceAttribute `json:"attributes"`
}

type traceInstrumentationLibrarySpan struct {
	Spans []traceSpan `json:"spans"`
}

type traceBatch struct {
	Resource                    traceResource                     `json:"resource"`
	InstrumentationLibrarySpans []traceInstrumentationLibrarySpan `json:"instrumentationLibrarySpans"`
}

type traceJSON struct {
	Batches []traceBatch `json:"batches"`
}

type GrafanaJSONTraceExporter struct {
	mux sync.Mutex
	w   io.Writer
}

// ExportSpans formats the provided spans in JSON compatible with Grafana
func (x *GrafanaJSONTraceExporter) ExportSpans(ctx context.Context, roSpans []trace.ReadOnlySpan) error {
	x.mux.Lock()
	defer x.mux.Unlock()
	enc := json.NewEncoder(x.w)

	// each span is a separate batch to enable spans with descriptive service names
	batches := make([]traceBatch, 0, len(roSpans))

	for _, span := range roSpans {
		ts := traceSpan{
			TraceID:           span.SpanContext().TraceID().String(),
			SpanID:            span.SpanContext().SpanID().String(),
			ParentSpanID:      span.Parent().SpanID().String(),
			Name:              span.Name(),
			Kind:              span.SpanKind().String(),
			StartTimeUnixNano: strconv.FormatInt(span.StartTime().UnixNano(), 10),
			EndTimeUnixNano:   strconv.FormatInt(span.EndTime().UnixNano(), 10),
			Attributes:        make([]traceAttribute, 0, len(span.Attributes())),
		}
		for _, a := range span.Attributes() {
			ts.Attributes = append(ts.Attributes, traceAttribute{
				Key:   string(a.Key),
				Value: traceAttributeValue{StringValue: a.Value.AsString()},
			})
		}

		// set the service name for the batch
		service := "ecm-distro-tools"
		for _, a := range ts.Attributes {
			if a.Key == "service" {
				service = a.Value.StringValue
			}
		}
		batch := traceBatch{
			Resource: traceResource{Attributes: []traceAttribute{
				{Key: "service.name", Value: traceAttributeValue{StringValue: service}},
			}},
			InstrumentationLibrarySpans: []traceInstrumentationLibrarySpan{
				{Spans: []traceSpan{ts}},
			},
		}

		batches = append(batches, batch)
	}

	return enc.Encode(traceJSON{Batches: batches})
}

func (x *GrafanaJSONTraceExporter) Shutdown(ctx context.Context) error {
	return nil
}

func makeGrafanaJSONTraceExporter(w io.Writer) (trace.SpanExporter, error) {
	return &GrafanaJSONTraceExporter{w: w}, nil
}
