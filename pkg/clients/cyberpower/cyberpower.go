package cyberpower

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ManagementNodeType int
type NodeState int
type LoadState int
type BatteryState int
type UPSState int
type SourceState int

const (
	ManagementNodeTypeNone ManagementNodeType = iota
	ManagementNodeTypeRoot
	ManagementNodeTypeGroup
	ManagementNodeTypeDaisyChainPDUGroup
	ManagementNodeTypeUPS
	ManagementNodeTypePDU
	ManagementNodeTypeATS
	ManagementNodeTypeLoadAgent
	ManagementNodeTypeLoadClient
	ManagementNodeTypeLoadEquipment
	ManagementNodeTypeLoadPDU
	ManagementNodeTypeLoadATS
	ManagementNodeTypeLoadESXi
	ManagementNodeTypeLoadStorage
	ManagementNodeTypeLoadEmpty
	ManagementNodeTypeVCenter
	ManagementNodeTypeCluster
	ManagementNodeTypeESXi
	ManagementNodeTypeStorage
	ManagementNodeTypeVAPP
	ManagementNodeTypeVM
	ManagementNodeTypeBankThreePhasePDU
)

const (
	NodeStateNone NodeState = iota
	NodeStateSevere
	NodeStateWarning
	NodeStateNormal
	NodeStateUntouched
	NodeStatePowerOff
	NodeStateShutdown
	NodeStateStart
	NodeStateStarting
	NodeStateStopped
	NodeStateStopping
)

const (
	LoadStateNoLoad LoadState = iota
	LoadStateNormal
	LoadStateLowLoad
	LoadStateNearOverLoad
	LoadStateOverLoad
)

const (
	BatteryStateNormal BatteryState = iota
	BatteryStateNotPresent
	BatteryStateCharging
	BatteryStateDischarging
	BatteryStateFullyCharged
	BatteryStateLowBattery
	BatteryStateBatteryTest
)

const (
	UPSStateUPSPowerFailure UPSState = iota
	UPSStateUPSBuck
	UPSStateUPSBoost
	UPSStateUPSBypass
	UPSStateUPSNormal
)

const (
	SourceStateSelectedNormal SourceState = iota
	SourceStateSelectedFailure
	SourceStateUnselectedNormal
	SourceStateUnselectedFailure
)

type Client struct {
	httpClient      *http.Client
	powerPanelToken string
	tokenExpiration time.Time
	hashedUsername  string
	hashedPassword  string
	powerPanelURL   string
}

type ManagementTreeResponse struct {
	ChildrenNodeList []Node `json:"childrenNodeList"`
}

type Node struct {
	ID        int                `json:"id"`
	Type      ManagementNodeType `json:"type"`
	Name      string             `json:"name"`
	NodeBrief NodeBrief          `json:"nodeBrief"`
}

type NodeBrief struct {
	NodeState            NodeState  `json:"nodeState"`
	StateDescriptionList []string   `json:"stateDescriptionList"`
	OutputLoad           OutputLoad `json:"outputLoad"`
}

type OutputLoad struct {
	Percentage   int     `json:"percentage"`
	CurrentWatts float32 `json:"currentWatts"`
}

func NewClient(powerPanelURL string, hashedUsername, hashedPassword string, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	client := &Client{
		powerPanelToken: "undefined",
		tokenExpiration: time.Time{},
		hashedUsername:  hashedUsername,
		hashedPassword:  hashedPassword,
		powerPanelURL:   powerPanelURL,
		httpClient:      httpClient,
	}
	if err := client.RefreshToken(); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) buildRequest(method, path string, data io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", c.powerPanelURL, path), data)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.powerPanelToken))
	return req, nil
}

func (c *Client) RefreshToken() error {
	token, err := c.getAuthToken()
	if err != nil {
		return err
	}
	c.tokenExpiration = time.Now().Add(30 * time.Minute)
	c.powerPanelToken = token
	return nil
}

func (c *Client) getAuthToken() (string, error) {
	req, err := c.buildRequest(
		http.MethodPost,
		"rest/v1/login/verify",
		bytes.NewBuffer([]byte(fmt.Sprintf("{\"userName\":\"%s\",\"password\":\"%s\"}", c.hashedUsername, c.hashedPassword))),
	)
	if err != nil {
		return "", err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	header, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	tokenHeader := strings.Trim(string(header), `"`)
	tokenParts := strings.Split(tokenHeader, " ")
	if len(tokenParts) != 2 {
		return "", fmt.Errorf("auth header does not contain correct number of parts (%v)", len(tokenParts))
	}
	return tokenParts[1], nil
}

func (c *Client) GetManagementTree() (*ManagementTreeResponse, error) {
	// check if a token refresh is needed
	if c.tokenExpiration.Before(time.Now()) {
		if err := c.RefreshToken(); err != nil {
			return nil, err
		}
	}

	req, err := c.buildRequest(http.MethodGet, "rest/v1/equipment/management_tree", nil)
	if err != nil {
		return nil, err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var managementTreeResponse ManagementTreeResponse
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &managementTreeResponse); err != nil {
		return nil, err
	}
	return &managementTreeResponse, nil
}
