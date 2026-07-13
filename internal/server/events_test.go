package server

import (
	"encoding/json"
	"testing"

	"github.com/TrustEdgeOrg/TrustTwin/internal/constants"
	"github.com/TrustEdgeOrg/TrustTwin/internal/models"
)

func TestDecodeEventsSingle(t *testing.T) {
	body := []byte(`{"type":"client_details","payload":{"hostname":"x"}}`)
	events, err := decodeEvents(body)
	if err != nil || len(events) != 1 || events[0].Type != constants.TypeClientDetails {
		t.Fatalf("events=%+v err=%v", events, err)
	}
}

func TestDecodeEventsBatch(t *testing.T) {
	body := []byte(`{"events":[{"type":"process_start","payload":{"pid":1}},{"type":"process_exit","payload":{"pid":1}}]}`)
	events, err := decodeEvents(body)
	if err != nil || len(events) != 2 {
		t.Fatalf("events=%+v err=%v", events, err)
	}
}

func TestDecodeEventsBatchModel(t *testing.T) {
	raw, _ := json.Marshal(models.EventBatch{
		Events: []models.Event{{Type: constants.TypeProcessStart}},
	})
	events, err := decodeEvents(raw)
	if err != nil || len(events) != 1 {
		t.Fatalf("events=%+v err=%v", events, err)
	}
}
