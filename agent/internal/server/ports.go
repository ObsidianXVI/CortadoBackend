package server

import (
	"context"
	"sort"
	"strconv"
	"time"

	pb "github.com/your-org/cortado/agent/gen/agent/v1"
	portmonitor "github.com/your-org/cortado/agent/internal/ports"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *AgentServer) ListPorts(ctx context.Context, req *pb.ListPortsRequest) (*pb.ListPortsResponse, error) {
	ports, err := s.portMonitor.List()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list ports: %v", err)
	}

	return &pb.ListPortsResponse{
		Ports: portsToProto(ports),
	}, nil
}

func (s *AgentServer) WatchPorts(req *pb.WatchPortsRequest, stream pb.WorkspaceAgentService_WatchPortsServer) error {
	previousPorts, err := s.portMonitor.List()
	if err != nil {
		return status.Errorf(codes.Internal, "list ports: %v", err)
	}
	previous := mapPorts(previousPorts)

	ticker := time.NewTicker(s.portPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			currentPorts, err := s.portMonitor.List()
			if err != nil {
				return status.Errorf(codes.Internal, "watch ports: %v", err)
			}

			current := mapPorts(currentPorts)
			for _, event := range diffPorts(previous, current) {
				if err := stream.Send(event); err != nil {
					return err
				}
			}
			previous = current
		}
	}
}

func diffPorts(previous, current map[string]portmonitor.Port) []*pb.PortEvent {
	events := make([]*pb.PortEvent, 0, len(previous)+len(current))

	for key, port := range current {
		if _, ok := previous[key]; ok {
			continue
		}
		events = append(events, &pb.PortEvent{
			Type: pb.PortEventType_PORT_EVENT_TYPE_ADDED,
			Port: portToProto(port),
		})
	}
	for key, port := range previous {
		if _, ok := current[key]; ok {
			continue
		}
		events = append(events, &pb.PortEvent{
			Type: pb.PortEventType_PORT_EVENT_TYPE_REMOVED,
			Port: portToProto(port),
		})
	}

	sort.Slice(events, func(i, j int) bool {
		left := events[i].GetPort()
		right := events[j].GetPort()
		if left.GetPort() != right.GetPort() {
			return left.GetPort() < right.GetPort()
		}
		if left.GetNetwork() != right.GetNetwork() {
			return left.GetNetwork() < right.GetNetwork()
		}
		if events[i].GetType() != events[j].GetType() {
			return events[i].GetType() < events[j].GetType()
		}
		return left.GetHost() < right.GetHost()
	})

	return events
}

func mapPorts(ports []portmonitor.Port) map[string]portmonitor.Port {
	mapped := make(map[string]portmonitor.Port, len(ports))
	for _, port := range ports {
		mapped[portKey(port)] = port
	}
	return mapped
}

func portKey(port portmonitor.Port) string {
	return port.Network + "|" + port.Host + "|" + strconv.FormatUint(uint64(port.Port), 10)
}

func portsToProto(ports []portmonitor.Port) []*pb.PortInfo {
	response := make([]*pb.PortInfo, 0, len(ports))
	for _, port := range ports {
		response = append(response, portToProto(port))
	}
	return response
}

func portToProto(port portmonitor.Port) *pb.PortInfo {
	return &pb.PortInfo{
		Host:    port.Host,
		Network: port.Network,
		Port:    port.Port,
	}
}
