package tiflowapi

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	httputil "github.com/pingcap/tiflow-operator/pkg/util/http"
)

const (
	DefaultTimeout = 5 * time.Second
)

// MasterClient provides master server's api
type MasterClient interface {
	// GetMasters returns all master members from cluster
	GetMasters() ([]*MastersInfo, error)
	GetExecutors() ([]*WorkersInfo, error)
	GetLeader() (LeaderInfo, error)
	EvictLeader() error
	DeleteMaster(name string) error
	DeleteExecutor(name string) error
}

const leaderPrefix = "api/v1/leader"

type MastersInfo struct {
	Name      string `json:"name,omitempty"`
	MemberID  string `json:"memberID,omitempty"`
	Alive     bool   `json:"alive,omitempty"`
	ClientURL string `json:"clientURLs,omitempty"`
}

type WorkersInfo struct {
	Name string `json:"name,omitempty"`
	Addr string `json:"addr,omitempty"`
}

type LeaderInfo struct {
	AdvertiseAddr string `json:"advertise_addr,omitempty"`
}

// masterClient is default implementation of MasterClient
type masterClient struct {
	url        string
	httpClient *http.Client
}

func (c masterClient) GetMasters() ([]*MastersInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (c masterClient) GetExecutors() ([]*WorkersInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (c masterClient) GetLeader() (LeaderInfo, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, leaderPrefix)
	body, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return LeaderInfo{}, err
	}
	leaderInfo := LeaderInfo{}
	err = json.Unmarshal(body, &leaderInfo)
	return leaderInfo, err
}

func (c masterClient) EvictLeader() error {
	apiURL := fmt.Sprintf("%s/%s", c.url, leaderPrefix)
	_, err := httputil.PostBodyOK(c.httpClient, apiURL, nil)
	return err
}

func (c masterClient) DeleteMaster(name string) error {
	//TODO implement me
	panic("implement me")
}

func (c masterClient) DeleteExecutor(name string) error {
	//TODO implement me
	panic("implement me")
}

// NewMasterClient returns a new MasterClient
func NewMasterClient(url string, timeout time.Duration, tlsConfig *tls.Config) MasterClient {
	return &masterClient{
		url: url,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: &http.Transport{TLSClientConfig: tlsConfig, DisableKeepAlives: true},
		},
	}
}
