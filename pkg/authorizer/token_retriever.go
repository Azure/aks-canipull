package authorizer

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/Azure/aks-canipull/pkg/log"

	"github.com/Azure/go-autorest/autorest/adal"

	"github.com/Azure/msi-acrpull/pkg/authorizer/types"
)

const (
	msiMetadataEndpoint = "http://169.254.169.254/metadata/identity/oauth2/token"
)

// TokenRetriever is an instance of ManagedIdentityTokenRetriever
type TokenRetriever struct {
	metadataEndpoint        string
	resourceManagerEndpoint string
	activeDirectoryEndpoint string
}

// NewTokenRetriever returns a new token retriever
func NewTokenRetriever(activeDirectoryEndpoint string, resourceManagerEndpoint string) *TokenRetriever {
	return &TokenRetriever{
		metadataEndpoint:        msiMetadataEndpoint,
		resourceManagerEndpoint: resourceManagerEndpoint,
		activeDirectoryEndpoint: activeDirectoryEndpoint,
	}
}

// AcquireARMTokenMSI acquires the managed identity ARM access token
func (tr *TokenRetriever) AcquireARMTokenMSI(ctx context.Context, clientID string) (types.AccessToken, error) {
	token, err := tr.refreshToken(ctx, clientID, "")
	if err != nil {
		return "", fmt.Errorf("failed to refresh ARM access token: %w", err)
	}

	return token, nil
}

// AcquireARMTokenSP acquires the service principal ARM access token
func (tr *TokenRetriever) AcquireARMTokenSP(ctx context.Context, clientID, clientSecret, tenantID string) (types.AccessToken, error) {
	oauthConfig, err := adal.NewOAuthConfig(tr.activeDirectoryEndpoint, tenantID)
	if err != nil {
		return "", fmt.Errorf("failed to get OAuth config: %w", err)
	}

	spt, err := adal.NewServicePrincipalToken(*oauthConfig, clientID, clientSecret, tr.resourceManagerEndpoint)
	if err != nil {
		return "", fmt.Errorf("failed to get ARM access token: %w", err)
	}

	err = spt.Refresh()
	if err != nil {
		return "", fmt.Errorf("failed to refresh ARM access token: %w", err)
	}

	return types.AccessToken(spt.OAuthToken()), nil
}

func (tr *TokenRetriever) refreshToken(ctx context.Context, clientID, resourceID string) (types.AccessToken, error) {
	logger := log.FromContext(ctx)

	msiEndpoint, err := url.Parse(tr.metadataEndpoint)
	if err != nil {
		return "", err
	}

	parameters := url.Values{}
	if clientID != "" {
		parameters.Add("client_id", clientID)
	} else {
		parameters.Add("mi_res_id", resourceID)
	}

	parameters.Add("resource", tr.resourceManagerEndpoint)
	parameters.Add("api-version", "2018-02-01")

	msiEndpoint.RawQuery = parameters.Encode()

	logger.V(9).Info("GET %s", msiEndpoint.String())

	req, err := http.NewRequest("GET", msiEndpoint.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Metadata", "true")

	client := &http.Client{}
	var resp *http.Response
	defer closeResponse(resp)

	resp, err = client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send metadata endpoint request: %w", err)
	}

	if resp.StatusCode != 200 {
		responseBytes, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("Metadata endpoint returned error status: %d. body: %s", resp.StatusCode, string(responseBytes))
	}

	responseBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read metadata endpoint response: %w", err)
	}

	var tokenResp tokenResponse
	err = json.Unmarshal(responseBytes, &tokenResp)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal metadata endpoint response: %w", err)
	}

	return types.AccessToken(tokenResp.AccessToken), nil
}

func closeResponse(resp *http.Response) {
	if resp == nil {
		return
	}
	resp.Body.Close()
}
