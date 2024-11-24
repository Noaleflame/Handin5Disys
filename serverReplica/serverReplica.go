package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	proto "handin5/grpc"

	"google.golang.org/grpc"
)

type ReplicaServer struct {
	proto.UnimplementedBiddingServiceServer
	CurrentHighestBid int32
	primaryClient     proto.BiddingServiceClient
	isLeader          bool
	TimesBidded int32
	mu                sync.Mutex
	bidTimer          *time.Timer
	AuctionOngoing    bool
	logger      *log.Logger



}

func (s *ReplicaServer) heartbeat() {

	
    retryCount := 0
    maxRetries := 3 // Max retries before triggering leader election
    for {
        _, err := s.primaryClient.Ping(context.Background(), &proto.PingRequest{})
        if err != nil {
            fmt.Println("Primary not responding.")
			s.logger.Printf("Primary not responding.")


            // Retry logic
            if retryCount < maxRetries {
                fmt.Printf("Retrying... (%d/%d)\n", retryCount+1, maxRetries)
				s.logger.Printf("Retrying... (%d/%d)\n", retryCount+1, maxRetries)
                retryCount++
                time.Sleep(2 * time.Second) // Wait before retry
                continue
            }

            // After max retries, initiate leader election
            fmt.Println("Initiating leader election...")
			s.logger.Printf("Initiating leader election...")
            s.initiateLeaderElection()
            return
		}
        retryCount = 0 // Reset retry count if ping is successful
        time.Sleep(2 * time.Second) // Heartbeat interval
    }
}

func (s *ReplicaServer) GetIsAuctionOngoing(ctx context.Context, req *proto.RequestIsAuctionOngoing) (*proto.ResponseIsAuctionOngoing, error) {
	return &proto.ResponseIsAuctionOngoing{AuctionStillGoing: s.AuctionOngoing}, nil
}

func (s *ReplicaServer) GetResult(ctx context.Context, req *proto.ResultRequest) (*proto.ResultResponse, error) {
	return &proto.ResultResponse{WinnerBid: s.CurrentHighestBid}, nil
}


func (s *ReplicaServer) requestPrimaryUpdates() {
	// This will request the primary server to get the current highest bid and times bidded
	for {

		_, err := s.primaryClient.Ping(context.Background(), &proto.PingRequest{})
        if err != nil {
            fmt.Println("Primary server is not reachable. Skipping updates. Error:", err)
            // Wait before retrying connection
            time.Sleep(2 * time.Second)
            return
        }

		// Get highest bid and times bidded from primary server
		highestBidResp, err := s.primaryClient.GetHighestBid(context.Background(), &proto.HighestBidRequest{})
		if err != nil {
			fmt.Println("Error getting highest bid:", err)
			time.Sleep(2 * time.Second) // Wait before retrying
			continue
		}

		timesBiddedResp, err := s.primaryClient.GetTimesBidded(context.Background(), &proto.TimesBiddedRequest{})
		if err != nil {
			fmt.Println("Error getting times bidded:", err)
			time.Sleep(2 * time.Second) // Wait before retrying
			continue
		}

		// Update the replica with the fetched data
		s.mu.Lock()
		s.CurrentHighestBid = highestBidResp.HighesBid
		s.TimesBidded = timesBiddedResp.TimesBidded
		s.mu.Unlock()


		// Wait before the next update request
	}
}




func (s *ReplicaServer) initiateLeaderElection() {
	s.mu.Lock()
	defer s.mu.Unlock()

	fmt.Println("Replica is taking over as the new leader...")
	s.logger.Printf("Replica is taking over as the new leader...")
	s.isLeader = true
	s.primaryClient = nil
}
func (s *ReplicaServer) GetTimesBidded(ctx context.Context, req *proto.TimesBiddedRequest) (*proto.TimesBiddedResponse, error) {
	return &proto.TimesBiddedResponse{TimesBidded: s.TimesBidded}, nil
}

func (s *ReplicaServer) PlaceBid(ctx context.Context, req *proto.BidRequest) (*proto.BidResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	

	if req.Amount <= s.CurrentHighestBid {
		return &proto.BidResponse{
			Ack:     proto.AckStatus_FAIL,
			Comment: "Bid was below the current highest bid",
		}, nil
	}

	s.TimesBidded++
	s.resetTimer()
	fmt.Print("TimesBidded ")
	fmt.Println(s.TimesBidded)
	s.CurrentHighestBid = req.Amount
	fmt.Printf("New highest bid: %d by Node: %s\n", s.CurrentHighestBid, req.NodeID)

	return &proto.BidResponse{
		Ack:     proto.AckStatus_SUCCESS,
		Comment: "Bid was successful",
	}, nil
}

func (s *ReplicaServer) resetTimer() {
	if s.bidTimer != nil {
		s.bidTimer.Stop()
	}
	s.bidTimer = time.AfterFunc(15*time.Second, func() {
		fmt.Println("No bids received in the last 15 seconds.")
		s.logger.Printf("No bids received in the last 15 seconds.")
		
		s.StopAuction()
	})
}
func (s *ReplicaServer) StopAuction() {
	s.AuctionOngoing = false
	fmt.Printf("Highest bid was: %d\n", s.CurrentHighestBid)
}

func (s *ReplicaServer) GetHighestBid(ctx context.Context, req *proto.HighestBidRequest) (*proto.HighestBidResponse, error) {
	return &proto.HighestBidResponse{HighesBid: s.CurrentHighestBid}, nil
}
func (s *ReplicaServer) Ping(ctx context.Context, req *proto.PingRequest) (*proto.PingResponse, error) {
    return &proto.PingResponse{Alive: true}, nil
}

func main() {
	logFile, err := os.OpenFile("serverLogUser.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer logFile.Close()

	// Creates a new logger that writes to the file
	logger := log.New(logFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	primaryAddress := "localhost:50051"

	conn, err := grpc.Dial(primaryAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to primary: %v", err)
	}
	defer conn.Close()

	primaryClient := proto.NewBiddingServiceClient(conn)

	replica := &ReplicaServer{
		primaryClient: primaryClient,
		isLeader:      false,
		AuctionOngoing: true,
		TimesBidded: 0,
	}

	go replica.heartbeat()
	go replica.requestPrimaryUpdates()


	listener, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterBiddingServiceServer(grpcServer, replica)

	fmt.Println("Replica server started on port 50052")
	logger.Printf("Replica server started on port 50052")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
