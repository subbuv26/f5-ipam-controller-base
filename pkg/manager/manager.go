package manager

import log "github.com/subbuv26/f5-ipam-controller/pkg/vlogger"

// Manager defines the interface that the IPAM system should implement
type Manager interface {
	// Creates an A record
	CreateARecord(name, ipAddr string) bool
	// Deletes an A record and releases the IP address
	DeleteARecord(name, ipAddr string)
	// Gets and reserves the next available IP address
	GetNextAddr(cidr string) string
	// Releases an IP address
	ReleaseAddr(ipAddr string)
}

const F5IPAMProvider = "f5ipam"

type Params struct {
	Provider string
	IPAMManagerParams
}

func NewManager(params Params) Manager {
	switch params.Provider {
	case F5IPAMProvider:
		f5IPAMParams := IPAMManagerParams{Range: params.Range}
		return NewIPAMManager(f5IPAMParams)
	default:
		log.Errorf("Unknown Provider: %v", params.Provider)
	}
	return nil
}
