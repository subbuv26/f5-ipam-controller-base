package provider

import (
	"fmt"
	"net"
	"strings"

	"github.com/subbuv26/f5-ipam-controller/pkg/provider/sqlite"
	log "github.com/subbuv26/f5-ipam-controller/pkg/vlogger"
)

type IPAMProvider struct {
	store *sqlite.DBStore
	cidrs map[string]bool
}

type Params struct {
	Range string
}

func NewProvider(params Params) *IPAMProvider {
	//ipArr := []string{"172.16.1.1-172.16.1.5/24", "172.16.1.50/22-172.16.1.55/22"}
	ipRanges := parseIPRange(params.Range)
	if ipRanges == nil {
		return nil
	}

	prov := &IPAMProvider{
		store: sqlite.NewStore(),
		cidrs: nil,
	}
	prov.generateExternalIPAddr(ipRanges)
	return prov

}

func parseIPRange(ipRange string) []string {
	if len(ipRange) == 0 {
		return nil
	}
	log.Debugf("Parsing IP Ranges: %v", ipRange)
	ranges := strings.Split(ipRange, ",")
	var ipRanges []string
	for _, ipRange := range ranges {
		ipRanges = append(ipRanges, strings.Trim(ipRange, " "))
	}
	return ipRanges
}

// generateExternalIPAddr ...
func (prov *IPAMProvider) generateExternalIPAddr(ipRnages []string) {
	var startRangeIP, endRangeIP, Subnet, ExternalIPType string
	if len(ipRnages) == 0 {
		log.Fatal("No IP range provided")
	}

	for _, ip := range ipRnages {
		log.Debugf("IP Range: %v", ip)
		ip = strings.Trim(ip, " ")
		ipRangeArr := strings.Split(ip, "-")

		//checking the cidr of both the IPS if same then proceed otherwise error log
		ipRangeStart := strings.Split(ipRangeArr[0], "/")
		ipRangeEnd := strings.Split(ipRangeArr[1], "/")

		if ipRangeStart[1] != ipRangeEnd[1] {
			log.Debugf("IPv4 Range Subnet mask is inconsistent")
			continue
		} else {
			ExternalIPType = ipv4or6(ip)
			log.Debugf("\nIP-Address is of Type  %v: ", ExternalIPType)
			Subnet = ipRangeStart[1]
		}

		startRangeIP = ipRangeStart[0]
		endRangeIP = ipRangeEnd[0]

		log.Debugf("IP Pool: %v to %v/%v", startRangeIP, endRangeIP, Subnet)

		//endip validation
		ipEnd, _, err := net.ParseCIDR(endRangeIP + "/" + Subnet)
		if err != nil {
			fmt.Print("Parsing err :  ", err)
			//return nil
		}

		//startip validation
		ipStart, ipnetStart, err := net.ParseCIDR(startRangeIP + "/" + Subnet)
		if err != nil {
			fmt.Print("Parsing err : ", err)
			//return nil
		}
		ips := []string{}
		for ; ipnetStart.Contains(ipStart); inc(ipStart) {
			ips = append(ips, ipStart.String())
			// if len(ips) == EXTERNAL_IP_RANGE_COUNT {
			// 	break
			// }
			if ipStart.String() == ipEnd.String() {
				break
			}
		}
		prov.store.InsertIP(ips, Subnet)
	}

	prov.store.DisplayIPRecords()

	//ipAllocated := store.AllocateIP()
	//fmt.Println("********** : ", ipAllocated)
	//prov.store.DisplayIPRecords()
	//
	//fmt.Println("Going in allocate ip")
	//store.ReleaseIP("172.16.1.1")
	//store.DisplayIPRecords()
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

//external-ip-address parameter is of type ipv4 or ipv6
func ipv4or6(s string) string {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.':
			return "IPv4"
		case ':':
			return "IPv6"
		}
	}
	return "Invalid Address"

}

// Creates an A record
func (prov *IPAMProvider) CreateARecord(hostname, ipAddr string) bool {
	prov.store.CreateARecord(hostname, ipAddr)
	log.Debugf("Created 'A' Record. Host:%v, IP:%v", hostname, ipAddr)
	return true
}

// Deletes an A record and releases the IP address
func (prov *IPAMProvider) DeleteARecord(hostname, ipAddr string) {
	prov.store.DeleteARecord(hostname, ipAddr)
	log.Debugf("Deleted 'A' Record. Host:%v, IP:%v", hostname, ipAddr)
}

func (prov *IPAMProvider) GetIPAddress(hostname string) string {
	return prov.store.GetIPAddress(hostname)
}

// Gets and reserves the next available IP address
func (prov *IPAMProvider) GetNextAddr(cidr string) string {
	if _, ok := prov.cidrs[cidr]; !ok {
		log.Debugf("Unsupported CIDR: %v", cidr)
		return ""
	}
	return prov.store.AllocateIP(cidr)
}

// Releases an IP address
func (prov *IPAMProvider) ReleaseAddr(ipAddr string) {
	prov.store.ReleaseIP(ipAddr)
}
