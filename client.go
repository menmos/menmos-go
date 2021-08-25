package menmos

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/menmos/menmos-go/config"
	"github.com/menmos/menmos-go/payload"
	"github.com/pkg/errors"
)

const userAgent = "menmos-go"

// Client provides an API to interact with a menmos cluster.
type Client struct {
	httpClient    *http.Client
	host          string
	token         string
	maxRetryCount uint32
}

func New(host string, username string, password string) (*Client, error) {
	var err error
	// Block out redirections.
	// We need to handle those ourselves.
	customClient := http.Client{
		CheckRedirect: func(redirRequest *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	client := &Client{
		httpClient:    &customClient,
		host:          strings.TrimSuffix(host, "/"),
		token:         "",
		maxRetryCount: 40, // TODO: Make configurable.
	}

	client.token, err = client.authenticate(username, password)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// NewFromProfile initializes a new menmos client from its profile name.
func NewFromProfile(profileName string) (*Client, error) {
	profile, err := config.LoadProfileFromDefaultConfig(profileName)
	if err != nil {
		return nil, err
	}
	return New(profile.Host, profile.Username, profile.Password)
}

// low-level wrapper function to create an authenticated request to menmos.
func (c *Client) makeRequest(method string, path string, data io.Reader) (*http.Request, error) {
	request, err := http.NewRequest(method, c.host+path, data)
	if err != nil {
		return nil, errors.Wrapf(err, "%s %s - failed to create request", method, path)
	}

	if len(c.token) != 0 {
		request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	request.Header.Add("User-Agent", fmt.Sprintf("%s/%s", userAgent, Version))

	return request, nil
}

// Wrapper function to create a request that sends a JSON payload.
func (c *Client) makeJSONRequest(method string, path string, data interface{}) (*http.Request, error) {
	var dataReader io.Reader = nil
	if data != nil {
		bodyBytes, err := json.Marshal(&data)
		if err != nil {
			return nil, errors.Wrapf(err, "%s %s - failed to serialize body", method, path)
		}
		dataReader = bytes.NewReader(bodyBytes)
	}

	req, err := c.makeRequest(method, path, dataReader)
	if err != nil {
		return nil, err
	}

	if dataReader != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	return req, nil
}

// Performs a request and returns the redirect location.
func (c *Client) doWithRedirect(request *http.Request) (*url.URL, error) {
	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("%s %s - failed to perform redirect request", request.Method, request.URL))
	}

	if isTemporaryRedirect(resp.StatusCode) {
		redirectLocation, err := resp.Location()
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("%s %s - failed to get redirect location", request.Method, request.URL))
		}
		return redirectLocation, nil
	}

	return nil, fmt.Errorf("%s %s - expected redirect, got none", request.Method, request.URL)

}

func (c *Client) doJSONRequest(req *http.Request, response interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "%s %s - request failed", req.Method, req.URL)
	}
	defer resp.Body.Close()

	if !isStatusSuccess(resp.StatusCode) {
		return errors.New(fmt.Sprintf("%s %s - unexpected status '%s'", req.Method, req.URL, resp.Status))
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&response); err != nil {
		return errors.Wrapf(err, "%s %s - failed to deserialize response", req.Method, req.URL)
	}

	return nil
}

func (c *Client) authenticate(username string, password string) (string, error) {
	var response payload.LoginResponse

	request, err := c.makeJSONRequest("POST", "/auth/login", &payload.LoginRequest{Username: username, Password: password})
	if err != nil {
		return "", err
	}

	if err := c.doJSONRequest(request, &response); err != nil {
		return "", errors.Wrap(err, "failed to authenticate")
	}

	return response.Token, nil
}

func (c *Client) readRange(blobID string, start int64, end int64) (io.ReadCloser, error) {
	if start > end {
		return nil, fmt.Errorf("invalid range for read request: %d-%d", start, end)
	}

	req, err := c.makeJSONRequest("GET", fmt.Sprintf("/blob/%s", blobID), nil)
	if err != nil {
		return nil, err
	}

	redirectLocation, err := c.doWithRedirect(req)
	if err != nil {
		return nil, err
	}

	req.URL = redirectLocation
	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "read request failed")
	}

	return resp.Body, nil
}

func (c *Client) setMultipartRequestBody(payload io.ReadCloser, req *http.Request) error {
	if payload == nil {
		return nil
	}

	defer payload.Close()

	// TODO: This buffer thing isn't great - it loads the whole buffer to write in memory...
	// For better performance with very large files we'd need to improve this so its streaming instead.
	var bodyBuffer bytes.Buffer
	var err error
	w := multipart.NewWriter(&bodyBuffer)
	var fw io.Writer
	if fw, err = w.CreateFormField("src"); err != nil {
		return errors.Wrap(err, "failed to build multipart form body")
	}

	if _, err = io.Copy(fw, payload); err != nil {
		return errors.Wrap(err, "failed to build multipart form body")
	}
	if err := w.Close(); err != nil {
		return errors.Wrap(err, "failed to close body writer")
	}

	req.Body = io.NopCloser(&bodyBuffer)

	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	return nil
}

