// Package ccu provides access to the Akamai Cache Control Utility API V2.
// See: https://developer.akamai.com/api/purge/ccu/overview.html
package ccu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	apiURL = "https://api.ccu.akamai.com"
)

var DefaultClient = &Client{
	HTTPClient: http.DefaultClient,
	Username:   os.Getenv("AKAMAI_CCU_USERNAME"),
	Password:   os.Getenv("AKAMAI_CCU_PASSWORD"),
}

type Client struct {
	HTTPClient *http.Client
	Username   string
	Password   string
	Logger     *log.Logger
}

type Response struct {
	SupportID   string `json:"supportId"`
	StatusCode  int    `json:"httpStatus"`
	Title       string `json:"title"`
	Detail      string `json:"detail"`
	DescribedBy string `json:"describedBy"`
	RawResponse string `json:"-"`
}

func (e *Response) Error() string {
	if e.Title == "" {
		return "unknown"
	}
	if e.Detail == "" {
		return e.Title
	}
	return fmt.Sprintf("%s: %s", e.Title, e.Detail)
}

// assertError returns an error if the HTTP Status Code for the Response is not
// in the 200 range.
func (r *Response) assertError() error {
	if r.StatusCode < 200 || r.StatusCode > 299 {
		return r
	}
	return nil
}

// newHTTPRequest returns a http.Request with the given parameters. If v is not
// nil, it is encoded as JSON in the request body.
func (c *Client) newHTTPRequest(method, url string, v interface{}, ctx context.Context) (*http.Request, error) {
	var body io.Reader
	if v != nil {
		b := &bytes.Buffer{}
		enc := json.NewEncoder(b)
		if err := enc.Encode(v); err != nil {
			return nil, fmt.Errorf("error encoding request as JSON: %v", err)
		}
		body = b
	}
	hreq, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %v", err)
	}
	if body != nil {
		hreq.Header.Set("Content-Type", "application/json")
	}
	hreq.SetBasicAuth(c.Username, c.Password)
	hreq = hreq.WithContext(ctx)
	return hreq, nil
}

// do sends an HTTP request and returns an HTTP response. If v is not nil, the
// body of the response if decoded as JSON into v.
func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending HTTP request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized")
	}
	if v != nil {
		dec := json.NewDecoder(resp.Body)
		if err = dec.Decode(v); err != nil {
			// TODO: include response body somewhere in error
			return nil, fmt.Errorf("error decoding JSON response: %v", err)
		}
	}
	return resp, nil
}

type QueueLengthResponse struct {
	Response

	QueueLength int `json:"queueLength"`
}

func (c *Client) GetQueueLength(ctx context.Context) (*QueueLengthResponse, error) {
	url := fmt.Sprintf("%s/ccu/v2/queues/default", apiURL)
	hreq, err := c.newHTTPRequest("GET", url, nil, ctx)
	if err != nil {
		return nil, err
	}
	v := &QueueLengthResponse{}
	_, err = c.do(hreq, v)
	if err != nil {
		return nil, err
	}
	err = v.Response.assertError()
	if err != nil {
		return nil, err
	}
	return v, nil
}

type PurgeRequest struct {
	Queue   string   `json:"-"`
	Type    string   `json:"type,omitempty"`
	Action  string   `json:"action,omitempty"`
	Domain  string   `json:"domain,omitempty"`
	Objects []string `json:"objects"`
}

func NewPurgeRequest(objects ...string) *PurgeRequest {
	return &PurgeRequest{
		Type:    "arl",
		Action:  "remove",
		Domain:  "production",
		Objects: objects,
	}
}

type PurgeResponse struct {
	Response

	EstimatedSeconds int       `json:"estimatedSeconds"`
	PurgeID          string    `json:"purgeId"`
	ProgressURI      string    `json:"progressUri"`
	PingAfterSeconds int       `json:"pingAfterSeconds"`
	Time             time.Time `json:"-"`
}

func (p *PurgeResponse) ETA() time.Time {
	if p.Time.IsZero() {
		return time.Time{}
	}
	return p.Time.Add(time.Second * time.Duration(p.EstimatedSeconds))
}

func (c *Client) Purge(req *PurgeRequest, ctx context.Context) (*PurgeResponse, error) {
	q := req.Queue
	if q == "" {
		q = "default"
	}
	url := fmt.Sprintf("%s/ccu/v2/queues/%s", apiURL, q)
	hreq, err := c.newHTTPRequest("POST", url, req, ctx)
	if err != nil {
		return nil, err
	}
	v := &PurgeResponse{}
	_, err = c.do(hreq, v)
	if err != nil {
		return nil, err
	}
	err = v.Response.assertError()
	if err != nil {
		return nil, err
	}
	v.Time = time.Now()
	return v, err
}

type PurgeStatusResponse struct {
	Response

	OriginalEstimatedSeconds int    `json:"originalEstimatedSeconds"`
	OriginalQueueLength      int    `json:"originalQueueLength"`
	PurgeID                  string `json:"purgeId"`
	CompletionTime           string `json:"completionTime"`
	SubmittedBy              string `json:"submittedBy"`
	PurgeStatus              string `json:"purgeStatus"`
	SubmissionTime           string `json:"submissionTime"`
}

func (p *PurgeStatusResponse) IsDone() bool {
	return p.PurgeStatus == "Done"
}

func (c *Client) GetPurgeStatus(purgeID string, ctx context.Context) (*PurgeStatusResponse, error) {
	url := fmt.Sprintf("%s/ccu/v2/purges/%s", apiURL, purgeID)
	req, err := c.newHTTPRequest("GET", url, nil, ctx)
	if err != nil {
		return nil, err
	}
	v := &PurgeStatusResponse{}
	_, err = c.do(req, v)
	if err != nil {
		return nil, err
	}
	err = v.Response.assertError()
	if err != nil {
		return nil, err
	}
	return v, nil
}
