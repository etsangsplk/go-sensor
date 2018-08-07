package lightstepx

const (
	// TOO: K8s team may not support default settings but have to use specific protocol
	// due to loadbalancing etc. So leave these here for now, until they deploy.
	DefaultTransportProtocol = "UseGRPC"
	DefaultURIScheme         = ""
)
