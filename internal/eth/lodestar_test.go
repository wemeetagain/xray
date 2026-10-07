package eth

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ethp2p/xray/internal/decode"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
)

func TestLodestarPayloads(t *testing.T) {
	data, err := os.ReadFile("testdata/lodestar.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Genesis        uint64 `json:"genesis"`
		SecondsPerSlot uint64 `json:"secondsPerSlot"`
		Fixtures       []struct {
			Name   string  `json:"name"`
			Fork   string  `json:"fork"`
			Topic  string  `json:"topic"`
			Slot   *uint64 `json:"slot"`
			Snappy []byte  `json:"snappy"`
		} `json:"fixtures"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, f := range fixture.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			slot, ok, meta := DecodePayloadForFork(f.Topic, f.Snappy, f.Fork == "gloas")
			if f.Slot != nil && (!ok || slot != *f.Slot) {
				t.Fatalf("slot=(%d,%v), want %d", slot, ok, *f.Slot)
			}
			if f.Name == "envelope" && (!meta.HasTimestamp || meta.Timestamp != fixture.Genesis+42*fixture.SecondsPerSlot) {
				t.Fatalf("wrong envelope metadata: %+v", meta)
			}
			if f.Name == "gloas_block" && (meta.HasTxCount || meta.HasCommitments) {
				t.Fatalf("Gloas block fabricated execution payload metadata: %+v", meta)
			}
			if f.Topic == "beacon_block" && (!meta.HasProposer || meta.ProposerIndex != 11) {
				t.Fatalf("wrong block proposer: %+v", meta)
			}
			if f.Name == "gloas_column" || f.Name == "fulu_column" {
				if !meta.HasSidecarIndex || meta.SidecarIndex != 7 {
					t.Fatalf("wrong column index: %+v", meta)
				}
			}
		})
	}
}

func TestGossipSubLargeFragmentedRPC(t *testing.T) {
	topic := "/eth2/11223344/unknown/ssz_snappy"
	frame, err := makeFrame(&pubsubpb.RPC{Publish: []*pubsubpb.Message{{Topic: &topic, Data: make([]byte, 2<<20)}}})
	if err != nil {
		t.Fatal(err)
	}
	decoder := GossipSubDecoder().New()
	total := 0
	emit := func(n int, _ []decode.Tag, _ any) { total += n }
	for offset := 0; offset < len(frame); offset += 65536 {
		end := min(offset+65536, len(frame))
		if err := decoder.ObserveRead(frame[offset:end], emit); err != nil {
			t.Fatal(err)
		}
	}
	if total != len(frame) {
		t.Fatalf("attributed %d bytes, want %d", total, len(frame))
	}
}
