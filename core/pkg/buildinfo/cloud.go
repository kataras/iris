package buildinfo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// CloudProvider represents the cloud provider.
type CloudProvider string

const (
	ProviderUnknown CloudProvider = "Unknown"
	ProviderAWS     CloudProvider = "AWS"
	ProviderAzure   CloudProvider = "Azure"
	ProviderGCP     CloudProvider = "GCP"
)

// String returns the string representation of the CloudProvider.
func (p CloudProvider) String() string {
	return string(p)
}

// AWSPlatform represents the type of AWS container platform.
type AWSPlatform string

const (
	AWSPlatformUnknown   AWSPlatform = "Unknown"
	AWSPlatformECS       AWSPlatform = "ECS"
	AWSPlatformEKS       AWSPlatform = "EKS"
	AWSPlatformFargate   AWSPlatform = "Fargate"
	AWSPlatformLambda    AWSPlatform = "Lambda"
	AWSPlatformEC2       AWSPlatform = "EC2"
	AWSPlatformAppRunner AWSPlatform = "AppRunner"
	AWSPlatformBatch     AWSPlatform = "Batch"
	AWSPlatformLightsail AWSPlatform = "Lightsail"
)

// AzurePlatform represents the type of Azure container platform.
type AzurePlatform string

const (
	AzurePlatformUnknown           AzurePlatform = "Unknown"
	AzurePlatformContainerApps     AzurePlatform = "ContainerApps"
	AzurePlatformAppService        AzurePlatform = "AppService"
	AzurePlatformContainerInstance AzurePlatform = "ContainerInstance"
	AzurePlatformKubernetes        AzurePlatform = "AKS"
)

// GCPPlatform represents the type of GCP container platform.
type GCPPlatform string

const (
	GCPPlatformUnknown       GCPPlatform = "Unknown"
	GCPPlatformGKE           GCPPlatform = "GKE"           // Google Kubernetes Engine
	GCPPlatformCloudRun      GCPPlatform = "CloudRun"      // Cloud Run
	GCPPlatformAppEngine     GCPPlatform = "AppEngine"     // App Engine
	GCPPlatformComputeEngine GCPPlatform = "ComputeEngine" // Compute Engine
	GCPPlatformCloudFunction GCPPlatform = "CloudFunction" // Cloud Functions
)

// AWSContainerInfo provides detailed information about the AWS container environment.
type AWSContainerInfo struct {
	Platform         AWSPlatform `json:"platform"`
	IsAWSContainer   bool        `json:"is_aws_container"`
	TaskARN          string      `json:"task_arn,omitempty"`          // ECS Task ARN
	TaskID           string      `json:"task_id,omitempty"`           // ECS Task ID
	TaskFamily       string      `json:"task_family,omitempty"`       // ECS Task Family
	TaskRevision     string      `json:"task_revision,omitempty"`     // ECS Task Revision
	ContainerName    string      `json:"container_name,omitempty"`    // Name of the container
	ContainerID      string      `json:"container_id,omitempty"`      // ID of the container
	ClusterName      string      `json:"cluster_name,omitempty"`      // Name of the ECS/EKS cluster
	ServiceName      string      `json:"service_name,omitempty"`      // Name of the service (ECS, App Runner, etc.)
	Region           string      `json:"region,omitempty"`            // AWS region
	AvailabilityZone string      `json:"availability_zone,omitempty"` // Availability Zone (for EC2)
	AccountID        string      `json:"account_id,omitempty"`        // AWS Account ID
	ImageURI         string      `json:"image_uri,omitempty"`         // URI of the container image
	LogGroup         string      `json:"log_group,omitempty"`         // Log group for the container
	LogStream        string      `json:"log_stream,omitempty"`        // Log stream for the container
	CPU              string      `json:"cpu,omitempty"`               // CPU allocated to the container
	Memory           string      `json:"memory,omitempty"`            // Memory allocated to the container
	LaunchType       string      `json:"launch_type,omitempty"`       // Launch type (EC2 or Fargate)
	PodName          string      `json:"pod_name,omitempty"`          // Pod name (for EKS)
	Namespace        string      `json:"namespace,omitempty"`         // Namespace (for EKS)
	NodeName         string      `json:"node_name,omitempty"`         // Node name (for EKS)
	FunctionName     string      `json:"function_name,omitempty"`     // Lambda function name
	FunctionVersion  string      `json:"function_version,omitempty"`  // Lambda function version
	Hostname         string      `json:"hostname,omitempty"`          // Hostname of the container
}

// AzureContainerInfo holds information about the Azure container environment.
type AzureContainerInfo struct {
	Platform         AzurePlatform `json:"platform"`
	IsAzureContainer bool          `json:"is_azure_container"`
	ContainerAppName string        `json:"container_app_name,omitempty"` // Name of the Azure Container App
	Revision         string        `json:"revision,omitempty"`           // Revision of the Azure Container App
	Replica          string        `json:"replica,omitempty"`            // Replica name or ID
	Region           string        `json:"region,omitempty"`             // Azure region
	ResourceGroup    string        `json:"resource_group,omitempty"`     // Resource group name
	SubscriptionID   string        `json:"subscription_id,omitempty"`    // Azure subscription ID
	Environment      string        `json:"environment,omitempty"`        // Environment name (for Container Apps)
	AppName          string        `json:"app_name,omitempty"`           // Application name (if applicable)
	PodName          string        `json:"pod_name,omitempty"`           // Pod name (for AKS)
	Namespace        string        `json:"namespace,omitempty"`          // Namespace (for AKS)
}

