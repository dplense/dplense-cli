package microsoft

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	auth "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
)

// newGraphService creates an authenticated Microsoft Graph client and request adapter
// using client credentials flow (Azure AD app registration).
func newGraphService(tenantID, clientID, clientSecret string) (*msgraphsdk.GraphServiceClient, abstractions.RequestAdapter, error) {
	if tenantID == "" || clientID == "" || clientSecret == "" {
		return nil, nil, fmt.Errorf("microsoft config requires tenant_id, client_id, and client_secret")
	}

	credential, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("azure credential: %w", err)
	}

	authProvider, err := auth.NewAzureIdentityAuthenticationProviderWithScopes(
		credential,
		[]string{"https://graph.microsoft.com/.default"},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("azure auth provider: %w", err)
	}

	adapter, err := msgraphsdk.NewGraphRequestAdapter(authProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("graph adapter: %w", err)
	}

	return msgraphsdk.NewGraphServiceClient(adapter), adapter, nil
}
