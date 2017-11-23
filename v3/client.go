/*
Package ccu provides access to the Akamai Cache Control Utility API V3.

This API is only available to users with access to Fast Purge. For traditional
purge requests, use the v2 package instead.

See: https://developer.akamai.com/api/purge/ccu/overview.html
*/
package ccu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/client-v1"
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/edgegrid"
)

// DefaultClient is the default Client.
var DefaultClient = &Client{
	HTTPClient: http.DefaultClient,
}

// A Client is an CCU v3 API client.
type Client struct {
	// HTTPClient is the http.Client used for all HTTP requests.
	HTTPClient *http.Client

	// Config is the edgegrid.Config used to configure the authentication headers
	// and API endpoint for all API requests.
	Config *edgegrid.Config
}

// A Response is the base object containing fields common to all API responses.
//
// If an API error occurs, Response satisfies the error interface and can be
// used to extract error details.
type Response struct {
	// SupportID is an identifier to provide Akamai Technical Support if issues
	// arise.
	SupportID string `json:"supportId"`

	// StatusCode is the HTTP code that indicates the status of the request.
	StatusCode int `json:"httpStatus"`

	// Title describes the response type.
	Title string `json:"title"`

	// Detail contains detailed information about the HTTP status code returned
	// with the response.
	Detail string `json:"detail"`

	// DescribeBy is the URL for the API’s machine readable documentation
	DescribedBy string `json:"describedBy"`
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

// config returns the Config associated with a client, or initializes a default
// configuration if none exists.
func (c *Client) config() (*edgegrid.Config, error) {
	if c.Config != nil {
		return c.Config, nil
	}
	cfg, err := edgegrid.Init("", "") // ~.edgerc, default
	if err != nil {
		return nil, err
	}
	c.Config = &cfg
	return c.Config, nil
}

// newHTTPRequest returns a http.Request with the given parameters. If v is not
// nil, it is encoded as JSON in the request body.
//
// Authentication headers are appended according to the client.Config.
func (c *Client) newHTTPRequest(method, path string, v interface{}, ctx context.Context) (*http.Request, error) {
	cfg, err := c.config()
	if err != nil {
		return nil, fmt.Errorf("error reading edgegrid configuration: %v", err)
	}
	hreq, err := client.NewJSONRequest(*cfg, method, path, v)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request: %v", err)
	}
	hreq = edgegrid.AddRequestHeader(*cfg, hreq)
	if ctx != nil {
		hreq = hreq.WithContext(ctx)
	}
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
	if v != nil {
		dec := json.NewDecoder(resp.Body)
		if err = dec.Decode(v); err != nil {
			return resp, fmt.Errorf("error decoding JSON response: %v", err)
		}
	}
	return resp, nil
}

// A PurgeRequest represents an API Purge request to be sent by a client.
type PurgeRequest struct {
	// Type must be one of "url" (default), "cpcode" or "tag".
	Type string `json:"-"`

	// Action must be one of "invalidate" (default) or "delete"
	Action string `json:"-"`

	// Network must be one of "production" (default) or "staging"
	Network string `json:"-"`

	// Hostname identifies the domain from which the content is purged, assuming
	// Type is "url" and Objects contains a list of URL paths.
	Hostname string `json:"hostname,omitempty"`

	// Objects is a list of URLs, CP Codes or Tags to purge
	Objects []string `json:"objects"`
}

// Path returns the API path for the request.
func (p *PurgeRequest) Path() string {
	return fmt.Sprintf("/ccu/v3/%s/%s/%s", p.Action, p.Type, p.Network)
}

type PurgeResponse struct {
	Response

	EstimatedSeconds int    `json:"estimatedSeconds"`
	PurgeID          string `json:"purgeId"`
}

// Purge allows you to purge edge content.
//
// The given context.Context is used to allow cancellation of long running
// requests.
func (c *Client) Purge(req *PurgeRequest, ctx context.Context) (*PurgeResponse, error) {
	if req.Action == "" {
		req.Action = "invalidate"
	}
	if req.Network == "" {
		req.Network = "production"
	}
	if req.Type == "" {
		req.Type = "url"
	}
	hreq, err := c.newHTTPRequest("POST", req.Path(), req, ctx)
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
	return v, err
}