// GCPContainerInfo provides detailed information about the GCP container environment.
type GCPContainerInfo struct {
	Platform           GCPPlatform `json:"platform"`
	IsGCPContainer     bool        `json:"is_gcp_container"`
	ProjectID          string      `json:"project_id,omitempty"`           // GCP Project ID
	ServiceName        string      `json:"service_name,omitempty"`         // Service name (for Cloud Run)
	ServiceVersion     string      `json:"service_version,omitempty"`      // Service version (for Cloud Run)
	ServiceRevision    string      `json:"service_revision,omitempty"`     // Service revision (for Cloud Run)
	InstanceID         string      `json:"instance_id,omitempty"`          // Instance ID (for Cloud Run, Functions)
	InstanceName       string      `json:"instance_name,omitempty"`        // Instance name (for Compute Engine)
	Region             string      `json:"region,omitempty"`               // GCP region
	Zone               string      `json:"zone,omitempty"`                 // GCP zone (for Compute Engine)
	ClusterName        string      `json:"cluster_name,omitempty"`         // GKE cluster name
	PodName            string      `json:"pod_name,omitempty"`             // Pod name (for GKE)
	Namespace          string      `json:"namespace,omitempty"`            // Namespace (for GKE)
	NodeName           string      `json:"node_name,omitempty"`            // Node name (for GKE)
	ContainerName      string      `json:"container_name,omitempty"`       // Container name (for GKE)
	FunctionName       string      `json:"function_name,omitempty"`        // Cloud Function name
	FunctionRegion     string      `json:"function_region,omitempty"`      // Cloud Function region
	FunctionEntryPoint string      `json:"function_entry_point,omitempty"` // Cloud Function entry point
	LogName            string      `json:"log_name,omitempty"`             // Log name for the service
	TraceID            string      `json:"trace_id,omitempty"`             // Trace ID for Cloud Functions
	Hostname           string      `json:"hostname,omitempty"`             // Hostname of the container
	ServiceAccount     string      `json:"service_account,omitempty"`      // Service account email (for Cloud Run, Functions)
	NumericProjectID   string      `json:"numeric_project_id,omitempty"`   // Numeric project ID (for GCP)
}

// ECSTaskMetadata represents ECS task metadata from the metadata endpoint.
type ECSTaskMetadata struct {
	Cluster          string `json:"Cluster"`
	TaskARN          string `json:"TaskARN"`
	Family           string `json:"Family"`
	Revision         string `json:"Revision"`
	DesiredStatus    string `json:"DesiredStatus"`
	KnownStatus      string `json:"KnownStatus"`
	LaunchType       string `json:"LaunchType"`
	AvailabilityZone string `json:"AvailabilityZone"`
	Containers       []struct {
		Name        string `json:"Name"`
		ContainerID string `json:"ContainerID"`
		Image       string `json:"Image"`
		ImageID     string `json:"ImageID"`
		LogOptions  struct {
			LogGroup  string `json:"awslogs-group"`
			LogStream string `json:"awslogs-stream"`
			Region    string `json:"awslogs-region"`
		} `json:"LogOptions"`
	} `json:"Containers"`
}

// GCPMetadata represents metadata from GCP metadata server.
type GCPMetadata struct {
	ProjectID        string `json:"project-id"`
	NumericProjectID int64  `json:"numeric-project-id"`
	InstanceID       string `json:"instance-id"`
	InstanceName     string `json:"instance-name"`
	Zone             string `json:"zone"`
}

// ContainerInfo provides unified container information across cloud providers.
type ContainerInfo struct {
	Provider      CloudProvider `json:"provider"`
	Platform      string        `json:"platform"`                 // Specific platform (ECS, AKS, GKE, etc.)
	IsContainer   bool          `json:"is_container"`             // Indicates if running in a container
	AppName       string        `json:"app_name,omitempty"`       // Application name (if applicable)
	ContainerID   string        `json:"container_id,omitempty"`   // Container ID
	ContainerName string        `json:"container_name,omitempty"` // Container name
	Revision      string        `json:"revision,omitempty"`       // Revision or version of the container
	Replica       string        `json:"replica,omitempty"`        // Replica name or ID
	Cluster       string        `json:"cluster,omitempty"`        // Cluster name (if applicable)
	Region        string        `json:"region,omitempty"`         // Cloud region
	AccountID     string        `json:"account_id,omitempty"`     // Account ID (if applicable)
	ProjectID     string        `json:"project_id,omitempty"`     // For GCP
	Hostname      string        `json:"hostname,omitempty"`       // Hostname of the container
	Environment   string        `json:"environment,omitempty"`    // Environment (e.g., production, staging)
	LogGroup      string        `json:"log_group,omitempty"`      // Log group (if applicable)
	LogStream     string        `json:"log_stream,omitempty"`     // Log stream (if applicable)

	// Provider-specific info.
	AWS   *AWSContainerInfo   `json:"aws,omitempty"`
	Azure *AzureContainerInfo `json:"azure,omitempty"`
	GCP   *GCPContainerInfo   `json:"gcp,omitempty"`
}

// getHostname safely gets the hostname.
func getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

