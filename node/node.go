package main

import (
	"context"
	"fmt"
	proto "handin5/grpc"
	"log"
	"math/rand"
	"time"

	"google.golang.org/grpc"
)

type Node struct {
	NodeID string;
	CurrentHighestBid int32;
	CurrentBid int32;
	Balance int32;
	ParticipatingInAuction bool;
	TimesBidded int32;

}

func randomIntBetween(min int32, max int32) int32 {
	if (min >= max) {
		panic("Invalid range: min must be less than max")
	}
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
		amount = randomIntBetween(n.CurrentHighestBid, n.CurrentHighestBid + 100)
		//amount = randomIntBetween(n.CurrentHighestBid, n.Balance)
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
        //fmt.Printf("Bid placed successfully: %s\n", resp.Comment)

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

func (n *Node) GetTimesBidded(client proto.BiddingServiceClient) {
	resp, err := client.GetTimesBidded(context.Background(), &proto.TimesBiddedRequest{})
	if err != nil {
        log.Fatalf("failed to get Times bidded: %v", err)
    }
	n.TimesBidded = resp.GetTimesBidded()
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
		TimesBidded: 0,
		Balance: randomIntBetween(1000,10000),
	}
	fmt.Printf("BALANCE: %v\n", node.Balance)

	for {
	node.GetTimesBidded(client)
	node.GetHighestBid(client)
	if(node.CurrentBid == node.CurrentHighestBid && node.TimesBidded > 0) {
		continue
	}
	if(node.CurrentHighestBid <= node.Balance) {
		node.PlaceBid(client)
		fmt.Printf("This is the current bid : ")
		fmt.Println(node.CurrentBid)
	}
	//fmt.Print("NodeID  " + node.NodeID)
	time.Sleep(4 * time.Second)
	}
}