func (c *Client) pushInternal(path string, body io.ReadCloser, meta payload.BlobMeta) (string, error) {
	req, err := c.makeRequest("POST", path, nil)
	if err != nil {
		return "", err
	}

	metaBytes, err := json.Marshal(&meta)
	if err != nil {
		return "", errors.Wrap(err, "failed to serialize blob metadata")
	}
	req.Header.Add("X-Blob-Meta", base64.StdEncoding.EncodeToString(metaBytes))

	redirectLocation, err := c.doWithRedirect(req)
	if err != nil {
		return "", err
	}

	if err := c.setMultipartRequestBody(body, req); err != nil {
		return "", err
	}

	req.URL = redirectLocation
	req.Header.Add("X-Blob-Meta", base64.StdEncoding.EncodeToString(metaBytes))

	var response payload.PushResponse

	if err := c.doJSONRequest(req, &response); err != nil {
		return "", nil
	}

	return response.ID, nil
}

// IsHealthy returns whether the menmos cluster is healthy.
func (c *Client) IsHealthy() (bool, error) {
	var response payload.MessageResponse

	req, err := c.makeRequest("GET", "/health", nil)
	if err != nil {
		return false, err
	}

	if err := c.doJSONRequest(req, &response); err != nil {
		return false, errors.Wrap(err, "healthcheck failed")
	}

	return true, nil
}

// Query executes a query on the menmos cluster.
func (c *Client) Query(query *payload.Query) (*payload.QueryResponse, error) {
	var response payload.QueryResponse

	request, err := c.makeJSONRequest("POST", "/query", query)
	if err != nil {
		return nil, err
	}

	if err := c.doJSONRequest(request, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Get returns the body of the specified blob.
// If `readRange` is non-nil, Get will return that section of the blob.
func (c *Client) GetBody(blobID string, readRange *Range) (io.ReadCloser, error) {
	if readRange != nil {
		return &rangeReader{BlobID: blobID, Client: c, RangeStart: readRange.Start, RangeEnd: readRange.End}, nil
	}

	req, err := c.makeJSONRequest("GET", fmt.Sprintf("/blob/%s", blobID), nil)
	if err != nil {
		return nil, err
	}

	redirectLocation, err := c.doWithRedirect(req)
	if err != nil {
		return nil, err
	}

	req.URL = redirectLocation

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (c *Client) GetMetadata(blobID string) (payload.BlobMeta, error) {
	req, err := c.makeJSONRequest("GET", fmt.Sprintf("/blob/%s/metadata", blobID), nil)
	if err != nil {
		return payload.BlobMeta{}, err
	}

	var response payload.GetMetadataResponse
	if err := c.doJSONRequest(req, &response); err != nil {
		return payload.BlobMeta{}, err
	}

	if response.Metadata == nil {
		return payload.BlobMeta{}, fmt.Errorf("get meta: blob '%s' not found", blobID)
	}

	return *response.Metadata, nil
}

// Delete deletes a blob from the cluster.
func (c *Client) Delete(blobID string) error {
	req, err := c.makeJSONRequest("DELETE", fmt.Sprintf("/blob/%s", blobID), nil)
	if err != nil {
		return err
	}

	redirectLocation, err := c.doWithRedirect(req)
	if err != nil {
		return err
	}

	req.URL = redirectLocation

	var response payload.MessageResponse
	if err := c.doJSONRequest(req, &response); err != nil {
		return err
	}

	return nil
}

// Push creates a blob with the provided body and metadata to the cluster.
// If the body is nil, the blob is created empty.
func (c *Client) CreateBlob(body io.ReadCloser, meta payload.BlobMeta) (string, error) {
	return c.pushInternal("/blob", body, meta)
}

// UpdateBlob updates the entirety of a blob's contents and metadata at once.
func (c *Client) UpdateBlob(blobID string, body io.ReadCloser, meta payload.BlobMeta) error {
	_, err := c.pushInternal(fmt.Sprintf("/blob/%s", blobID), body, meta)
	return err
}

// UpdateMeta updates exclusively the blob metadata.
func (c *Client) UpdateMeta(blobID string, meta payload.BlobMeta) error {
	var response payload.MessageResponse
	req, err := c.makeJSONRequest("PUT", fmt.Sprintf("/blob/%s/metadata", blobID), &meta)
	if err != nil {
		return err
	}

	redirectLocation, err := c.doWithRedirect(req)
	if err != nil {
		return err
	}

	// Go doesn't like keeping the body for both requests, we'll rebuild from scratch.
	req, err = c.makeJSONRequest("PUT", fmt.Sprintf("/blob/%s/metadata", blobID), &meta)
	if err != nil {
		return err
	}

	req.URL = redirectLocation

	if err := c.doJSONRequest(req, &response); err != nil {
		return err
	}
	return nil
}

// Lists all storage nodes in the cluster.
func (c *Client) ListStorageNodes() ([]payload.StorageNodeInfo, error) {
	var response payload.ListStorageNodesResponse

	req, err := c.makeJSONRequest("GET", "/node/storage", nil)
	if err != nil {
		return nil, err
	}

	if err := c.doJSONRequest(req, &response); err != nil {
		return nil, err
	}

	return response.StorageNodes, nil
}
