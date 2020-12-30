package ipamspec

const (
	CREATE = "Create"
	DELETE = "Delete"
)

type IPAMRequest struct {
	HostName  string
	CIDR      string
	Operation string
}

type IPAMResponse struct {
	Request IPAMRequest
	IPAddr  string
	Status  bool
}
