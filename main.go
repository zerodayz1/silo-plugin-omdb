package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"sync"

	"google.golang.org/protobuf/types/known/structpb"

	pluginv1 "github.com/Silo-Server/silo-plugin-sdk/pkg/pluginproto/silo/plugin/v1"
	publicmanifest "github.com/Silo-Server/silo-plugin-sdk/pkg/pluginsdk/manifest"
	"github.com/Silo-Server/silo-plugin-sdk/pkg/pluginsdk/runtime"
	"github.com/zerodayz1/silo-plugin-omdb/omdb"
)

// version is set at build time via -ldflags "-X main.version=...".
var version string

type runtimeServer struct {
	pluginv1.UnimplementedRuntimeServer
	manifest *pluginv1.PluginManifest

	mu     sync.RWMutex
	client *omdb.Client // nil until Configure provides an API key
}

type metadataServer struct {
	pluginv1.UnimplementedMetadataProviderServer
	runtime *runtimeServer
}

//go:embed manifest.json
var manifestJSON []byte

func (s *runtimeServer) GetManifest(_ context.Context, _ *pluginv1.GetManifestRequest) (*pluginv1.GetManifestResponse, error) {
	return &pluginv1.GetManifestResponse{Manifest: s.manifest}, nil
}

func (s *runtimeServer) Configure(_ context.Context, req *pluginv1.ConfigureRequest) (*pluginv1.ConfigureResponse, error) {
	for _, entry := range req.GetConfig() {
		if entry.GetKey() != "omdb_credentials" {
			continue
		}
		fields := entry.GetValue().GetFields()
		apiKey := ""
		if v, ok := fields["api_key"]; ok {
			apiKey = v.GetStringValue()
		}
		if apiKey != "" {
			s.mu.Lock()
			s.client = omdb.NewClient(apiKey)
			s.mu.Unlock()
		}
	}
	return &pluginv1.ConfigureResponse{}, nil
}

// Search returns an empty result set. OMDb requires a known IMDb ID and does
// not support title search on the free tier.
func (s *metadataServer) Search(_ context.Context, _ *pluginv1.SearchMetadataRequest) (*pluginv1.SearchMetadataResponse, error) {
	return &pluginv1.SearchMetadataResponse{}, nil
}

func (s *metadataServer) GetSeasons(_ context.Context, _ *pluginv1.GetSeasonsRequest) (*pluginv1.GetSeasonsResponse, error) {
	return &pluginv1.GetSeasonsResponse{}, nil
}

func (s *metadataServer) GetEpisodes(_ context.Context, _ *pluginv1.GetEpisodesRequest) (*pluginv1.GetEpisodesResponse, error) {
	return &pluginv1.GetEpisodesResponse{}, nil
}

// GetMetadata looks up IMDb and Rotten Tomatoes ratings for the item's IMDb ID
// and returns only the ratings — all other fields are left empty so they do not
// overwrite data already contributed by a richer provider such as TMDB.
func (s *metadataServer) GetMetadata(ctx context.Context, req *pluginv1.GetMetadataRequest) (*pluginv1.GetMetadataResponse, error) {
	s.runtime.mu.RLock()
	client := s.runtime.client
	s.runtime.mu.RUnlock()

	if client == nil {
		return &pluginv1.GetMetadataResponse{}, nil
	}

	// The capability ID is "imdb", so ProviderId carries the IMDb ID directly.
	imdbID := req.GetProviderId()
	if imdbID == "" {
		imdbID = stringMapFromStruct(req.GetProviderIds())["imdb"]
	}
	if imdbID == "" {
		return &pluginv1.GetMetadataResponse{}, nil
	}

	ratings, err := client.GetRatings(ctx, imdbID)
	if err != nil || ratings == nil {
		return &pluginv1.GetMetadataResponse{}, nil
	}

	providerIDs, _ := stringStruct(map[string]string{"imdb": imdbID})
	return &pluginv1.GetMetadataResponse{
		Item: &pluginv1.MetadataItem{
			ProviderId:  imdbID,
			ItemType:    req.GetItemType(),
			ProviderIds: providerIDs,
			Ratings:     ratingsStruct(ratings),
		},
	}, nil
}

func main() {
	manifest, err := loadManifest()
	if err != nil {
		panic(err)
	}

	rs := &runtimeServer{manifest: manifest}

	runtime.Serve(runtime.ServeConfig{
		Servers: runtime.CapabilityServers{
			Runtime:          rs,
			MetadataProvider: &metadataServer{runtime: rs},
		},
	})
}

func loadManifest() (*pluginv1.PluginManifest, error) {
	manifest, err := publicmanifest.Load(manifestJSON)
	if err != nil {
		return nil, fmt.Errorf("load embedded manifest: %w", err)
	}
	if version != "" {
		manifest.Version = version
	}
	executablePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}
	binaryData, err := os.ReadFile(executablePath)
	if err != nil {
		return nil, fmt.Errorf("read executable %q: %w", executablePath, err)
	}
	checksum := sha256.Sum256(binaryData)
	manifest.Checksum = hex.EncodeToString(checksum[:])
	return manifest, nil
}

func stringMapFromStruct(value *structpb.Struct) map[string]string {
	result := make(map[string]string)
	if value == nil {
		return result
	}
	for key, raw := range value.AsMap() {
		if text, ok := raw.(string); ok && text != "" {
			result[key] = text
		}
	}
	return result
}

func stringStruct(value map[string]string) (*structpb.Struct, error) {
	converted := make(map[string]any, len(value))
	for k, v := range value {
		if v != "" {
			converted[k] = v
		}
	}
	if len(converted) == 0 {
		return nil, nil
	}
	return structpb.NewStruct(converted)
}

func ratingsStruct(r *omdb.Ratings) *structpb.Struct {
	values := make(map[string]any)
	if r.IMDB != 0 {
		values["imdb"] = r.IMDB
	}
	if r.RTCritic != 0 {
		values["rt_critic"] = float64(r.RTCritic)
	}
	if len(values) == 0 {
		return nil
	}
	s, err := structpb.NewStruct(values)
	if err != nil {
		return nil
	}
	return s
}