// DetectAWSContainer detects and returns AWS container information.
func DetectAWSContainer() *AWSContainerInfo {
	info := &AWSContainerInfo{
		Platform: AWSPlatformUnknown,
		Hostname: getHostname(),
	}

	// Check for AWS Lambda.
	if lambdaFunctionName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME"); lambdaFunctionName != "" {
		info.IsAWSContainer = true
		info.Platform = AWSPlatformLambda
		info.FunctionName = lambdaFunctionName
		info.FunctionVersion = os.Getenv("AWS_LAMBDA_FUNCTION_VERSION")
		info.Region = os.Getenv("AWS_REGION")
		info.AccountID = getAccountIDFromContext()
		info.LogGroup = os.Getenv("AWS_LAMBDA_LOG_GROUP_NAME")
		info.LogStream = os.Getenv("AWS_LAMBDA_LOG_STREAM_NAME")
		info.Memory = os.Getenv("AWS_LAMBDA_FUNCTION_MEMORY_SIZE")
		return info
	}

	// Check for ECS/Fargate.
	if ecsMetadataURI := os.Getenv("ECS_CONTAINER_METADATA_URI_V4"); ecsMetadataURI != "" {
		info.IsAWSContainer = true
		fetchECSMetadata(info, ecsMetadataURI)
		return info
	}

	// Fallback to older ECS metadata.
	if ecsMetadataURI := os.Getenv("ECS_CONTAINER_METADATA_URI"); ecsMetadataURI != "" {
		info.IsAWSContainer = true
		fetchECSMetadata(info, ecsMetadataURI)
		return info
	}

	// Check for EKS (Kubernetes on AWS).
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		// Check if it's specifically EKS by looking for AWS-specific annotations.
		if isEKS() {
			info.IsAWSContainer = true
			info.Platform = AWSPlatformEKS
			info.PodName = os.Getenv("HOSTNAME")
			info.Namespace = os.Getenv("POD_NAMESPACE")
			info.NodeName = os.Getenv("NODE_NAME")
			info.ClusterName = getEKSClusterName()
			info.Region = getAWSRegion()
			info.AccountID = getAccountIDFromIAM()
			return info
		}
	}

	// Check for AWS App Runner.
	if appRunnerServiceName := os.Getenv("APPRUNNER_SERVICE_NAME"); appRunnerServiceName != "" {
		info.IsAWSContainer = true
		info.Platform = AWSPlatformAppRunner
		info.ServiceName = appRunnerServiceName
		info.Region = os.Getenv("AWS_REGION")
		info.CPU = os.Getenv("APPRUNNER_CPU")
		info.Memory = os.Getenv("APPRUNNER_MEMORY")
		return info
	}

	// Check for AWS Batch.
	if batchJobID := os.Getenv("AWS_BATCH_JOB_ID"); batchJobID != "" {
		info.IsAWSContainer = true
		info.Platform = AWSPlatformBatch
		info.TaskID = batchJobID
		info.TaskARN = os.Getenv("AWS_BATCH_JOB_ATTEMPT")
		info.ClusterName = os.Getenv("AWS_BATCH_COMPUTE_ENVIRONMENT_NAME")
		info.Region = os.Getenv("AWS_DEFAULT_REGION")
		return info
	}

	// Check if running on EC2 (general).
	if isEC2() {
		info.IsAWSContainer = true
		info.Platform = AWSPlatformEC2
		info.Region = getEC2Region()
		info.AvailabilityZone = getEC2AvailabilityZone()
		return info
	}

	// Check for Lightsail containers.
	if lightsailServiceName := os.Getenv("LIGHTSAIL_SERVICE_NAME"); lightsailServiceName != "" {
		info.IsAWSContainer = true
		info.Platform = AWSPlatformLightsail
		info.ServiceName = lightsailServiceName
		info.Region = os.Getenv("AWS_REGION")
		return info
	}

	return info
}

// fetchECSMetadata fetches metadata from ECS metadata endpoint.
func fetchECSMetadata(info *AWSContainerInfo, metadataURI string) {
	// Set platform.
	if os.Getenv("AWS_EXECUTION_ENV") == "AWS_ECS_FARGATE" {
		info.Platform = AWSPlatformFargate
		info.LaunchType = "FARGATE"
	} else {
		info.Platform = AWSPlatformECS
		info.LaunchType = "EC2"
	}

	// Basic info from environment.
	info.ContainerName = os.Getenv("CONTAINER_NAME")
	info.TaskARN = os.Getenv("ECS_TASK_ARN")
	info.ClusterName = os.Getenv("ECS_CLUSTER")
	info.ServiceName = os.Getenv("ECS_SERVICE_NAME")
	info.Region = os.Getenv("AWS_REGION")
	info.LogGroup = os.Getenv("AWS_LOG_GROUP")
	info.LogStream = os.Getenv("AWS_LOG_STREAM")

	// Fetch detailed metadata from endpoint.
	client := &http.Client{Timeout: 2 * time.Second}

	// Get task metadata
	if taskResp, err := client.Get(metadataURI + "/task"); err == nil {
		defer taskResp.Body.Close()
		if body, err := io.ReadAll(taskResp.Body); err == nil {
			var metadata ECSTaskMetadata
			if json.Unmarshal(body, &metadata) == nil {
				info.TaskARN = metadata.TaskARN
				info.TaskFamily = metadata.Family
				info.TaskRevision = metadata.Revision
				info.ClusterName = metadata.Cluster
				info.AvailabilityZone = metadata.AvailabilityZone
				info.LaunchType = metadata.LaunchType

				// Extract task ID from ARN.
				if parts := strings.Split(metadata.TaskARN, "/"); len(parts) > 0 {
					info.TaskID = parts[len(parts)-1]
				}

				// Get container specific info.
				for _, container := range metadata.Containers {
					if container.Name == info.ContainerName {
						info.ContainerID = container.ContainerID
						info.ImageURI = container.Image
						info.LogGroup = container.LogOptions.LogGroup
						info.LogStream = container.LogOptions.LogStream
						break
					}
				}
			}
		}
	}
}

