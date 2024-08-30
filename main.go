package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
	log *slog.Logger
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	peer, ok := grpccredentials.PeerIDFromContext(ctx)
	if !ok {
		s.log.Info("Failed to get peer ID")
	} else {
		s.log.Info("Received request", "from", peer.String())
	}
	return &pb.HelloReply{Message: "Pong"}, nil
}

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("encountered fatal error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	socketPath := os.Getenv("SPIFFE_ENDPOINT_SOCKET")
	if socketPath == "" {
		return fmt.Errorf("SPIFFE_ENDPOINT_SOCKET is not defined")
	}
	listen := os.Getenv("LISTEN")
	if listen == "" {
		return fmt.Errorf("LISTEN is not defined")
	}
	target := os.Getenv("TARGET")
	if target == "" {
		return fmt.Errorf("TARGET is not defined")
	}

	source, err := workloadapi.NewX509Source(ctx, workloadapi.WithClientOptions(workloadapi.WithAddr(socketPath)))
	if err != nil {
		return fmt.Errorf("unable to create X509Source: %w", err)
	}
	defer source.Close()

	svid, err := source.GetX509SVID()
	if err != nil {
		return fmt.Errorf("unable to get X509SVID: %w", err)
	}

	log := slog.With("me", svid.ID.String())

	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		// Create a server with credentials that do mTLS and verify that the presented certificate has SPIFFE ID `spiffe://example.org/client`
		s := grpc.NewServer(grpc.Creds(
			grpccredentials.MTLSServerCredentials(source, source, tlsconfig.AuthorizeAny()),
		))

		lis, err := net.Listen("tcp", listen)
		if err != nil {
			return fmt.Errorf("error creating listener: %w", err)
		}

		pb.RegisterGreeterServer(s, &server{
			log: log.With("component", "server"),
		})
		stop := context.AfterFunc(egCtx, s.Stop)
		defer stop()
		if err := s.Serve(lis); err != nil {
			return fmt.Errorf("failed to serve: %w", err)
		}
		return nil
	})
	eg.Go(func() error {
		log := log.With("component", "client")
		s, err := grpc.NewClient(target, grpc.WithTransportCredentials(
			grpccredentials.MTLSClientCredentials(source, source, tlsconfig.AuthorizeAny()),
		))
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer s.Close()

		greeterClient := pb.NewGreeterClient(s)

		for {
			_, err := greeterClient.SayHello(egCtx, &pb.HelloRequest{Name: "Ping"})
			if err != nil {
				log.Error("Received error", "error", err)
			} else {
				log.Info("Sent message")
			}

			select {
			case <-egCtx.Done():
				return nil
			case <-time.After(5 * time.Second):

			}
		}
	})

	return eg.Wait()
}
