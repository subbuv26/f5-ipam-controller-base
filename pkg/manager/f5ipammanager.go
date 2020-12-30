package manager

import (
	"net"
	"strings"

	"github.com/subbuv26/f5-ipam-controller/pkg/provider"
	log "github.com/subbuv26/f5-ipam-controller/pkg/vlogger"
)

type IPAMManagerParams struct {
	Range string
}

type IPAMManager struct {
	provider *provider.IPAMProvider
}

func NewIPAMManager(params IPAMManagerParams) *IPAMManager {
	provParams := provider.Params{Range: params.Range}
	prov := provider.NewProvider(provParams)
	if prov == nil {
		log.Error("Unable to create Provider")
		return nil
	}
	return &IPAMManager{provider: prov}
}

// Creates an A record
func (ipMgr *IPAMManager) CreateARecord(name, ipAddr string) bool {
	if !isIPV4Addr(ipAddr) {
		log.Errorf("Invalid IP Address Provided")
		return false
	}
	// TODO: Validate name to be a proper dns name
	ipMgr.provider.CreateARecord(name, ipAddr)
	return true
}

// Deletes an A record and releases the IP address
func (ipMgr *IPAMManager) DeleteARecord(name, ipAddr string) {
	if !isIPV4Addr(ipAddr) {
		log.Errorf("Invalid IP Address Provided")
		return
	}
	// TODO: Validate name to be a proper dns name
	ipMgr.provider.DeleteARecord(name, ipAddr)
}

// Gets and reserves the next available IP address
func (ipMgr *IPAMManager) GetNextAddr(cidr string) string {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Debugf("Invalid CIDR Provided: %v", cidr)
		return ""
	}
	return ipMgr.provider.GetNextAddr(cidr)
}

// Releases an IP address
func (ipMgr *IPAMManager) ReleaseAddr(ipAddr string) {

	if !isIPV4Addr(ipAddr) {
		log.Errorf("Invalid IP Address Provided")
		return
	}
	ipMgr.provider.ReleaseAddr(ipAddr)
}

func isIPV4Addr(ipAddr string) bool {
	if net.ParseIP(ipAddr) == nil {
		return false
	}

	// presence of ":" indicates it is an IPV6
	if strings.Contains(ipAddr, ":") {
		return false
	}

	return true
}
