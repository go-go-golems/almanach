package native

const (
	ProvisioningServiceUUID = "021a9004-0382-4aea-bff4-6b3f1c5adfb4"
)

var FallbackEndpointUUIDs = map[string]string{
	EndpointProvCtrl:    "021aff4f-0382-4aea-bff4-6b3f1c5adfb4",
	EndpointProvScan:    "021aff50-0382-4aea-bff4-6b3f1c5adfb4",
	EndpointProvSession: "021aff51-0382-4aea-bff4-6b3f1c5adfb4",
	EndpointProvConfig:  "021aff52-0382-4aea-bff4-6b3f1c5adfb4",
	EndpointProtoVer:    "021aff53-0382-4aea-bff4-6b3f1c5adfb4",
}
