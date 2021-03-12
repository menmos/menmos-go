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

	"github.com/menmos/menmos-go/config"
	"github.com/menmos/menmos-go/payload"
	"github.com/pkg/errors"
)

// Client provides an API to interact with a menmos cluster.
type Client struct {
	httpClient    *http.Client
	host          string
	token         string
	maxRetryCount uint32
}

// NewFromProfile initializes a new menmos client from its profile name.
func NewFromProfile(profileName string) (*Client, error) {
	profile, err := config.LoadProfileByName(profileName)
	if err != nil {
		return nil, err
	}
	customClient := http.Client{
		CheckRedirect: func(redirRequest *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	client := &Client{
		httpClient:    &customClient,
		host:          profile.Host, // TODO: Ensure host is *not* slash-terminated.
		token:         "",
		maxRetryCount: 40, // TODO: Make configurable.
	}

	client.token, err = client.authenticate(profile.Username, profile.Password)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) makeRequest(method string, path string, data io.Reader) (*http.Request, error) {
	request, err := http.NewRequest(method, c.host+path, data)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create %s request", method)
	}

	if len(c.token) != 0 {
		request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	request.Header.Add("User-Agent", "menmos-go/0.1") // TODO: Extract "menmos-go" as constant and properly detect version.

	return request, nil
}

func (c *Client) makeJSONRequest(method string, path string, data interface{}) (*http.Request, error) {
	var dataReader io.Reader = nil
	if data != nil {
		bodyBytes, err := json.Marshal(&data)
		fmt.Printf("payload data: %s\n", string(bodyBytes))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to serialize %s body", method)
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

func (c *Client) doWithRedirect(request *http.Request) (*url.URL, error) {
	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 307 {
		// We have a redirect, retry.
		resp, err = c.httpClient.Do(request)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return resp.Location()
	}

	return nil, errors.New("no redirect")

}

func (c *Client) get(path string, response interface{}) error {
	req, err := c.makeJSONRequest("GET", path, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "GET request failed")
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&response); err != nil {
		return errors.Wrap(err, "failed to deseriallize GET response")
	}

	return nil
}

func (c *Client) doJSONRequest(req *http.Request, response interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "%s request failed", req.Method)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 || resp.StatusCode < 200 {
		return errors.New(fmt.Sprintf("unexpected status '%s'", resp.Status))
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&response); err != nil {
		return errors.Wrapf(err, "failed to deserialize %s response", req.Method)
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

// IsHealthy returns whether the menmos cluster is healthy.
func (c *Client) IsHealthy() (bool, error) {
	var response payload.MessageResponse

	if err := c.get("/health", &response); err != nil {
		return false, errors.Wrap(err, "healthcheck failed")
	}

	return true, nil
}

// Query executes a query on the menmos cluster.
func (c *Client) Query(query *payload.Query) (*payload.QueryResponse, error) {
	var response payload.QueryResponse

	request, err := c.makeJSONRequest("POST", "/query", query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create query request")
	}

	if err := c.doJSONRequest(request, &response); err != nil {
		return nil, errors.Wrap(err, "query failed")
	}

	return &response, nil
}

func (c *Client) readRange(blobID string, start int64, end int64) (io.ReadCloser, error) {
	if start > end {
		return nil, errors.New("invalid range for read request")
	}

	req, err := c.makeJSONRequest("GET", fmt.Sprintf("/blob/%s", blobID), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create read request")
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

func (c *Client) Get(blobID string, readRange *Range) (io.ReadCloser, error) {
	if readRange != nil {
		return &rangeReader{BlobID: blobID, Client: c, RangeStart: readRange.Start, RangeEnd: readRange.End}, nil
	}

	req, err := c.makeJSONRequest("GET", fmt.Sprintf("/blob/%s", blobID), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create read request")
	}

	redirectLocation, err := c.doWithRedirect(req)
	if err != nil {
		return nil, errors.Wrap(err, "redirect failed")
	}

	req.URL = redirectLocation

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "read request failed")
	}
	return resp.Body, nil
}

func (c *Client) Delete(blobID string) error {
	req, err := c.makeJSONRequest("DELETE", fmt.Sprintf("/blob/%s", blobID), nil)
	if err != nil {
		return errors.Wrap(err, "failed to create delete request")
	}

	redirectLocation, err := c.doWithRedirect(req)
	if err != nil {
		return errors.Wrap(err, "redirect failed")
	}

	req.URL = redirectLocation

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "delete request failed")
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return nil
}

func (c *Client) setMultipartRequestBody(payload io.ReadCloser, req *http.Request) error {
	if payload == nil {
		return nil
	}

	defer payload.Close()

	// TODO: This buffer thing isn't great - it loads the whole buffer to write in memory...
	// This would need to be seriously improved before a production release.
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
		fmt.Println("failed to close writer")
		return errors.Wrap(err, "failed to close writer")
	}

	req.Body = io.NopCloser(&bodyBuffer)

	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	return nil
}

func (c *Client) pushInternal(path string, body io.ReadCloser, meta payload.BlobMeta) (string, error) {

	req, err := c.makeRequest("POST", path, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create push request")
	}

	metaBytes, err := json.Marshal(&meta)
	if err != nil {
		return "", errors.Wrap(err, "failed to serialize blob metadata")
	}
	fmt.Println("Blob Meta: ", string(metaBytes))
	req.Header.Add("X-Blob-Meta", base64.StdEncoding.EncodeToString(metaBytes))

	redirectLocation, err := c.doWithRedirect(req)
	if err != nil {
		return "", errors.Wrap(err, "redirect failed")
	}

	if err := c.setMultipartRequestBody(body, req); err != nil {
		return "", err
	}

	req.URL = redirectLocation
	req.Header.Add("X-Blob-Meta", base64.StdEncoding.EncodeToString(metaBytes))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "push request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 || resp.StatusCode < 200 {
		return "", errors.New(fmt.Sprintf("unexpected status '%s'", resp.Status))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response body")
	}

	var response payload.PushResponse

	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return "", errors.Wrap(err, "failed to deserialize push response")
	}

	return response.ID, nil
}

func (c *Client) Push(body io.ReadCloser, meta payload.BlobMeta) (string, error) {
	return c.pushInternal("/blob", body, meta)
}

func (c *Client) UpdateBlob(blobID string, body io.ReadCloser, meta payload.BlobMeta) error {
	_, err := c.pushInternal(fmt.Sprintf("/blob/%s", blobID), body, meta)
	return err
}

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

	req, err = c.makeJSONRequest("PUT", fmt.Sprintf("/blob/%s/metadata", blobID), &meta)
	if err != nil {
		return err
	}

	req.URL = redirectLocation

	if err := c.doJSONRequest(req, &response); err != nil {
		return errors.Wrap(err, "failed to update metadata")
	}
	return nil
}
