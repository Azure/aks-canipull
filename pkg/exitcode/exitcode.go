package exitcode

const (
	AzureConfigNotFound               int = 1
	AzureConfigReadFailure            int = 2
	AzureConfigUnmarshalFailure       int = 3
	AzureCloudUnknown                 int = 4
	ServicePrincipalCredentialInvalid int = 10
	DNSResolutionFailure              int = 52
	MissingImagePullPermision         int = 60
)
