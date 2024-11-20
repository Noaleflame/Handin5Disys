package main

import (
	"context"
	"fmt"
	proto "handin5/grpc"
	"net"
	"time"

	"google.golang.org/grpc"
)

type Server struct {
	proto.UnimplementedBiddingServiceServer;
	CurrentHighestBid int32;
	AuctionOngoing bool;
	TimesBidded int32;
	bidTimer          *time.Timer
    replicas          []proto.BiddingServiceClient

	//Nodes []node.Node;
}

func (s Server)BeginAuction() {
	//Starts the auction and assigns default values
	s.AuctionOngoing = true
	s.CurrentHighestBid = 0;
	s.TimesBidded = 0;
	s.resetTimer()


}

func(s Server)StopAuction() {
	s.AuctionOngoing = false
	fmt.Print("Highest bid was: ")
	fmt.Print(s.CurrentHighestBid)
}
func (s *Server) resetTimer() {
	if s.bidTimer != nil {
		s.bidTimer.Stop() // Stop any existing timer
	}

	// Create a new timer that will call StopAuction after 5 seconds
	s.bidTimer = time.AfterFunc(15*time.Second, func() {
		fmt.Println("No bids received in the last 8 seconds.")
		s.StopAuction()
	})
}

func (s *Server) GetHighestBid(ctx context.Context, req *proto.HighestBidRequest) (*proto.HighestBidResponse, error) {
    // Return the current highest bid
    return &proto.HighestBidResponse{HighesBid: s.CurrentHighestBid}, nil
}

func (s *Server) GetTimesBidded(ctx context.Context, req *proto.TimesBiddedRequest) (*proto.TimesBiddedResponse, error){
	return &proto.TimesBiddedResponse{TimesBidded: s.TimesBidded}, nil
}

func (s *Server) PlaceBid(ctx context.Context, req *proto.BidRequest) (*proto.BidResponse,error) {
	if req.Amount <= s.CurrentHighestBid {
		return &proto.BidResponse{
			Ack: proto.AckStatus_FAIL, // Set the Ack status to FAIL
			Comment:"Bid was below Current highest bid", // Set the comment
		}, nil
	}

	s.CurrentHighestBid = req.Amount
	s.TimesBidded++
	s.resetTimer()
	return &proto.BidResponse{
		Ack: proto.AckStatus_SUCCESS, // Set the Ack status to FAIL
		Comment:"The bid was successful", // Set the comment
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

	go func() {
		// Start the auction after the server is running
		s.BeginAuction()
	}()

	fmt.Println("Server started on port 50051")
	if err := grpcServer.Serve(lis); err != nil {
		fmt.Printf("failed to serve: %v", err)
	}

	

}