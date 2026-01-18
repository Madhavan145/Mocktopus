package grpc

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"moktopus/simulation-service/internal/engine"
	"moktopus/simulation-service/internal/model"
	"moktopus/simulation-service/internal/store"

	pb "moktopus/simulation-service/internal/grpc/simulationpb"
)

// Server wraps the gRPC server for the simulation service.
type Server struct {
	grpcServer *grpc.Server
	port       string
}

// simulationServiceServer implements the SimulationService gRPC service.
type simulationServiceServer struct {
	pb.UnimplementedSimulationServiceServer
	engine engine.Engine
	store  store.Store
}

// NewServer creates a new gRPC server listening on the given port.
func NewServer(port string, eng engine.Engine, st store.Store) *Server {
	srv := grpc.NewServer()

	svc := &simulationServiceServer{
		engine: eng,
		store:  st,
	}
	pb.RegisterSimulationServiceServer(srv, svc)
	reflection.Register(srv)

	return &Server{
		grpcServer: srv,
		port:       port,
	}
}

// Serve starts the gRPC server. Blocks until Stop is called.
func (s *Server) Serve() error {
	lis, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		return fmt.Errorf("grpc listen: %w", err)
	}
	log.Printf("gRPC server listening on :%s", s.port)
	return s.grpcServer.Serve(lis)
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}

// StartSimulation handles the StartSimulation RPC.
func (s *simulationServiceServer) StartSimulation(ctx context.Context, req *pb.StartSimulationRequest) (*pb.StartSimulationResponse, error) {
	if req.GetTopology() == nil || len(req.GetTopology().GetNodes()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "topology with at least one node is required")
	}
	if req.GetTraffic() == nil {
		return nil, status.Error(codes.InvalidArgument, "traffic config is required")
	}

	// Convert proto types to internal model types.
	runReq := model.RunRequest{
		ProjectID: req.GetProjectId(),
		Topology:  protoToTopology(req.GetTopology()),
		Traffic:   protoToTraffic(req.GetTraffic()),
	}

	resp, err := s.engine.Run(runReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "simulation failed: %v", err)
	}

	runID := fmt.Sprintf("run_%d", resp.Metadata.SeedUsed)

	return &pb.StartSimulationResponse{
		RunId:  runID,
		Status: "completed",
	}, nil
}

// StopSimulation handles the StopSimulation RPC.
func (s *simulationServiceServer) StopSimulation(ctx context.Context, req *pb.StopSimulationRequest) (*pb.StopSimulationResponse, error) {
	if req.GetRunId() == "" {
		return nil, status.Error(codes.InvalidArgument, "run_id is required")
	}
	// Currently simulations run synchronously, so stop is a no-op acknowledgement.
	return &pb.StopSimulationResponse{}, nil
}

// Replay handles the Replay RPC.
func (s *simulationServiceServer) Replay(ctx context.Context, req *pb.ReplayRequest) (*pb.ReplayResponse, error) {
	if req.GetRunId() == "" {
		return nil, status.Error(codes.InvalidArgument, "run_id is required")
	}
	return &pb.ReplayResponse{
		RunId:  req.GetRunId(),
		Status: "replaying",
	}, nil
}

// --- Converters ---

func protoToTopology(t *pb.Topology) model.Topology {
	nodes := make([]model.TopologyNode, 0, len(t.GetNodes()))
	for _, n := range t.GetNodes() {
		cfg := make(map[string]any, len(n.GetConfig()))
		for k, v := range n.GetConfig() {
			cfg[k] = v
		}
		nodes = append(nodes, model.TopologyNode{
			ID:     n.GetId(),
			Type:   n.GetType(),
			Config: cfg,
		})
	}
	edges := make([]model.TopologyEdge, 0, len(t.GetEdges()))
	for _, e := range t.GetEdges() {
		edges = append(edges, model.TopologyEdge{
			ID:     e.GetId(),
			Source: e.GetSource(),
			Target: e.GetTarget(),
		})
	}
	return model.Topology{Nodes: nodes, Edges: edges}
}

func protoToTraffic(t *pb.TrafficConfig) model.TrafficConfig {
	return model.TrafficConfig{
		DurationSeconds:      int(t.GetDurationSeconds()),
		RequestsPerSecond:    int(t.GetRequestsPerSecond()),
		MaxRetries:           int(t.GetMaxRetries()),
		BaseNetworkLatencyMs: t.GetBaseNetworkLatencyMs(),
		RetryPenaltyMs:       t.GetRetryPenaltyMs(),
	}
}