// isEKS checks if running in EKS.
func isEKS() bool {
	// Check for EKS-specific environment variables or files.
	if _, err := os.Stat("/var/run/secrets/eks.amazonaws.com/serviceaccount/token"); err == nil {
		return true
	}

	// Check for AWS-specific node labels (requires API access).
	nodeName := os.Getenv("NODE_NAME")
	return nodeName != "" && strings.Contains(nodeName, ".compute.amazonaws.com")
}

// getEKSClusterName attempts to get EKS cluster name.
func getEKSClusterName() string {
	// Try environment variable first.
	if clusterName := os.Getenv("EKS_CLUSTER_NAME"); clusterName != "" {
		return clusterName
	}

	// Try to parse from node name or other sources.
	if nodeName := os.Getenv("NODE_NAME"); nodeName != "" {
		// Extract from node name if it follows a pattern.
		parts := strings.Split(nodeName, ".")
		if len(parts) > 1 {
			return parts[0]
		}
	}

	return ""
}

// isEC2 checks if running on EC2.
func isEC2() bool {
	// Check EC2 metadata service
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("http://169.254.169.254/latest/meta-data/instance-id")
	if err == nil {
		resp.Body.Close()
		return resp.StatusCode == 200
	}
	return false
}

// getEC2Region gets region from EC2 metadata.
func getEC2Region() string {
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region
	}

	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("http://169.254.169.254/latest/meta-data/placement/region")
	if err == nil {
		defer resp.Body.Close()
		if body, err := io.ReadAll(resp.Body); err == nil {
			return string(body)
		}
	}
	return ""
}

// getEC2AvailabilityZone gets AZ from EC2 metadata.
func getEC2AvailabilityZone() string {
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("http://169.254.169.254/latest/meta-data/placement/availability-zone")
	if err == nil {
		defer resp.Body.Close()
		if body, err := io.ReadAll(resp.Body); err == nil {
			return string(body)
		}
	}
	return ""
}

// getAWSRegion gets AWS region from various sources.
func getAWSRegion() string {
	// Check common environment variables.
	for _, key := range []string{"AWS_REGION", "AWS_DEFAULT_REGION"} {
		if region := os.Getenv(key); region != "" {
			return region
		}
	}

	// Try EC2 metadata
	return getEC2Region()
}

// getAccountIDFromContext extracts account ID from Lambda context.
func getAccountIDFromContext() string {
	if arn := os.Getenv("AWS_LAMBDA_FUNCTION_ARN"); arn != "" {
		// ARN format: arn:aws:lambda:region:account-id:function:function-name.
		parts := strings.Split(arn, ":")
		if len(parts) >= 5 {
			return parts[4]
		}
	}
	return ""
}

// getAccountIDFromIAM gets account ID from IAM role.
func getAccountIDFromIAM() string {
	// This would require AWS SDK or IAM metadata endpoint.
	// Simplified version.
	if roleARN := os.Getenv("AWS_ROLE_ARN"); roleARN != "" {
		parts := strings.Split(roleARN, ":")
		if len(parts) >= 5 {
			return parts[4]
		}
	}
	return ""
}

// FormatContainerInfo formats container info for logging.
func (c *AWSContainerInfo) FormatContainerInfo() string {
	if !c.IsAWSContainer {
		return "Not running in AWS Container"
	}

	switch c.Platform {
	case AWSPlatformECS, AWSPlatformFargate:
		return fmt.Sprintf("'%s' [Platform: '%s', Task: '%s', Container: '%s', Cluster: '%s']",
			c.ServiceName, c.Platform, c.TaskID, c.ContainerName, c.ClusterName)
	case AWSPlatformEKS:
		return fmt.Sprintf("'%s' [Platform: '%s', Pod: '%s', Namespace: '%s', Node: '%s']",
			c.ClusterName, c.Platform, c.PodName, c.Namespace, c.NodeName)
	case AWSPlatformLambda:
		return fmt.Sprintf("'%s' [Platform: '%s', Version: '%s', Region: '%s']",
			c.FunctionName, c.Platform, c.FunctionVersion, c.Region)
	default:
		return fmt.Sprintf("[Platform: '%s', Region: '%s']", c.Platform, c.Region)
	}
}

// GetAWSEnvironmentVariables returns relevant AWS environment variables.
func GetAWSEnvironmentVariables() map[string]string {
	vars := make(map[string]string)

	awsVars := []string{
		// ECS/Fargate
		"ECS_CONTAINER_METADATA_URI_V4",
		"ECS_CONTAINER_METADATA_URI",
		"ECS_TASK_ARN",
		"ECS_CLUSTER",
		"ECS_SERVICE_NAME",
		"CONTAINER_NAME",
		"AWS_EXECUTION_ENV",

		// Lambda
		"AWS_LAMBDA_FUNCTION_NAME",
		"AWS_LAMBDA_FUNCTION_VERSION",
		"AWS_LAMBDA_FUNCTION_ARN",
		"AWS_LAMBDA_LOG_GROUP_NAME",
		"AWS_LAMBDA_LOG_STREAM_NAME",
		"AWS_LAMBDA_FUNCTION_MEMORY_SIZE",

		// EKS
		"KUBERNETES_SERVICE_HOST",
		"POD_NAME",
		"POD_NAMESPACE",
		"NODE_NAME",
		"EKS_CLUSTER_NAME",

		// App Runner
		"APPRUNNER_SERVICE_NAME",
		"APPRUNNER_CPU",
		"APPRUNNER_MEMORY",

		// Batch
		"AWS_BATCH_JOB_ID",
		"AWS_BATCH_JOB_ATTEMPT",
		"AWS_BATCH_COMPUTE_ENVIRONMENT_NAME",

		// General AWS
		"AWS_REGION",
		"AWS_DEFAULT_REGION",
		"AWS_ACCOUNT_ID",
		"AWS_ROLE_ARN",
		"HOSTNAME",
	}

	for _, key := range awsVars {
		if value := os.Getenv(key); value != "" {
			vars[key] = value
		}
	}

	return vars
}

