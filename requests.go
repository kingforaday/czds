package czds

import (
	"fmt"
	"io"
	"time"
)

// Filters for RequestsFilter.Status
// Statuses for RequestStatus.Status
const (
	RequestAll       = ""
	RequestSubmitted = "Submitted"
	RequestPending   = "Pending"
	RequestApproved  = "Approved"
	RequestDenied    = "Denied"
	RequestRevoked   = "Revoked"
	RequestExpired   = "Expired"
)

// Filters for RequestsSort.Direction
const (
	SortAsc  = "asc"
	SortDesc = "desc"
)

// Filters for RequestsSort.Field
const (
	SortByTLD         = "tld"
	SortByStatus      = "status"
	SortByLastUpdated = "last_updated"
	SortByExpiration  = "expired"
	SortByCreated     = "created"
)

// Status from TLDStatus.CurrentStatus and RequestsInfo.Status
const (
	StatusAvailable = "available"
	StatusSubmitted = "submitted"
	StatusPending   = "pending"
	StatusApproved  = "approved"
	StatusDenied    = "denied"
	StatusExpired   = "expired"
	StatusRevoked   = "revoked" // unverified
)

// RequestsFilter is used to set what results should be returned by GetRequests
type RequestsFilter struct {
	Status     string             `json:"status"` // should be set to one of the Request* constants
	Filter     string             `json:"filter"` // zone name search
	Pagination RequestsPagination `json:"pagination"`
	Sort       RequestsSort       `json:"sort"`
}

// RequestsSort sets which field and direction the results for the RequestsFilter request should be returned with
type RequestsSort struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

// RequestsPagination sets the page size and offset for paginated results for RequestsFilter
type RequestsPagination struct {
	Size int `json:"size"`
	Page int `json:"page"`
}

// Request holds information about a request in RequestsResponse from GetRequests()
type Request struct {
	RequestID   string    `json:"requestId"`
	TLD         string    `json:"tld"`
	ULabel      string    `json:"ulable"` // UTF-8 decoded punycode, looks like API has a typo
	Status      string    `json:"status"` // should be set to one of the Request* constants
	Created     time.Time `json:"created"`
	LastUpdated time.Time `json:"last_updated"`
	Expired     time.Time `json:"expired"` // Note: epoch 0 means no expiration set
	SFTP        bool      `json:"sftp"`
}

// RequestsResponse holds Requests from from GetRequests() and total number of requests that match the query but may not be returned due to pagination
type RequestsResponse struct {
	Requests      []Request `json:"requests"`
	TotalRequests int64     `json:"totalRequests"`
}

// TLDStatus is information about a particular TLD returned from GetTLDStatus() or included in RequestsInfo
type TLDStatus struct {
	TLD           string `json:"tld"`
	ULabel        string `json:"ulable"`        // UTF-8 decoded punycode, looks like API has a typo
	CurrentStatus string `json:"currentStatus"` // should be set to one of the Status* constants
	SFTP          bool   `json:"sftp"`
}

// HistoryEntry contains a timestamp and description of action that happened for a RequestsInfo
// For example: requested, expired, approved, etc..
type HistoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
}

// FtpDetails contains FTP information for RequestsInfo
type FtpDetails struct {
	PrivateDataError bool `json:"privateDataError"`
}

// RequestsInfo contains the detailed information about a particular zone request returned by GetRequestInfo()
type RequestsInfo struct {
	RequestID        string         `json:"requestId"`
	TLD              *TLDStatus     `json:"tld"`
	FtpIps           []string       `json:"ftpips"`
	Status           string         `json:"status"` // should be set to one of the Status* constants
	TcVersion        string         `json:"tcVersion"`
	Created          time.Time      `json:"created"`
	RequestIP        string         `json:"requestIp"`
	Reason           string         `json:"reason"`
	LastUpdated      time.Time      `json:"last_updated"`
	Expired          time.Time      `json:"expired"` // Note: epoch 0 means no expiration set
	History          []HistoryEntry `json:"history"`
	FtpDetails       *FtpDetails    `json:"ftpDetails"`
	PrivateDataError bool           `json:"privateDataError"`
}

