package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net"

	pb "github.com/pieterlouw/grpc-beyond/proto"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var listenPort = flag.String("l", ":7100", "Specify the port that the server will listen on")

/* usersService implements GoReleaseServiceServer as defined in the generated code:
// Server API for GoReleaseService service
type GoReleaseServiceServer interface {
	GetReleaseInfo(context.Context, *GetReleaseInfoRequest) (*ReleaseInfo, error)
	ListReleases(context.Context, *ListReleasesRequest) (*ListReleasesResponse, error)
}
*/

type releaseInfo struct {
	ReleaseDate     string `json:"releaseDate"`
	ReleaseNotesURL string `json:"releaseNotesURL"`
}

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
		ReleaseNotesURL: ri.ReleaseNotesURL,
	}, nil
}

func (g *goReleaseService) ListReleases(ctx context.Context, r *pb.ListReleasesRequest) (*pb.ListReleasesResponse, error) {
	var releases []*pb.ReleaseInfo

	for k, v := range g.releases {
		ri := &pb.ReleaseInfo{
			Version:         k,
			ReleaseDate:     v.ReleaseDate,
			ReleaseNotesURL: v.ReleaseNotesURL,
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

	lis, err := net.Listen("tcp", *listenPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Println("Listening on ", *listenPort)
	server := grpc.NewServer()

	pb.RegisterGoReleaseServiceServer(server, svc)

	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