// DetectAzureContainer detects and returns Azure container information.
func DetectAzureContainer() *AzureContainerInfo {
	info := &AzureContainerInfo{
		Platform: AzurePlatformUnknown,
	}

	// Check for Azure Container Apps.
	if containerAppName := os.Getenv("CONTAINER_APP_NAME"); containerAppName != "" {
		info.IsAzureContainer = true
		info.Platform = AzurePlatformContainerApps
		info.ContainerAppName = containerAppName
		info.AppName = containerAppName
		info.Revision = os.Getenv("CONTAINER_APP_REVISION")
		info.Replica = os.Getenv("CONTAINER_APP_REPLICA_NAME")
		info.Environment = os.Getenv("CONTAINER_APP_ENV_NAME")

		// Try to parse revision and replica from hostname if not set.
		if info.Revision == "" || info.Replica == "" {
			parseAzureContainerAppInfo(info)
		}
		return info
	}

	// Alternative check - Azure Container Apps also sets these.
	if os.Getenv("CONTAINER_APP_ENV_NAME") != "" {
		info.IsAzureContainer = true
		info.Platform = AzurePlatformContainerApps
	}

	// Check for Azure App Service.
	if siteName := os.Getenv("WEBSITE_SITE_NAME"); siteName != "" {
		info.IsAzureContainer = true
		info.Platform = AzurePlatformAppService
		info.AppName = siteName
		info.Revision = os.Getenv("WEBSITE_DEPLOYMENT_ID")
		info.Replica = os.Getenv("WEBSITE_INSTANCE_ID")
		info.Environment = os.Getenv("WEBSITE_SLOT_NAME")
		return info
	}

	// Check for Azure Container Instances.
	if os.Getenv("ACI_INSTANCE") == "true" {
		info.IsAzureContainer = true
		info.Platform = AzurePlatformContainerInstance
		info.AppName = os.Getenv("ACI_NAME")
		info.ResourceGroup = os.Getenv("ACI_RESOURCE_GROUP")
		return info
	}

	// Check for AKS (Azure Kubernetes Service).
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" && os.Getenv("AKS_CLUSTER_NAME") != "" {
		info.IsAzureContainer = true
		info.Platform = AzurePlatformKubernetes
		info.AppName = os.Getenv("AKS_CLUSTER_NAME")
		info.PodName = os.Getenv("HOSTNAME")
		info.Namespace = os.Getenv("POD_NAMESPACE")
		return info
	}

	// Additional Azure info.
	info.Region = os.Getenv("REGION")
	if info.Region == "" {
		info.Region = os.Getenv("AZURE_REGION")
	}
	info.ResourceGroup = os.Getenv("RESOURCE_GROUP")
	info.SubscriptionID = os.Getenv("SUBSCRIPTION_ID")

	// Additional generic Azure checks.
	if info.Region != "" || info.SubscriptionID != "" {
		info.IsAzureContainer = true
	}

	return info
}

// parseAzureContainerAppInfo attempts to parse revision and replica from hostname.
func parseAzureContainerAppInfo(info *AzureContainerInfo) {
	// Azure Container Apps hostname format:
	// app-name--revision-hash-pod
	// Example: iris-go-prod--0000014-5d7df89b7c-9qfxn

	hostname := getHostname()
	if hostname == "" {
		return
	}

	// First, try to extract the app name and revision.
	parts := strings.Split(hostname, "--")
	if len(parts) >= 2 {
		// The replica is the full hostname.
		info.Replica = hostname

		// Extract revision.
		revisionParts := strings.Split(parts[1], "-")
		if len(revisionParts) > 0 {
			info.Revision = parts[0] + "--" + revisionParts[0]
		}
	}
}

// FormatContainerInfo formats container info similar to Azure logs.
func (c *AzureContainerInfo) FormatContainerInfo() string {
	if !c.IsAzureContainer {
		return "Not running in Azure Container"
	}

	return fmt.Sprintf("'%s' [Revision: '%s', Replica: '%s']",
		c.AppName, c.Revision, c.Replica)
}

// GetAzureEnvironmentVariables returns a map of relevant Azure environment variables.
func GetAzureEnvironmentVariables() map[string]string {
	vars := make(map[string]string)

	// List of Azure-related environment variables to check.
	azureVars := []string{
		"CONTAINER_APP_NAME",
		"CONTAINER_APP_REVISION",
		"CONTAINER_APP_REPLICA_NAME",
		"CONTAINER_APP_ENV_NAME",
		"WEBSITE_SITE_NAME",
		"WEBSITE_DEPLOYMENT_ID",
		"WEBSITE_INSTANCE_ID",
		"AZURE_REGION",
		"REGION",
		"RESOURCE_GROUP",
		"SUBSCRIPTION_ID",
		"AKS_CLUSTER_NAME",
		"POD_NAME",
		"POD_NAMESPACE",
		"HOSTNAME",
	}

	for _, key := range azureVars {
		if value := os.Getenv(key); value != "" {
			vars[key] = value
		}
	}

	return vars
}

