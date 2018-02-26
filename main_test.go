package main

import (
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	"testing"
)

func TestKeepEvent(t *testing.T) {

	e := &events.Envelope{EventType: events.Envelope_CounterEvent.Enum()}
	if keepEvent(e,false) {
		t.Error("CounterEvent was not discarded")
	}

	e = &events.Envelope{
		EventType: events.Envelope_ValueMetric.Enum(),
		Origin:    proto.String("gorouter"),
		ValueMetric: &events.ValueMetric{
			Name: proto.String("latency"),
		},
	}
	if keepEvent(e,false) {
		t.Error("Latency ValueMetric from gorouter was not discarded")
	}

	e.ValueMetric.Name = proto.String("latency.foo")
	if keepEvent(e,false) {
		t.Error("Latency ValueMetric from gorouter was not discarded")
	}

	e.ValueMetric.Name = proto.String("route_lookup_time")
	if keepEvent(e,false) {
		t.Error("Latency ValueMetric from gorouter was not discarded")
	}

	e = &events.Envelope{EventType: events.Envelope_ContainerMetric.Enum()}
	if !keepEvent(e,false) {
		t.Error("ContainerMetric was not kept")
	}

	e = &events.Envelope{EventType: events.Envelope_ValueMetric.Enum()}
	if !keepEvent(e,false) {
		t.Error("ValueMetric was not kept")
	}

	e = &events.Envelope{EventType: events.Envelope_ValueMetric.Enum(), Origin: proto.String("gorouter")}
	if !keepEvent(e,false) {
		t.Error("ValueMetric from gorouter was not kept")
	}
	e = &events.Envelope{EventType: events.Envelope_ValueMetric.Enum(), Origin: proto.String("gorouter"),ValueMetric: &events.ValueMetric{Name: proto.String("numCPUS")}}
	if !keepEvent(e,true) {
		t.Error("ValueMetric from gorouter was not kept")
	}
	e = &events.Envelope{EventType: events.Envelope_ValueMetric.Enum(), Origin: proto.String("rep"),ValueMetric: &events.ValueMetric{Name: proto.String("CapacityTotalContainers")}}
	if !keepEvent(e,true) {
		t.Error("ValueMetric from gorouter was not kept")
	}
}
