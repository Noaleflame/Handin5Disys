package node

import (
	"context"
	"fmt"
	proto "handin5/grpc"
	"log"

	"google.golang.org/grpc"
)

type Node struct {
	CurrentHighestBid int32;
	CurrentBid int32;
	Balance int32;
	ParticipatingInAuction bool;
}

func (n *Node) PlaceBid(client proto.BiddingServiceClient) {
	
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