// DetectGCPContainer detects and returns GCP container information
func DetectGCPContainer() *GCPContainerInfo {
	info := &GCPContainerInfo{
		Platform: GCPPlatformUnknown,
		Hostname: getHostname(),
	}

	// Check for Cloud Run.
	if service := os.Getenv("K_SERVICE"); service != "" {
		info.IsGCPContainer = true
		info.Platform = GCPPlatformCloudRun
		info.ServiceName = service
		info.ServiceRevision = os.Getenv("K_REVISION")
		info.ServiceVersion = os.Getenv("K_CONFIGURATION")
		info.Region = os.Getenv("CLOUD_RUN_REGION")
		info.ServiceAccount = os.Getenv("K_SERVICE_ACCOUNT_EMAIL")
		info.ProjectID = getGCPProjectID()
		info.NumericProjectID = os.Getenv("GOOGLE_CLOUD_PROJECT_NUMBER")

		// Cloud Run specific
		info.InstanceID = os.Getenv("INSTANCE_ID")
		info.LogName = fmt.Sprintf("projects/%s/logs/run.googleapis.com%%2Frequests", info.ProjectID)

		return info
	}

	// Check for Cloud Functions.
	if functionName := os.Getenv("FUNCTION_NAME"); functionName != "" {
		info.IsGCPContainer = true
		info.Platform = GCPPlatformCloudFunction
		info.FunctionName = functionName
		info.FunctionRegion = os.Getenv("FUNCTION_REGION")
		info.FunctionEntryPoint = os.Getenv("ENTRY_POINT")
		info.ServiceVersion = os.Getenv("X_GOOGLE_FUNCTION_VERSION")
		info.ProjectID = getGCPProjectID()
		info.ServiceAccount = os.Getenv("FUNCTION_IDENTITY")

		// Function execution info.
		info.InstanceID = os.Getenv("INSTANCE_ID")
		info.TraceID = os.Getenv("FUNCTION_REQUEST_ID")

		return info
	}

	// Check for App Engine.
	if gaeService := os.Getenv("GAE_SERVICE"); gaeService != "" {
		info.IsGCPContainer = true
		info.Platform = GCPPlatformAppEngine
		info.ServiceName = gaeService
		info.ServiceVersion = os.Getenv("GAE_VERSION")
		info.InstanceID = os.Getenv("GAE_INSTANCE")
		info.ProjectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
		info.Region = os.Getenv("GAE_REGION")
		info.ServiceAccount = os.Getenv("GAE_SERVICE_ACCOUNT")

		// App Engine specific
		info.Zone = os.Getenv("GAE_ZONE")
		info.LogName = fmt.Sprintf("projects/%s/logs/appengine.googleapis.com%%2Frequest_log", info.ProjectID)

		return info
	}

	// Check for GKE (Google Kubernetes Engine).
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" && isGKE() {
		info.IsGCPContainer = true
		info.Platform = GCPPlatformGKE
		info.PodName = os.Getenv("HOSTNAME")
		info.Namespace = os.Getenv("POD_NAMESPACE")
		info.NodeName = os.Getenv("NODE_NAME")
		info.ContainerName = os.Getenv("CONTAINER_NAME")
		info.ClusterName = getGKEClusterName()
		info.ProjectID = getGCPProjectID()

		// GKE specific metadata.
		fetchGKEMetadata(info)

		return info
	}

	// Check for Compute Engine.
	if isComputeEngine() {
		info.IsGCPContainer = true
		info.Platform = GCPPlatformComputeEngine
		fetchComputeEngineMetadata(info)
		return info
	}

	// Additional GCP detection.
	if projectID := getGCPProjectID(); projectID != "" {
		info.IsGCPContainer = true
		info.ProjectID = projectID
	}

	return info
}

// isGKE checks if running in GKE.
func isGKE() bool {
	// Check for GKE-specific environment variables
	if os.Getenv("GOOGLE_CLOUD_PROJECT") != "" {
		return true
	}

	// Check for GKE metadata endpoint
	if _, err := getGCPMetadataValue("instance/attributes/cluster-name"); err == nil {
		return true
	}

	// Check node name pattern
	nodeName := os.Getenv("NODE_NAME")
	return nodeName != "" && strings.Contains(nodeName, "gke-")
}

// getGKEClusterName attempts to get GKE cluster name.
func getGKEClusterName() string {
	// Try environment variable first
	if clusterName := os.Getenv("GKE_CLUSTER_NAME"); clusterName != "" {
		return clusterName
	}

	// Try metadata server
	if clusterName, err := getGCPMetadataValue("instance/attributes/cluster-name"); err == nil {
		return clusterName
	}

	// Parse from node name
	if nodeName := os.Getenv("NODE_NAME"); nodeName != "" {
		// GKE node names follow pattern: gke-CLUSTER_NAME-...
		if strings.HasPrefix(nodeName, "gke-") {
			parts := strings.Split(nodeName, "-")
			if len(parts) >= 2 {
				// Reconstruct cluster name (it might contain hyphens)
				for i := 1; i < len(parts)-2; i++ {
					if parts[i] == "default" || parts[i] == "pool" {
						return strings.Join(parts[1:i], "-")
					}
				}
			}
		}
	}

	return ""
}

// fetchGKEMetadata fetches additional GKE metadata.
func fetchGKEMetadata(info *GCPContainerInfo) {
	// Get cluster location
	if location, err := getGCPMetadataValue("instance/attributes/cluster-location"); err == nil {
		if before, _, ok := strings.CutLast(location, "-"); ok {
			info.Region = before
			info.Zone = location
		} else {
			info.Region = location
		}
	}

	// Get instance details.
	if instanceID, err := getGCPMetadataValue("instance/id"); err == nil {
		info.InstanceID = instanceID
	}

	if instanceName, err := getGCPMetadataValue("instance/name"); err == nil {
		info.InstanceName = instanceName
	}
}

// isComputeEngine checks if running on Compute Engine.
func isComputeEngine() bool {
	// Check GCP metadata server.
	_, err := getGCPMetadataValue("instance/id")
	return err == nil
}

