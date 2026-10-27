package remoteml

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func (c *Client) Preparation(parent context.Context) (aimodel.Preparation, error) {
	unavailable := aimodel.Preparation{State: "unavailable"}
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	case <-ctx.Done():
		return unavailable, nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(c.endpoint, "/predict")+"/api/v1/preparation", nil)
	if err != nil {
		return unavailable, nil
	}
	response, err := c.http.Do(request)
	if err != nil {
		return unavailable, nil
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return unavailable, nil
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 4097))
	if err != nil || len(data) > 4096 {
		return unavailable, nil
	}
	var value aimodel.Preparation
	if json.Unmarshal(data, &value) != nil {
		return unavailable, nil
	}
	switch value.State {
	case "preparing", "downloading", "loading", "ready", "failed":
	default:
		return unavailable, nil
	}
	if value.CompletedFiles < 0 || value.TotalFiles < value.CompletedFiles || value.TotalFiles > 512 ||
		value.CompletedBytes < 0 || value.TotalBytes < value.CompletedBytes || value.TotalBytes > 8<<30 {
		return unavailable, nil
	}
	return value, nil
}