// RequestSubmission contains the information required to submit a new request with SubmitRequest()
type RequestSubmission struct {
	AllTLDs          bool     `json:"allTlds"`
	TLDNames         []string `json:"tldNames"`
	Reason           string   `json:"reason"`
	TcVersion        string   `json:"tcVersion"` // terms and conditions revision version
	AdditionalFTPIps []string `json:"additionalFtfIps,omitempty"`
}

// Terms holds the terms and conditions details from GetTerms()
type Terms struct {
	Version    string    `json:"version"`
	Content    string    `json:"content"`
	ContentURL string    `json:"contentUrl"`
	Created    time.Time `json:"created"`
}

// GetRequests searches for the status of zones requests as seen on the
// CZDS dashboard page "https://czds.icann.org/zone-requests/all"
func (c *Client) GetRequests(filter *RequestsFilter) (*RequestsResponse, error) {
	requests := new(RequestsResponse)
	err := c.jsonAPI("POST", "/czds/requests/all", filter, requests)
	return requests, err
}

// GetRequestInfo gets detailed information about a particular request and its timeline
// as seen on the CZDS dashboard page "https://czds.icann.org/zone-requests/{ID}"
func (c *Client) GetRequestInfo(requestID string) (*RequestsInfo, error) {
	request := new(RequestsInfo)
	err := c.jsonAPI("GET", "/czds/requests/"+requestID, nil, request)
	return request, err
}

// GetTLDStatus gets the current status of all TLDs and their ability to be requested
func (c *Client) GetTLDStatus() ([]TLDStatus, error) {
	requests := make([]TLDStatus, 0, 20)
	err := c.jsonAPI("GET", "/czds/tlds", nil, &requests)
	return requests, err
}

// GetTerms gets the current terms and conditions from the CZDS portal
// page "https://czds.icann.org/terms-and-conditions"
// this is required to accept the terms and conditions when submitting a new request
func (c *Client) GetTerms() (*Terms, error) {
	terms := new(Terms)
	// this does not appear to need auth, but we auth regardless
	err := c.jsonAPI("GET", "/czds/terms/condition", nil, terms)
	return terms, err
}

// SubmitRequest submits a new request for access to new zones
func (c *Client) SubmitRequest(request *RequestSubmission) error {
	err := c.jsonAPI("POST", "/czds/requests/create", request, nil)
	return err
}

// DownloadAllRequests outputs the contents of the csv file downloaded by
// the "Download All Requests" button on the CZDS portal to the provided output
func (c *Client) DownloadAllRequests(output io.Writer) error {
	url := c.BaseURL + "/czds/requests/report"
	resp, err := c.apiRequest(true, "GET", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	n, err := io.Copy(output, resp.Body)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%s was empty", url)
	}

	return nil
}

// RequestTLDs is a helper function that requests access to the provided tlds with the provided reason
// TLDs provided should be marked as able to request from GetTLDStatus()
func (c *Client) RequestTLDs(tlds []string, reason string) error {
	// get terms
	terms, err := c.GetTerms()
	if err != nil {
		return err
	}

	// submit request
	request := &RequestSubmission{
		TLDNames:  tlds,
		Reason:    reason,
		TcVersion: terms.Version,
	}
	err = c.SubmitRequest(request)
	return err
}

// RequestAllTLDs is a helper function to request access to all available TLDs with the provided reason
func (c *Client) RequestAllTLDs(reason string) ([]string, error) {
	// get available to request
	status, err := c.GetTLDStatus()
	if err != nil {
		return nil, err
	}
	// check to see if any available to request
	requestTLDs := make([]string, 0, 10)
	for _, tld := range status {
		switch tld.CurrentStatus {
		case StatusAvailable, StatusExpired, StatusDenied, StatusRevoked:
			requestTLDs = append(requestTLDs, tld.TLD)
		}
	}
	// if none, return now
	if len(requestTLDs) == 0 {
		return requestTLDs, nil
	}

	// get terms
	terms, err := c.GetTerms()
	if err != nil {
		return nil, err
	}

	// submit request
	request := &RequestSubmission{
		AllTLDs:   true,
		TLDNames:  requestTLDs,
		Reason:    reason,
		TcVersion: terms.Version,
	}
	err = c.SubmitRequest(request)
	return requestTLDs, err
}
