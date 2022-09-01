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
	GetMasters() (MastersInfo, error)
	GetExecutors() (ExecutorsInfo, error)
	GetLeader() (LeaderInfo, error)
	EvictLeader() error
	DeleteMaster(name string) error
	DeleteExecutor(name string) error
}

const (
	leaderPrefix        = "api/v1/leader"
	leaderResignPrefix  = "api/v1/leader/resign"
	listMastersPrefix   = "api/v1/masters"
	listExecutorsPrefix = "api/v1/executors"
)

type Master struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Address  string `json:"address,omitempty"`
	IsLeader bool   `json:"is_leader,omitempty"`
}

type MastersInfo struct {
	Masters []*Master `json:"masters,omitempty"`
}

type Executor struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	Address    string `json:"address,omitempty"`
	Capability string `json:"capability,omitempty"`
}

type ExecutorsInfo struct {
	Executors []*Executor `json:"executors,omitempty"`
}

type LeaderInfo struct {
	AdvertiseAddr string `json:"advertise_addr,omitempty"`
}

// masterClient is default implementation of MasterClient
type masterClient struct {
	url        string
	httpClient *http.Client
}

func (c masterClient) GetMasters() (MastersInfo, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, listMastersPrefix)
	body, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return MastersInfo{}, err
	}

	var masters MastersInfo
	err = json.Unmarshal(body, &masters)
	return masters, err
}

func (c masterClient) GetExecutors() (ExecutorsInfo, error) {
	apiURL := fmt.Sprintf("%s/%s", c.url, listExecutorsPrefix)
	body, err := httputil.GetBodyOK(c.httpClient, apiURL)
	if err != nil {
		return ExecutorsInfo{}, err
	}

	var executors ExecutorsInfo
	err = json.Unmarshal(body, &executors)
	return executors, err
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
	apiURL := fmt.Sprintf("%s/%s", c.url, leaderResignPrefix)
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
