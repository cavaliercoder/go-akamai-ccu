package ccu

import (
	"context"
	"os"
	"testing"
)

func TestEnvironment(t *testing.T) {
	vars := []string{"AKAMAI_CCU_USERNAME", "AKAMAI_CCU_PASSWORD"}
	missing := []string{}
	for _, v := range vars {
		if os.Getenv(v) == "" {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("The following environment variables are not set: %v", missing)
	}
}
func TestGetQueueLength(t *testing.T) {
	t.Parallel()
	_, err := DefaultClient.GetQueueLength(context.Background())
	if err != nil {
		t.Errorf("%v", err)
	}
}
