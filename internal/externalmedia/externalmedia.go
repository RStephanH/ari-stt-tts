package externalmedia

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/charmbracelet/log"
)

// ExternalMediaParams contains parameters for external media
type ExternalMediaParams struct {
	ARIBaseURL string //ex : http://localhost:8088
	Username   string
	Password   string

	AppName string //ex : "myapp"
	HostIP  string //ex : "127.0.0.1"
	Port    int    //ex : 4002
	Format  string //ex : "slin16"
}

type ExternalMediaResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	State       string `json:"state"`
	ChannelVars struct {
		RTPAddress string `json:"UNICASTRTP_LOCAL_ADDRESS"`
		RTPPort    string `json:"UNICASTRTP_LOCAL_PORT"`
	} `json:"channelvars"`
}

// CreateExternalMedia creates the externalmedia channel on asterisk server
func CreateExternalMedia(p ExternalMediaParams) (string, error) {

	endpoint := fmt.Sprintf("%s/ari/channels/externalMedia", p.ARIBaseURL)
	log.Info("Creating External Media Channel", "endpoint", endpoint)

	params := url.Values{}
	params.Set("app", p.AppName)
	params.Set("external_host", fmt.Sprintf("%s:%d", p.HostIP, p.Port))
	params.Set("format", p.Format)

	reqURL := endpoint + "?" + params.Encode()
	log.Info("External Media Request URL", "url", reqURL)
	req, err := http.NewRequest("POST", reqURL, nil)
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(p.Username, p.Password)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("asterisk returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// The JSON contains a field "id" -> channel ID of the external media
	return string(bodyBytes), nil
}
