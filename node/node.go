package node

import (
	"context"
	"fmt"
	proto "handin5/grpc"
	"log"
	"rand"
	"time"

	"google.golang.org/grpc"
)

type Node struct {
	CurrentHighestBid int32;
	CurrentBid int32;
	Balance int32;
	ParticipatingInAuction bool;
	TimesBidded int32;

}

func randomIntBetween(min int32, max int32) int32 {
	rand.Seed(time.Now().UnixNano())
	var randomNumber int32
	randomNumber = min + rand.Int31n(max-min+1)
	return randomNumber
}

func (n *Node) PlaceBid(client proto.BiddingServiceClient) {
	var amount int32
	
	if(n.TimesBidded == 0) {
		//Start bid amount will be between 0 and half of the nodes balance
		//Makes sure that the node doesnt bid all its money at first
		amount = randomIntBetween(0, n.Balance/2)
	} else {
		//Else bid amount will be between currentHighestBid and balance
		//amount := randomIntBetween(currentBid, balance)
		amount = randomIntBetween(n.CurrentHighestBid, n.Balance)
	}
	resp, err := client.PlaceBid(context.Background(), &proto.BidRequest{Amount: amount})
	if err != nil {
		log.Fatalf("failed to place bid")
	}
	// Switch case for the ack response 
	switch resp.Ack {
    case proto.AckStatus_SUCCESS:
		n.TimesBidded++
		n.CurrentBid = amount
        fmt.Printf("Bid placed successfully: %s\n", resp.Comment)

    case proto.AckStatus_FAIL:
        fmt.Printf("Bid failed: %s\n", resp.Comment)
    case proto.AckStatus_EXCEPTION:
        fmt.Printf("Exception occurred: %s\n", resp.Comment)
    default:
        fmt.Printf("Unknown response: %s\n", resp.Comment)
    }
	
}

func (n *Node) GetHighestBid(client proto.BiddingServiceClient) {
	resp, err := client.GetHighestBid(context.Background(), &proto.HighestBidRequest{})
    if err != nil {
        log.Fatalf("failed to get highest bid: %v", err)
    }

	n.CurrentHighestBid = resp.GetHighesBid()

}

func main () {

	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := proto.NewBiddingServiceClient(conn)
	fmt.Println("Successfully connected to the server at localhost:50051")

	node := &Node{
		Balance: 1000, // Example balance
	}

	node.GetHighestBid(client)
}