// fetchComputeEngineMetadata fetches Compute Engine metadata.
func fetchComputeEngineMetadata(info *GCPContainerInfo) {
	// Get basic instance info
	if projectID, err := getGCPMetadataValue("project/project-id"); err == nil {
		info.ProjectID = projectID
	}

	if numericProjectID, err := getGCPMetadataValue("project/numeric-project-id"); err == nil {
		info.NumericProjectID = numericProjectID
	}

	if instanceID, err := getGCPMetadataValue("instance/id"); err == nil {
		info.InstanceID = instanceID
	}

	if instanceName, err := getGCPMetadataValue("instance/name"); err == nil {
		info.InstanceName = instanceName
	}

	if zone, err := getGCPMetadataValue("instance/zone"); err == nil {
		// Zone format: projects/PROJECT_NUMBER/zones/ZONE
		parts := strings.Split(zone, "/")
		if len(parts) > 0 {
			info.Zone = parts[len(parts)-1]
			// Extract region from zone (e.g., us-central1-a -> us-central1)
			if before, _, ok := strings.CutLast(info.Zone, "-"); ok {
				info.Region = before
			}
		}
	}

	if serviceAccount, err := getGCPMetadataValue("instance/service-accounts/default/email"); err == nil {
		info.ServiceAccount = serviceAccount
	}
}

