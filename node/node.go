package main

import (
	"context"
	"fmt"
	proto "handin5/grpc"
	"log"
	"math/rand"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Node struct {
	NodeID            string
	CurrentHighestBid int32
	CurrentBid        int32
	Balance           int32
	ParticipatingInAuction bool
	TimesBidded       int32
	client            proto.BiddingServiceClient
	conn              *grpc.ClientConn
	isAuctionOngoing bool
}

func randomIntBetween(min int32, max int32) int32 {
	if min >= max {
		panic("Invalid range: min must be less than max")
	}
	rand.Seed(time.Now().UnixNano())
	var randomNumber int32
	randomNumber = min + rand.Int31n(max-min+1)
	return randomNumber
}

func (n *Node) PlaceBid() {
	var amount int32

	if n.TimesBidded == 0 {
		// Start bid amount will be between 0 and half of the node's balance
		// Makes sure that the node doesn't bid all its money at first
		amount = randomIntBetween(0, n.Balance/2)
	} else {
		// Else bid amount will be between currentHighestBid and balance
		amount = randomIntBetween(n.CurrentHighestBid, n.CurrentHighestBid+100)
	}
	resp, err := n.client.PlaceBid(context.Background(), &proto.BidRequest{Amount: amount, NodeID: n.NodeID})
	if err != nil {
		log.Fatalf("failed to place bid")
	}
	// Switch case for the ack response
	switch resp.Ack {
	case proto.AckStatus_SUCCESS:
		n.TimesBidded++
		n.CurrentBid = amount
	case proto.AckStatus_FAIL:
		fmt.Printf("Bid failed: %s\n", resp.Comment)
		fmt.Print("Bid failed with this amount ")
		fmt.Println(amount)
	case proto.AckStatus_EXCEPTION:
		fmt.Printf("Exception occurred: %s\n", resp.Comment)
	default:
		fmt.Printf("Unknown response: %s\n", resp.Comment)
	}
}

func (n *Node) GetHighestBid() {
	resp, err := n.client.GetHighestBid(context.Background(), &proto.HighestBidRequest{})
	if err != nil {
		log.Fatalf("failed to get highest bid: %v", err)
	}

	n.CurrentHighestBid = resp.GetHighesBid()
}


func (n *Node) GetIsAuctionOngoing() bool {
	resp, err := n.client.GetIsAuctionOngoing(context.Background(), &proto.RequestIsAuctionOngoing{})
	if err != nil {
		log.Fatalf("failed to get auction ongoing status: %v", err)
	}
	// Set the auction status based on the response
	n.isAuctionOngoing = resp.GetAuctionStillGoing()
	return n.isAuctionOngoing
}


func (n *Node) GetTimesBidded() {
	resp, err := n.client.GetTimesBidded(context.Background(), &proto.TimesBiddedRequest{})
	if err != nil {
		log.Fatalf("failed to get Times bidded: %v", err)
	}
	n.TimesBidded = resp.GetTimesBidded()
}

func (n *Node) GetResult() {
	resp, err := n.client.GetResult(context.Background(), &proto.ResultRequest{})
	if err != nil {
		log.Fatalf("failed to get Tresponse bidded: %v", err)
	}
	fmt.Print("The winner bid is :")
	fmt.Print(resp.GetWinnerBid())
}

func (n *Node) ConnectToPrimary(primaryAddress string) error {
	// Retry logic for connecting to the primary server
	var err error
	n.conn, err = grpc.Dial(primaryAddress, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to connect to primary server: %v", err)
	}

	n.client = proto.NewBiddingServiceClient(n.conn) // Initialize the client
	if n.client == nil {
		return fmt.Errorf("failed to initialize client after connecting to server")
	}

	fmt.Printf("Successfully connected to the server at %s\n", primaryAddress)
	return nil
}


func (n *Node) ReconnectToPrimaryOnFailure(primaryAddress string) {
	// Retry indefinitely to reconnect to the new primary server
	for {
		err := n.ConnectToPrimary(primaryAddress)
		if err != nil {
			log.Printf("Failed to connect to primary server at %s, retrying...\n", primaryAddress)
			time.Sleep(2 * time.Second) // Wait before retrying
			continue
		}
		// Once connected, break out of the loop and proceed
		break
	}
}

func (n *Node) PingPrimary() bool {
	// Ensure the client is not nil before calling Ping
	if n.client == nil {
		log.Printf("client is nil, cannot ping primary server")
		return false
	}

	_, err := n.client.Ping(context.Background(), &proto.PingRequest{})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Unavailable {
			// If the primary server is unavailable, return false
			fmt.Println("Primary server is unavailable.")
			return false
		}
		log.Printf("Ping error: %v", err)
		return false
	}
	// If no error, the server is alive
	return true
}


func (node *Node) ping(replicaServerAddress string, err error) {
	if !node.PingPrimary() {
		// If ping fails, try to reconnect to the replica server
		fmt.Println("Primary server is unavailable, attempting to connect to the replica server...")
		time.Sleep(12 * time.Second)
		node.ReconnectToPrimaryOnFailure(replicaServerAddress)
		if err != nil {
			fmt.Println("Failed to connect to the replica server.")
		}
	}
}

func main() {
	clientID := os.Args[1]
	primaryServerAddress := "localhost:50051" // This should be the address of the current primary server
	replicaServerAddress := "localhost:50052"  // Address of the replica server

	node := &Node{
		NodeID:      clientID,
		Balance:     randomIntBetween(1000, 10000),
		TimesBidded: 0,
	}

	// Initially, try to connect to the primary server
	err := node.ConnectToPrimary(primaryServerAddress)
	if err != nil {
		log.Printf("Failed to connect to primary server: %v", err)
		// If primary connection fails, attempt to connect to the replica server
		err = node.ConnectToPrimary(replicaServerAddress)
		if err != nil {
			log.Fatalf("Failed to connect to both primary and replica servers: %v", err)
		}
	}
	defer node.conn.Close()

	fmt.Printf("BALANCE: %v\n", node.Balance)

	// Start the bidding loop
	for {
		node.ping(replicaServerAddress, err)
		if (node.GetIsAuctionOngoing()) {
			node.ping(replicaServerAddress, err)
			node.GetTimesBidded()
			node.ping(replicaServerAddress, err)
			node.GetHighestBid()
			//fmt.Print("This is the highestBid ")
			fmt.Print(node.CurrentHighestBid)
			node.ping(replicaServerAddress, err)


			if node.CurrentBid == node.CurrentHighestBid && node.TimesBidded > 0 {
				time.Sleep(4 * time.Second)

				continue
			}
			node.ping(replicaServerAddress, err)


			// Place a bid if the current bid is less than or equal to the balance
			node.ping(replicaServerAddress, err)

			if node.CurrentHighestBid < node.Balance {
				node.PlaceBid()
				fmt.Println()
				fmt.Printf("This is the current bid : %v\n", node.CurrentBid)
			}
			node.ping(replicaServerAddress, err)


			// Simulate connection failure scenario

			// Wait for a few seconds before retrying
			time.Sleep(4 * time.Second)
		} else {
			break
		}
	}
	node.GetResult()

}
