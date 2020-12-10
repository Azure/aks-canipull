package authorizer

import "github.com/Azure/msi-acrpull/pkg/authorizer/types"

//go:generate sh -c "mockgen github.com/Azure/msi-acrpull/pkg/authorizer Interface,ManagedIdentityTokenRetriever,ACRTokenExchanger > ./mock_$GOPACKAGE/interfaces.go"

// ManagedIdentityTokenRetriever is the interface to acquire an ARM access token.
type ManagedIdentityTokenRetriever interface {
	AcquireARMToken(clientID string, resourceID string) (types.AccessToken, error)
}

// ACRTokenExchanger is the interface to exchange an ACR access token.
type ACRTokenExchanger interface {
	ExchangeACRAccessToken(armToken types.AccessToken, acrFQDN string) (types.AccessToken, error)
}
