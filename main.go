package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Azure/aks-canipull/pkg/authorizer"
	"github.com/Azure/aks-canipull/pkg/utils"

	"github.com/Azure/aks-canipull/pkg/exitcode"
	"github.com/Azure/aks-canipull/pkg/log"
	az "github.com/Azure/go-autorest/autorest/azure"
	"k8s.io/legacy-cloud-providers/azure"

	flag "github.com/spf13/pflag"
)

const (
	DefaultAzureCfgPath string = "/etc/kubernetes/azure.json"
)

var (
	logLevel   *uint   = flag.UintP("verbose", "v", 2, "output verbosity level.")
	configPath *string = flag.String("config", "", "the azure.json config file path.")
)

func main() {
	ctx := log.WithLogLevel(context.Background(), *logLevel)
	time.Sleep(5 * time.Second)
	exitCode := Execute(ctx)
	os.Exit(exitCode)
}

func Execute(ctx context.Context) int {
	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Println("No ACR input. Expect `canipull myacr.azurecr.io`.")
		return 0
	}

	acr := flag.Args()[0]
	logger := log.FromContext(ctx)

	if _, err := net.LookupHost(acr); err != nil {
		logger.V(2).Info("Checking host name resolution (%s): FAILED", acr)
		logger.V(2).Info("Failed to resolve specified fqdn %s: %s", acr, err)
		return exitcode.DNSResolutionFailure
	}
	logger.V(2).Info("Checking host name resolution (%s): SUCCEEDED", acr)

	cname, err := net.LookupCNAME(acr)
	if err != nil {
		logger.V(2).Info("Checking CNAME (%s): FAILED", acr)
		logger.V(2).Info("Failed to get CNAME of the ACR: %s", err)
		return exitcode.DNSResolutionFailure
	}

	logger.V(2).Info("Canonical name for ACR (%s): %s", acr, cname)

	acrLocation := strings.Split(cname, ".")[1]
	if !strings.EqualFold(acrLocation, "privatelink") {
		logger.V(2).Info("ACR location: %s", acrLocation)
	}

	azConfigPath := *configPath
	if *configPath == "" {
		azConfigPath = DefaultAzureCfgPath
	}

	logger.V(6).Info("Loading azure.json file from %s", azConfigPath)
	if _, err := os.Stat(azConfigPath); err != nil {
		logger.V(2).Info("Failed to load azure.json. Are you running inside Kubernetes on Azure? \n")
		return exitcode.AzureConfigNotFound
	}

	var cfg azure.Config
	configBytes, err := ioutil.ReadFile(azConfigPath)
	if err != nil {
		logger.V(2).Info("Failed to read azure.json file: %s \n", err)
		return exitcode.AzureConfigReadFailure
	}

	if err := json.Unmarshal(configBytes, &cfg); err != nil {
		logger.V(2).Info("Failed to read azure.json file: %s", err)
		return exitcode.AzureConfigUnmarshalFailure
	}

	if !utils.LocationEquals(acrLocation, cfg.Location) && !strings.EqualFold(acrLocation, "privatelink") {
		logger.V(2).Info("Checking ACR location matches cluster location: FAILED")
		logger.V(2).Info("ACR location '%s' does not match your cluster location '%s'. This may result in slow image pulls and extra cost.", acrLocation, cfg.Location)
	}

	if cfg.AADClientID == "msi" && cfg.AADClientSecret == "msi" {
		logger.V(2).Info("Checking managed identity...")
		return validateMsiAuth(ctx, acr, cfg, logger)
	}

	logger.V(4).Info("The cluster uses service principal.")
	return validateServicePrincipalAuth(ctx, acr, cfg, logger)
}

func validateMsiAuth(ctx context.Context, acr string, cfg azure.Config, logger *log.Logger) int {
	logger.V(6).Info("Cluster cloud name: %s", cfg.Cloud)

	env, err := az.EnvironmentFromName(cfg.Cloud)
	if err != nil {
		logger.V(2).Info("Unknown Azure cloud name: %s", cfg.Cloud)
		return exitcode.AzureCloudUnknown
	}

	logger.V(2).Info("Kubelet managed identity client ID: %s", cfg.UserAssignedIdentityID)
	tr := authorizer.NewTokenRetriever(env.ActiveDirectoryEndpoint, env.ResourceManagerEndpoint)
	token, err := tr.AcquireARMTokenMSI(ctx, cfg.UserAssignedIdentityID)
	if err != nil {
		logger.V(2).Info("Validating managed identity existance: FAILED")
		logger.V(2).Info("Getting managed identity token failed with: %s", err)
		return exitcode.ServicePrincipalCredentialInvalid
	}
	logger.V(2).Info("Validating managed identity existance: SUCCEEDED")
	logger.V(9).Info("ARM access token: %s", token)

	te := authorizer.NewTokenExchanger()
	acrToken, err := te.ExchangeACRAccessToken(token, acr)
	if err != nil {
		logger.V(2).Info("Validating image pull permission: FAILED")
		logger.V(2).Info("ACR %s rejected token exchange: %s", acr, err)
		return exitcode.MissingImagePullPermision
	}

	logger.V(2).Info("Validating image pull permission: SUCCEEDED")
	logger.V(9).Info("ACR access token: %s", acrToken)
	logger.V(2).Info("\nYour cluster can pull images from %s!", acr)
	return 0
}

func validateServicePrincipalAuth(ctx context.Context, acr string, cfg azure.Config, logger *log.Logger) int {
	env, err := az.EnvironmentFromName(cfg.Cloud)
	if err != nil {
		logger.V(2).Info("Unknown Azure cloud name: %s", cfg.Cloud)
		return exitcode.AzureCloudUnknown
	}

	tr := authorizer.NewTokenRetriever(env.ActiveDirectoryEndpoint, env.ResourceManagerEndpoint)
	token, err := tr.AcquireARMTokenSP(ctx, cfg.AADClientID, cfg.AADClientSecret, cfg.TenantID)
	if err != nil {
		logger.V(2).Info("Validating service principal credential: FAILED")
		logger.V(2).Info("Sign in to AAD failed with: %s", err)
		return exitcode.ServicePrincipalCredentialInvalid
	}
	logger.V(2).Info("Validating service principal credential: SUCCEEDED")
	logger.V(9).Info("ARM access token: %s", token)

	te := authorizer.NewTokenExchanger()
	acrToken, err := te.ExchangeACRAccessToken(token, acr)
	if err != nil {
		logger.V(2).Info("Validating image pull permission: FAILED")
		logger.V(2).Info("ACR %s rejected token exchange: %s", acr, err)
		return exitcode.MissingImagePullPermision
	}

	logger.V(2).Info("Validating image pull permission: SUCCEEDED")
	logger.V(9).Info("ACR access token: %s", acrToken)
	logger.V(2).Info("\nYour cluster can pull images from %s!", acr)
	return 0
}
