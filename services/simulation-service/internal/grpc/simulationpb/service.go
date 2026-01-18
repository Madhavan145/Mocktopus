// Package simulationpb provides the gRPC service definitions for the simulation service.
// This file serves as a manually authored equivalent of protoc-generated code.
// When protoc tooling is available in CI, regenerate from packages/proto/simulation.proto.
package simulationpb

import (
	"context"

	"google.golang.org/grpc"
)

// --- Messages ---

type TrafficConfig struct {
	DurationSeconds      int32   `protobuf:"varint,1,opt,name=duration_seconds"`
	RequestsPerSecond    int32   `protobuf:"varint,2,opt,name=requests_per_second"`
	MaxRetries           int32   `protobuf:"varint,3,opt,name=max_retries"`
	BaseNetworkLatencyMs float64 `protobuf:"fixed64,4,opt,name=base_network_latency_ms"`
	RetryPenaltyMs       float64 `protobuf:"fixed64,5,opt,name=retry_penalty_ms"`
}

func (t *TrafficConfig) GetDurationSeconds() int32 {
	if t != nil {
		return t.DurationSeconds
	}
	return 0
}
func (t *TrafficConfig) GetRequestsPerSecond() int32 {
	if t != nil {
		return t.RequestsPerSecond
	}
	return 0
}
func (t *TrafficConfig) GetMaxRetries() int32 {
	if t != nil {
		return t.MaxRetries
	}
	return 0
}
func (t *TrafficConfig) GetBaseNetworkLatencyMs() float64 {
	if t != nil {
		return t.BaseNetworkLatencyMs
	}
	return 0
}
func (t *TrafficConfig) GetRetryPenaltyMs() float64 {
	if t != nil {
		return t.RetryPenaltyMs
	}
	return 0
}

type TopologyNode struct {
	Id     string            `protobuf:"bytes,1,opt,name=id"`
	Type   string            `protobuf:"bytes,2,opt,name=type"`
	Config map[string]string `protobuf:"bytes,3,rep,name=config"`
}

func (n *TopologyNode) GetId() string {
	if n != nil {
		return n.Id
	}
	return ""
}
func (n *TopologyNode) GetType() string {
	if n != nil {
		return n.Type
	}
	return ""
}
func (n *TopologyNode) GetConfig() map[string]string {
	if n != nil {
		return n.Config
	}
	return nil
}

type TopologyEdge struct {
	Id     string `protobuf:"bytes,1,opt,name=id"`
	Source string `protobuf:"bytes,2,opt,name=source"`
	Target string `protobuf:"bytes,3,opt,name=target"`
}

func (e *TopologyEdge) GetId() string {
	if e != nil {
		return e.Id
	}
	return ""
}
func (e *TopologyEdge) GetSource() string {
	if e != nil {
		return e.Source
	}
	return ""
}
func (e *TopologyEdge) GetTarget() string {
	if e != nil {
		return e.Target
	}
	return ""
}

type Topology struct {
	Nodes []*TopologyNode `protobuf:"bytes,1,rep,name=nodes"`
	Edges []*TopologyEdge `protobuf:"bytes,2,rep,name=edges"`
}

func (t *Topology) GetNodes() []*TopologyNode {
	if t != nil {
		return t.Nodes
	}
	return nil
}
func (t *Topology) GetEdges() []*TopologyEdge {
	if t != nil {
		return t.Edges
	}
	return nil
}

type StartSimulationRequest struct {
	ProjectId string         `protobuf:"bytes,1,opt,name=project_id"`
	Topology  *Topology      `protobuf:"bytes,2,opt,name=topology"`
	Traffic   *TrafficConfig `protobuf:"bytes,3,opt,name=traffic"`
}

func (r *StartSimulationRequest) GetProjectId() string {
	if r != nil {
		return r.ProjectId
	}
	return ""
}
func (r *StartSimulationRequest) GetTopology() *Topology {
	if r != nil {
		return r.Topology
	}
	return nil
}
func (r *StartSimulationRequest) GetTraffic() *TrafficConfig {
	if r != nil {
		return r.Traffic
	}
	return nil
}

type StartSimulationResponse struct {
	RunId  string `protobuf:"bytes,1,opt,name=run_id"`
	Status string `protobuf:"bytes,2,opt,name=status"`
}

type StopSimulationRequest struct {
	RunId string `protobuf:"bytes,1,opt,name=run_id"`
}

func (r *StopSimulationRequest) GetRunId() string {
	if r != nil {
		return r.RunId
	}
	return ""
}

type StopSimulationResponse struct{}

type ReplayRequest struct {
	RunId string `protobuf:"bytes,1,opt,name=run_id"`
}

func (r *ReplayRequest) GetRunId() string {
	if r != nil {
		return r.RunId
	}
	return ""
}

type ReplayResponse struct {
	RunId  string `protobuf:"bytes,1,opt,name=run_id"`
	Status string `protobuf:"bytes,2,opt,name=status"`
}

// --- Service Interface ---

type SimulationServiceServer interface {
	StartSimulation(context.Context, *StartSimulationRequest) (*StartSimulationResponse, error)
	StopSimulation(context.Context, *StopSimulationRequest) (*StopSimulationResponse, error)
	Replay(context.Context, *ReplayRequest) (*ReplayResponse, error)
}

// UnimplementedSimulationServiceServer provides default implementations for forward compatibility.
type UnimplementedSimulationServiceServer struct{}

func (UnimplementedSimulationServiceServer) StartSimulation(context.Context, *StartSimulationRequest) (*StartSimulationResponse, error) {
	return nil, nil
}
func (UnimplementedSimulationServiceServer) StopSimulation(context.Context, *StopSimulationRequest) (*StopSimulationResponse, error) {
	return nil, nil
}
func (UnimplementedSimulationServiceServer) Replay(context.Context, *ReplayRequest) (*ReplayResponse, error) {
	return nil, nil
}

// --- Service Registration ---

var serviceDesc = grpc.ServiceDesc{
	ServiceName: "moktopus.simulation.v1.SimulationService",
	HandlerType: (*SimulationServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "StartSimulation",
			Handler:    handleStartSimulation,
		},
		{
			MethodName: "StopSimulation",
			Handler:    handleStopSimulation,
		},
		{
			MethodName: "Replay",
			Handler:    handleReplay,
		},
	},
	Streams: []grpc.StreamDesc{},
}

func RegisterSimulationServiceServer(s *grpc.Server, srv SimulationServiceServer) {
	s.RegisterService(&serviceDesc, srv)
}

func handleStartSimulation(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartSimulationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SimulationServiceServer).StartSimulation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/moktopus.simulation.v1.SimulationService/StartSimulation",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SimulationServiceServer).StartSimulation(ctx, req.(*StartSimulationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func handleStopSimulation(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StopSimulationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SimulationServiceServer).StopSimulation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/moktopus.simulation.v1.SimulationService/StopSimulation",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SimulationServiceServer).StopSimulation(ctx, req.(*StopSimulationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func handleReplay(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReplayRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SimulationServiceServer).Replay(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/moktopus.simulation.v1.SimulationService/Replay",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SimulationServiceServer).Replay(ctx, req.(*ReplayRequest))
	}
	return interceptor(ctx, in, info, handler)
}
