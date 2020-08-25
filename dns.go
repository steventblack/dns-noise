//
// Copyright 2020 Steven T Black
//

package main

import (
	"fmt"
	"github.com/miekg/dns"
	"log"
	"net"
	"reflect"
)

// dnsServers contain the address(es) of the DNS servers to query.
// The servers specified may be different than the local DNS servers (e.g. piholes).
var dnsServers []string

// dnsServerConfig sets the IP addresses and port for the set of DNS servers to be queried.
// If a Nameserver struct is provide and valid, the configuration will reflect those settings.
// If a Nameserver struct is omitted or invalid, it will attempt to establish the configuration based on the system default as defined in /etc/resolv.conf.
func dnsServerConfig(ns []NameServer) {
	var servers []string
	servers, err := dnsStatedClientConfig(ns)
	if err != nil {
		log.Print(err.Error())
		servers, err = dnsDefaultClientConfig()
		if err != nil {
			log.Fatal("Unable to establish DNS server configuration")
		}
	}

	dnsServers = servers
}

// dnsStatedClientConfig sets the IP addresses and port for the set of DNS servers to be queried based on the information in the Nameserver passed in.
// If successful, it returns the set of host/port strings used for DNS client queries or an empty set and error.
// The query strings are appended in the order defined in the Nameserver struct.
func dnsStatedClientConfig(ns []NameServer) ([]string, error) {
	if ns == nil {
		return nil, fmt.Errorf("No configuration data for nameserver")
	}

	var servers []string
	for _, nsentry := range ns {
		ip := net.ParseIP(nsentry.Ip)
		if ip == nil {
			log.Printf("Invalid IP address in nameserver configuration: %v", nsentry.Ip)
			continue
		}

		// if port not set, default to the standard port for DNS: 53
		if nsentry.Port == 0 {
			nsentry.Port = 53
		}

		// if dealing w/ an IPv6 address, need to wrap it in '[]' in order to protect the ':port' suffix from causing a parsing error.
		var hostport string
		if ip.To4() == nil {
			if nsentry.Zone == "" {
				hostport = fmt.Sprintf("[%s]:%d", ip.String(), nsentry.Port)
			} else {
				hostport = fmt.Sprintf("[%s%%%s]:%d", ip.String(), nsentry.Zone, nsentry.Port)
			}
		} else {
			hostport = fmt.Sprintf("%s:%d", ip.String(), nsentry.Port)
		}

		log.Printf("configured hostport: '%s'", hostport)

		servers = append(servers, hostport)
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("No valid IP addresses found in nameserver configuration")
	}

	return servers, nil
}

// dnsDefaultClientConfig attempts to read the /etc/resolv.conf file and use it for DNS configuration.
// It utilizes the nameserver entries and the default port (53) to generate the host/port combination for DNS queries.
// If successful, it returns the set of host/port strings used for DNS client queries or an empty set and error.
// The query strings are appended in the order defined in the resolv.conf file.
func dnsDefaultClientConfig() ([]string, error) {
	conf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		log.Print(err.Error())
		return nil, err
	}

	var servers []string
	for _, host := range conf.Servers {
		hostport := host + ":" + conf.Port
		servers = append(servers, hostport)
	}

	return servers, nil
}

// dnsLookup performs a dns query for the domain and type specified.
// Supported lookup types include 'A', 'AAAA', 'CNAME', and 'MX'.
// Unrecognized or unhandled lookup types will be defaulted to a 'A' lookup.
func dnsLookup(domain, msgType string) {
	t := dns.StringToType[msgType]
	switch t {
	case dns.TypeA, dns.TypeAAAA, dns.TypeCNAME, dns.TypeMX:
		break
	default:
		log.Printf("Unexpected query type (%v); defaulting to 'A'", msgType)
		t = dns.TypeA
	}

	q := new(dns.Msg)
	q.SetQuestion(dns.Fqdn(domain), t)

	// try each dns server if a connection error is encountered
	// server response codes (e.g. NXDOMAIN) are *not* considered errors
	for _, d := range dnsServers {
		_, err := dnsQuery(q, d)
		if err != nil {
			log.Print(err.Error())
			continue
		}
		break
	}
}

// dnsQuery performs the query against the designated DNS server.
// If successful, it returns the response containing the appropriate resource records.
// If the server is unable to resolve the query, it returns the appropriate resource records for the failure.
// If there is a problem querying the server, nil is returned with a descriptive error.
// Note that this supports only a single query per server request.
func dnsQuery(q *dns.Msg, d string) (*dns.Msg, error) {
	r, err := dns.Exchange(q, d)
	if err != nil {
		return nil, err
	}

	// assumes single query message
	if r.Rcode != dns.RcodeSuccess {
		log.Printf("%v: %v; %v", dns.TypeToString[r.Question[0].Qtype], r.Question[0].Name, dns.RcodeToString[r.Rcode])
		return r, nil
	}

	for _, a := range r.Answer {
		switch a.(type) {
		case *dns.A:
			rr := a.(*dns.A)
			log.Printf("%v: %v; %v", dns.TypeToString[rr.Header().Rrtype], rr.A, dns.RcodeToString[r.Rcode])
		case *dns.AAAA:
			rr := a.(*dns.AAAA)
			log.Printf("%v: %v; %v", dns.TypeToString[rr.Header().Rrtype], rr.AAAA, dns.RcodeToString[r.Rcode])
		case *dns.CNAME:
			rr := a.(*dns.CNAME)
			log.Printf("%v: %v; %v", dns.TypeToString[rr.Header().Rrtype], rr.Target, dns.RcodeToString[r.Rcode])
		case *dns.MX:
			rr := a.(*dns.MX)
			log.Printf("%v: %v; %v", dns.TypeToString[rr.Header().Rrtype], rr.Mx, dns.RcodeToString[r.Rcode])
		default:
			log.Printf("%v: Unexpected answer type", reflect.TypeOf(a))
		}
	}

	return r, nil
}