// getGCPMetadataValue retrieves a value from GCP metadata server.
func getGCPMetadataValue(path string) (string, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/"+path, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("metadata request failed: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

// getGCPProjectID gets GCP project ID from various sources.
func getGCPProjectID() string {
	// Check environment variables in order of preference.
	for _, key := range []string{
		"GOOGLE_CLOUD_PROJECT",
		"GCP_PROJECT",
		"GCLOUD_PROJECT",
		"GOOGLE_PROJECT_ID",
	} {
		if projectID := os.Getenv(key); projectID != "" {
			return projectID
		}
	}

	// Try metadata server.
	if projectID, err := getGCPMetadataValue("project/project-id"); err == nil {
		return projectID
	}

	return ""
}

// FormatContainerInfo formats GCP container info for logging.
func (c *GCPContainerInfo) FormatContainerInfo() string {
	if !c.IsGCPContainer {
		return "Not running in GCP Container"
	}

	switch c.Platform {
	case GCPPlatformCloudRun:
		return fmt.Sprintf("'%s' [Platform: '%s', Revision: '%s', Region: '%s', Project: '%s']",
			c.ServiceName, c.Platform, c.ServiceRevision, c.Region, c.ProjectID)
	case GCPPlatformGKE:
		return fmt.Sprintf("'%s' [Platform: '%s', Pod: '%s', Namespace: '%s', Cluster: '%s']",
			c.ClusterName, c.Platform, c.PodName, c.Namespace, c.ClusterName)
	case GCPPlatformCloudFunction:
		return fmt.Sprintf("'%s' [Platform: '%s', Region: '%s', Project: '%s']",
			c.FunctionName, c.Platform, c.FunctionRegion, c.ProjectID)
	case GCPPlatformAppEngine:
		return fmt.Sprintf("'%s' [Platform: '%s', Version: '%s', Instance: '%s', Project: '%s']",
			c.ServiceName, c.Platform, c.ServiceVersion, c.InstanceID, c.ProjectID)
	case GCPPlatformComputeEngine:
		return fmt.Sprintf("'%s' [Platform: '%s', Zone: '%s', Project: '%s']",
			c.InstanceName, c.Platform, c.Zone, c.ProjectID)
	default:
		return fmt.Sprintf("[Platform: '%s', Project: '%s']", c.Platform, c.ProjectID)
	}
}

// GetGCPEnvironmentVariables returns relevant GCP environment variables.
func GetGCPEnvironmentVariables() map[string]string {
	vars := make(map[string]string)

	gcpVars := []string{
		// Cloud Run
		"K_SERVICE",
		"K_REVISION",
		"K_CONFIGURATION",
		"CLOUD_RUN_REGION",
		"K_SERVICE_ACCOUNT_EMAIL",

		// Cloud Functions
		"FUNCTION_NAME",
		"FUNCTION_REGION",
		"ENTRY_POINT",
		"X_GOOGLE_FUNCTION_VERSION",
		"FUNCTION_IDENTITY",
		"FUNCTION_REQUEST_ID",

		// App Engine
		"GAE_SERVICE",
		"GAE_VERSION",
		"GAE_INSTANCE",
		"GAE_REGION",
		"GAE_ZONE",
		"GAE_SERVICE_ACCOUNT",

		// GKE
		"GKE_CLUSTER_NAME",
		"CONTAINER_NAME",

		// General GCP
		"GOOGLE_CLOUD_PROJECT",
		"GCP_PROJECT",
		"GCLOUD_PROJECT",
		"GOOGLE_PROJECT_ID",
		"GOOGLE_CLOUD_PROJECT_NUMBER",

		// Kubernetes (for GKE)
		"KUBERNETES_SERVICE_HOST",
		"POD_NAME",
		"POD_NAMESPACE",
		"NODE_NAME",
		"HOSTNAME",
	}

	for _, key := range gcpVars {
		if value := os.Getenv(key); value != "" {
			vars[key] = value
		}
	}

	return vars
}

// DetectCloudContainer detects which cloud provider and container platform is being used.
func DetectCloudContainer() *ContainerInfo {
	info := &ContainerInfo{
		Provider: ProviderUnknown,
		Hostname: getHostname(),
	}

	// Check AWS
	awsInfo := DetectAWSContainer()
	if awsInfo.IsAWSContainer {
		info.Provider = ProviderAWS
		info.Platform = string(awsInfo.Platform)
		info.IsContainer = true
		info.AWS = awsInfo

		// Map common fields
		switch awsInfo.Platform {
		case AWSPlatformECS, AWSPlatformFargate:
			info.AppName = awsInfo.ServiceName
			info.ContainerID = awsInfo.ContainerID
			info.ContainerName = awsInfo.ContainerName
			info.Revision = awsInfo.TaskRevision
			info.Replica = awsInfo.TaskID
			info.Cluster = awsInfo.ClusterName
		case AWSPlatformEKS:
			info.AppName = awsInfo.ClusterName
			info.ContainerName = awsInfo.PodName
			info.Replica = awsInfo.PodName
			info.Cluster = awsInfo.ClusterName
		case AWSPlatformLambda:
			info.AppName = awsInfo.FunctionName
			info.Revision = awsInfo.FunctionVersion
		}

		info.Region = awsInfo.Region
		info.AccountID = awsInfo.AccountID
		info.LogGroup = awsInfo.LogGroup
		info.LogStream = awsInfo.LogStream

		return info
	}

	// Check Azure
	azureInfo := DetectAzureContainer()
	if azureInfo.IsAzureContainer {
		info.Provider = ProviderAzure
		info.Platform = string(azureInfo.Platform)
		info.IsContainer = true
		info.Azure = azureInfo

		// Map common fields
		info.AppName = azureInfo.AppName
		info.Revision = azureInfo.Revision
		info.Replica = azureInfo.Replica
		info.Region = azureInfo.Region
		info.Environment = azureInfo.Environment
		info.AccountID = azureInfo.SubscriptionID

		return info
	}

	// Check GCP
	gcpInfo := DetectGCPContainer()
	if gcpInfo.IsGCPContainer {
		info.Provider = ProviderGCP
		info.Platform = string(gcpInfo.Platform)
		info.IsContainer = true
		info.GCP = gcpInfo

		// Map common fields
		switch gcpInfo.Platform {
		case GCPPlatformCloudRun:
			info.AppName = gcpInfo.ServiceName
			info.Revision = gcpInfo.ServiceRevision
			info.Replica = gcpInfo.InstanceID
		case GCPPlatformGKE:
			info.AppName = gcpInfo.ClusterName
			info.ContainerName = gcpInfo.PodName
			info.Replica = gcpInfo.PodName
			info.Cluster = gcpInfo.ClusterName
		case GCPPlatformCloudFunction:
			info.AppName = gcpInfo.FunctionName
			info.Revision = gcpInfo.ServiceVersion
		case GCPPlatformAppEngine:
			info.AppName = gcpInfo.ServiceName
			info.Revision = gcpInfo.ServiceVersion
			info.Replica = gcpInfo.InstanceID
		case GCPPlatformComputeEngine:
			info.AppName = gcpInfo.InstanceName
			info.Replica = gcpInfo.InstanceID
		}

		info.Region = gcpInfo.Region
		if info.Region == "" {
			info.Region = gcpInfo.Zone
		}
		info.ProjectID = gcpInfo.ProjectID
		info.AccountID = gcpInfo.ProjectID // In GCP, project ID serves as account identifier.

		return info
	}

	return info
}

// FormatInfo returns a formatted string of container information.
func (c *ContainerInfo) FormatInfo() string {
	if !c.IsContainer {
		return "Not running in a cloud container"
	}

	switch c.Provider {
	case ProviderAWS:
		if c.AWS != nil {
			return c.AWS.FormatContainerInfo()
		}
	case ProviderAzure:
		if c.Azure != nil {
			return c.Azure.FormatContainerInfo()
		}
	case ProviderGCP:
		if c.GCP != nil {
			return c.GCP.FormatContainerInfo()
		}
	}

	return fmt.Sprintf("[Provider: %s, Platform: %s] '%s'",
		c.Provider, c.Platform, c.AppName)
}

// GetAllEnvironmentVariables returns all cloud-related environment variables.
func GetAllEnvironmentVariables() map[string]map[string]string {
	return map[string]map[string]string{
		"AWS":   GetAWSEnvironmentVariables(),
		"Azure": GetAzureEnvironmentVariables(),
		"GCP":   GetGCPEnvironmentVariables(),
	}
}

// LogCloudDetection logs the detected cloud container environment using the provided logger.
func LogCloudDetection(logger *slog.Logger) {
	logCloudInfo(logger, DetectCloudContainer())
}

// logCloudInfo logs info with logger. It is separate from LogCloudDetection so
// tests can supply a detection result.
func logCloudInfo(logger *slog.Logger, info *ContainerInfo) {
	if info.IsContainer {
		logger.LogAttrs(context.Background(), slog.LevelInfo, "Running in a cloud container",
			slog.String("cloud_provider", string(info.Provider)),
			slog.String("platform", info.Platform),
		)

		// Add to structured logging context.
		fields := map[string]any{
			"cloud_provider": string(info.Provider),
			"platform":       info.Platform,
			"app_name":       info.AppName,
			"revision":       info.Revision,
			"replica":        info.Replica,
			"region":         info.Region,
			"cluster":        info.Cluster,
		}

		// Add provider-specific fields.
		switch info.Provider {
		case ProviderAWS:
			fields["aws_account_id"] = info.AccountID
		case ProviderAzure:
			fields["azure_subscription_id"] = info.AccountID
		case ProviderGCP:
			fields["gcp_project_id"] = info.ProjectID
		}

		logger.LogAttrs(context.Background(), slog.LevelInfo, "Cloud container environment detected",
			slog.Any("container_info", fields),
		)

	} else {
		logger.Info("Not running in a cloud container environment")
	}
}
