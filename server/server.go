package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	proto "handin5/grpc"

	"google.golang.org/grpc"
)

type Server struct {
	proto.UnimplementedBiddingServiceServer
	CurrentHighestBid int32
	AuctionOngoing    bool
	TimesBidded       int32
	bidTimer          *time.Timer
	replicas          []proto.BiddingServiceClient
	NodesId           []string
	mu                sync.Mutex
}

func (s *Server) BeginAuction() {
	s.AuctionOngoing = true
	s.CurrentHighestBid = 0
	s.TimesBidded = 0
	s.resetTimer()
}

func (s *Server) StopAuction() {
	s.AuctionOngoing = false
}

func (s *Server) resetTimer() {
	if s.bidTimer != nil {
		s.bidTimer.Stop()
	}
	s.bidTimer = time.AfterFunc(15*time.Second, func() {
		fmt.Println("No bids received in the last 15 seconds.")
		s.StopAuction()
	})
}

func (s *Server) GetHighestBid(ctx context.Context, req *proto.HighestBidRequest) (*proto.HighestBidResponse, error) {
	return &proto.HighestBidResponse{HighesBid: s.CurrentHighestBid}, nil
}

func (s *Server) GetResult(ctx context.Context, req *proto.ResultRequest) (*proto.ResultResponse, error) {
	return &proto.ResultResponse{WinnerBid: s.CurrentHighestBid}, nil
}

func (s *Server) GetTimesBidded(ctx context.Context, req *proto.TimesBiddedRequest) (*proto.TimesBiddedResponse, error) {
	return &proto.TimesBiddedResponse{TimesBidded: s.TimesBidded}, nil
}

func (s *Server) PlaceBid(ctx context.Context, req *proto.BidRequest) (*proto.BidResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isNodeRegistered(req.NodeID) {
		s.NodesId = append(s.NodesId, req.NodeID)
		fmt.Printf("Node registered: %s\n", req.NodeID)
	}

	if req.Amount <= s.CurrentHighestBid {
		return &proto.BidResponse{
			Ack:     proto.AckStatus_FAIL,
			Comment: "Bid was below Current highest bid",
		}, nil
	}

	s.CurrentHighestBid = req.Amount
	s.TimesBidded++
	s.resetTimer()


	return &proto.BidResponse{
		Ack:     proto.AckStatus_SUCCESS,
		Comment: "The bid was successful",
	}, nil
}

func (s *Server) isNodeRegistered(nodeID string) bool {
	for _, id := range s.NodesId {
		if id == nodeID {
			return true
		}
	}
	return false
}

func (s *Server) Ping(ctx context.Context, req *proto.PingRequest) (*proto.PingResponse, error) {
    return &proto.PingResponse{Alive: true}, nil
}


func (s *Server) GetIsAuctionOngoing(ctx context.Context, req *proto.RequestIsAuctionOngoing) (*proto.ResponseIsAuctionOngoing, error) {
	return &proto.ResponseIsAuctionOngoing{AuctionStillGoing: s.AuctionOngoing}, nil
}


func main() {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	server := &Server{}

	proto.RegisterBiddingServiceServer(grpcServer, server)

	go func() {
		server.BeginAuction()
	}()

	fmt.Println("Primary server started on port 50051")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}