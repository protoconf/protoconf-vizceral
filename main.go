package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/avast/retry-go"
	"github.com/gorilla/handlers"
	"github.com/protoconf/protoconf-vizceral/pkg/pb"
	"github.com/protoconf/protoconf-vizceral/pkg/static"
	"golang.org/x/sync/errgroup"
	grpc "google.golang.org/grpc"
	jsonpb "google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	PROTOCONF_AGENT_ADDR  = flag.String("protoconf_addr", "localhost:4300", "protoconf agent addr")
	PROTOCONF_CONFIG_PATH = flag.String("config_path", "spinnaker", "protoconf agent addr")
)

type srv struct {
	node *pb.Node
}

func (s *srv) update(agentAddr, configPath string) {
	retry.Do(
		func() error {
			conn, err := grpc.Dial(agentAddr, grpc.WithInsecure())
			if err != nil {
				log.Println(err)
				return err
			}
			stub := pb.NewProtoconfServiceClient(conn)
			stream, err := stub.SubscribeForConfig(context.Background(), &pb.ConfigSubscriptionRequest{Path: configPath})
			if err != nil {
				log.Println(err)
				return err
			}
			for {
				update, err := stream.Recv()
				if err == io.EOF {
					log.Printf("Connection closed while streaming config path=%s", configPath)
					return err
				}
				if err != nil {
					log.Printf("Error while streaming config path=%s err=%s", configPath, err)
					return err
				}
				if err = anypb.UnmarshalTo(update.GetValue(), s.node, proto.UnmarshalOptions{}); err != nil {
					log.Printf("Error unmarshaling config path=%s value=%s err=%s", configPath, update.Value, err)
					return err
				}
				walkNodes(s.node, func(n *pb.Node) error {
					n.Updated = int32(time.Now().Unix())

					return nil
				})
			}
		},
	)
}

func walkNodes(n *pb.Node, f func(*pb.Node) error) error {
	g, _ := errgroup.WithContext(context.Background())
	for _, s := range n.Nodes {
		g.Go(func() error { return f(s) })
		g.Go(func() error { return walkNodes(s, f) })
	}
	return g.Wait()
}

func (s *srv) serve(w http.ResponseWriter, r *http.Request) {
	configBytes, err := jsonpb.Marshal(s.node)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(`{"error": "failed to marshal"}`))
		return
	}
	w.WriteHeader(200)
	w.Write([]byte(configBytes))
}

func root(w http.ResponseWriter, r *http.Request) {
	file := strings.TrimPrefix(r.URL.Path, "/")
	data, err := static.Asset(file)
	if err != nil {
		serveIndex(w, r)
		return
	}
	info, err := static.AssetInfo(file)
	if err != nil {
		serveIndex(w, r)
		return
	}
	http.ServeContent(w, r, r.URL.Path, info.ModTime(), bytes.NewReader(data))
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	data, err := static.Asset("index.html")
	if err != nil {
		log.Println(err)
		http.NotFound(w, r)
		return
	}
	info, err := static.AssetInfo("index.html")
	if err != nil {
		log.Println(err)
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, r.URL.Path, info.ModTime(), bytes.NewReader(data))
}

func main() {
	flag.Parse()
	log.Println("starting...", *PROTOCONF_AGENT_ADDR)

	s := &srv{node: &pb.Node{}}
	go s.update(*PROTOCONF_AGENT_ADDR, *PROTOCONF_CONFIG_PATH)
	mux := http.NewServeMux()
	// mux.Handle("/", http.FileServer(AssetFile()))
	mux.HandleFunc("/map.json", s.serve)
	mux.HandleFunc("/", root)
	http.ListenAndServe(":8080", handlers.LoggingHandler(os.Stdout, mux))
}
