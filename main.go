package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/aysegulrana/rpc_lab/pb"

	"google.golang.org/grpc"
)

const (
	host = "localhost"
	port = "8080"
)

type MyGreeterServer struct {
	pb.UnimplementedGreeterServer // composition
}

// Single request, single reply
// We simply get the name and return the message
func (s *MyGreeterServer) MonoHello(c context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {

	name := req.Name                                         // Get the name from the request object (struct is in the greeter.pb.go file)
	return &pb.HelloResponse{Message: "Hello, " + name}, nil // Return Hello + name and no error
}

// Single request, multiple replies
// We get the name from the client and return multiple responses to it
func (s *MyGreeterServer) LotsOfReplies(req *pb.HelloRequest, server pb.Greeter_LotsOfRepliesServer) error {
	name := req.Name // Get the name from the request object (struct is in the greeter.pb.go file)

	for i := 0; i < 10; i++ { // We will send 10 replies back
		server.Send(&pb.HelloResponse{Message: "Hello, " + name + " from iteration " + strconv.Itoa(i)})
	}

	return nil

}

func (s *MyGreeterServer) LotsOfGreetings(server pb.Greeter_LotsOfGreetingsServer) error {
	names := make([]string, 0)
	for {
		req, err := server.Recv()

		if err != nil {
			break
		}

		names = append(names, req.Name)
	}

	msg := "Hello"
	for _, name := range names {
		msg += "," + name
	}

	server.SendAndClose(&pb.HelloResponse{Message: msg})
	return nil
}

func (s *MyGreeterServer) BidiHello(server pb.Greeter_BidiHelloServer) error {

	for {
		req, err := server.Recv()

		if err != nil {
			break
		}

		server.Send(&pb.HelloResponse{Message: "Hello," + req.Name})
	}

	return nil

}

func main() {

	op_mode := os.Args[1]

	if op_mode == "server" {
		fmt.Println("Server Mode...")
		lis, _ := net.Listen("tcp", host+":"+port)

		var opts []grpc.ServerOption

		grpcServer := grpc.NewServer(opts...)
		myServer := MyGreeterServer{}
		pb.RegisterGreeterServer(grpcServer, &myServer)

		grpcServer.Serve(lis)

	} else if op_mode == "client" {
		fmt.Println("Client mode... Please enter your name:")

		var opts []grpc.DialOption
		opts = append(opts, grpc.WithInsecure())
		conn, err := grpc.Dial(host+":"+port, opts...)

		if err != nil {
			fmt.Println(err)
		}

		defer conn.Close()

		client := pb.NewGreeterClient(conn)

		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			name := scanner.Text()
			fmt.Println("Sending:", name)
			req := pb.HelloRequest{Name: name}
			reply, err := client.MonoHello(context.Background(), &req)

			if err == nil {
				fmt.Println("Reply:", reply)
			}
		}
	} else if op_mode == "client_stream" { // WIP
		fmt.Println("Streaming client mode...")

		var opts []grpc.DialOption
		opts = append(opts, grpc.WithInsecure())
		conn, err := grpc.Dial(host+":"+port, opts...)

		if err != nil {
			fmt.Println(err)
		}

		defer conn.Close()

		client := pb.NewGreeterClient(conn)
		stream, _ := client.BidiHello(context.Background())

		var wg sync.WaitGroup
		wg.Add(2) // Because we have 2 goroutines

		go func() {
			defer wg.Done() // Decrement the counter when the goroutine completes

			for i := 1; i <= 10; i++ {
				s := strconv.Itoa(i)
				req := pb.HelloRequest{Name: s}
				stream.Send(&req)
			}
			stream.CloseSend()
		}()

		go func() {
			defer wg.Done()

			for i := 1; i <= 10; i++ {
				ret, _ := stream.Recv()
				fmt.Println(ret)
			}
		}()

		wg.Wait()

	} else {
		fmt.Println("Unsupported Operating Mode")
	}

}
