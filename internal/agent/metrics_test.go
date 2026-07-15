package agent

import (
	"testing"
	"time"
)

func TestMetricsUploadAndAge(t *testing.T) {
	m := &Metrics{}
	if _, ok := m.LastUploadAge(time.Now()); ok {
		t.Fatal("expected no last upload")
	}
	m.RecordUploadSuccess()
	age, ok := m.LastUploadAge(time.Now())
	if !ok || age < 0 || age > 2 {
		t.Fatalf("age=%v ok=%v", age, ok)
	}
	m.RecordUploadFail()
	m.RecordAuthRecover()
	m.RecordQueueDropped()
	if m.UploadSuccess.Load() != 1 || m.UploadFail.Load() != 1 || m.AuthRecover.Load() != 1 || m.QueueDropped.Load() != 1 {
		t.Fatalf("counters success=%d fail=%d recover=%d dropped=%d",
			m.UploadSuccess.Load(), m.UploadFail.Load(), m.AuthRecover.Load(), m.QueueDropped.Load())
	}
}
