package ccu

import (
	"testing"
)

func TestPurge(t *testing.T) {
	t.Parallel()
	req := &PurgeRequest{
		Type:    "cpcode",
		Objects: []string{"123456"},
	}
	_, err := DefaultClient.Purge(req, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
}
