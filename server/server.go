package main

import (
	"context"
	"fmt"
	proto "handin5/grpc"
	"handin5/node"
	"net"

	"google.golang.org/grpc"
)

type Server struct {
	proto.UnimplementedBiddingServiceServer;
	CurrentHighestBid int32;
	AuctionOngoing bool;
	Nodes []node.Node;
}

func (s Server)BeginAuction() {
	//Starts the auction and assigns default values
	s.AuctionOngoing = true
	s.CurrentHighestBid = 0;

}

func (s *Server) GetHighestBid(ctx context.Context, req *proto.HighestBidRequest) (*proto.HighestBidResponse, error) {
    // Return the current highest bid
    return &proto.HighestBidResponse{HighesBid: s.CurrentHighestBid}, nil
}

func (s *Server) PlaceBid(ctx context.Context, req *proto.BidRequest) (*proto.BidResponse,error) {
	if req.Amount < s.CurrentHighestBid {
		return &proto.BidResponse{
			Ack: proto.AckStatus_FAIL, // Set the Ack status to FAIL
			Comment:"Bid was below Current highest bid", // Set the comment
		}, nil
	}

	s.CurrentHighestBid = req.Amount
	return &proto.BidResponse{
		Ack: proto.AckStatus_SUCCESS, // Set the Ack status to FAIL
		Comment:"The bid wass successful", // Set the comment
	}, nil

}

func main () {
	// Create a listener on port 50051
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Printf("failed to listen: %v", err)
		return
	}

	// Create a new gRPC server
	grpcServer := grpc.NewServer()
	s := &Server{}

	proto.RegisterBiddingServiceServer(grpcServer, s)

	fmt.Println("Server started on port 50051")
	if err := grpcServer.Serve(lis); err != nil {
		fmt.Printf("failed to serve: %v", err)
	}

	


	s.BeginAuction()
}