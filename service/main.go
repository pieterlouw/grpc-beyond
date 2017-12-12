package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"time"

	pb "github.com/pieterlouw/grpc-beyond/proto"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var listenPort = flag.String("l", ":7100", "Specify the port that the server will listen on")

type releaseInfo struct {
	ReleaseDate     string `json:"release_date"`
	ReleaseNotesURL string `json:"release_notes_url"`
}

/* goReleaseService implements GoReleaseServiceServer as defined in the generated code:
// Server API for GoReleaseService service
type GoReleaseServiceServer interface {
	GetReleaseInfo(context.Context, *GetReleaseInfoRequest) (*ReleaseInfo, error)
	ListReleases(context.Context, *ListReleasesRequest) (*ListReleasesResponse, error)
}
*/
type goReleaseService struct {
	releases map[string]releaseInfo
}

func (g *goReleaseService) GetReleaseInfo(ctx context.Context, r *pb.GetReleaseInfoRequest) (*pb.ReleaseInfo, error) {
	ri, ok := g.releases[r.GetVersion()]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "release verions %s not found", r.GetVersion())
	}

	// success
	return &pb.ReleaseInfo{
		Version:         r.GetVersion(),
		ReleaseDate:     ri.ReleaseDate,
		ReleaseNotesUrl: ri.ReleaseNotesURL,
	}, nil
}

func (g *goReleaseService) ListReleases(ctx context.Context, r *pb.ListReleasesRequest) (*pb.ListReleasesResponse, error) {
	var releases []*pb.ReleaseInfo

	for k, v := range g.releases {
		ri := &pb.ReleaseInfo{
			Version:         k,
			ReleaseDate:     v.ReleaseDate,
			ReleaseNotesUrl: v.ReleaseNotesURL,
		}

		releases = append(releases, ri)
	}

	return &pb.ListReleasesResponse{
		Releases: releases,
	}, nil
}

func main() {
	flag.Parse()
	svc := &goReleaseService{
		releases: make(map[string]releaseInfo),
	}

	jsonData, err := ioutil.ReadFile("../data/releases.json") // For read access.
	if err != nil {
		log.Fatalf("failed to read data file: %v", err)
	}

	//read releases from JSON data file
	err = json.Unmarshal(jsonData, &svc.releases)
	if err != nil {
		log.Fatalf("failed to marshal release data: %v", err)
	}

	// prepare TLS Config
	tlsCert := "../certs/demo.crt"
	tlsKey := "../certs/demo.key"
	cert, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
	if err != nil {
		log.Fatal(err)
	}

	// create gRPC TLS credentials
	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
	})

	// create new gRPC server with Transport Credentials
	server := grpc.NewServer(
		grpc.Creds(creds),
		grpc.UnaryInterceptor(unaryInterceptor),
	)

	lis, err := net.Listen("tcp", *listenPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Println("Listening on ", *listenPort)

	pb.RegisterGoReleaseServiceServer(server, svc)

	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// general unary interceptor function
func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	if info.FullMethod != "/proto.GoReleaseService/ListReleases" { //skip auth when ListReleases requested
		if err := authorize(ctx); err != nil {
			return nil, err
		}
	}

	h, err := handler(ctx, req)

	log.Printf("request - Method:%s\tDuration:%s\tError:%v\n", info.FullMethod, time.Since(start), err) //logging

	return h, err
}

// authorize function that is used by the interceptor functions.
func authorize(ctx context.Context) error {
	// warning: this is only for illustration purposes - don't implement authorization that is hardcoded!
	var authList = map[string]bool{
		base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", "scaramoucheX2", "Can-you-do-the-fandango?"))): true,
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "retrieving metadata failed")
	}

	elem, ok := md["authorization"]
	if !ok {
		return status.Errorf(codes.InvalidArgument, "no auth details supplied")
	}

	authorization := elem[0][len("Basic "):] //extract base64 basic auth value (similar to HTTP Basic Auth)
	valid, ok := authList[authorization]
	if !ok {
		return status.Errorf(codes.NotFound, "auth not found")
	}

	if !valid {
		return status.Errorf(codes.Unauthenticated, "auth failed")
	}

	return nil
}
