package znet

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	"github.com/rackspace/gophercloud/openstack/networking/v2/ports"
	"github.com/rackspace/gophercloud/pagination"
)

type Vps struct {
	ID      string
	NameTag string
	// ExternalIPv4Address net.IP
	// ExternalIPv6Address net.IP
	// ExternalPort        AttachedPort
	Ports          []AttachedPort
	SecurityGroups []secgroups.SecurityGroup
}

type AttachedPort struct {
	PortId    string     `mapstructure:"port_id" json:"port_id"`
	PortState string     `mapstructure:"port_state" json:"port_state"`
	FixedIPs  []ports.IP `mapstructure:"fixed_ips" json:"fixed_ips"`
}

func (v *Vps) FromServer(s servers.Server) error {
	nametag, ok := s.Metadata["instance_name_tag"]
	if !ok {
		return fmt.Errorf("Attribute not found. [%s]", "instance_name_tag")
	}

	v.ID = s.ID
	v.NameTag = nametag.(string)
	if err := mapstructure.Decode(s.SecurityGroups, &v.SecurityGroups); err != nil {
		return err
	}

	for name, a := range s.Addresses {
		if !strings.HasPrefix(name, "ext-") && // ConoHa
			!strings.HasPrefix(name, "p-global") { // Z.com shared global
			continue
		}

		fixedIps := make([]ports.IP, 0, len(s.Addresses))
		addrs := a.([]interface{})
		for _, iaddr := range addrs {
			addr, ok := iaddr.(map[string]interface{})
			if !ok {
				return fmt.Errorf("Can't convert to map[string]interface{}. [%v]", iaddr)
			}

			// version, ok := addr["version"]
			// if !ok {
			// 	return fmt.Errorf(`Not has "version" field. [%v]`, addr)
			// }

			straddr, ok := addr["addr"].(string)
			if !ok {
				return fmt.Errorf(`Not has "addr" field. [%v]`, addr)
			}
			fixedIps = append(fixedIps, ports.IP{IPAddress: straddr})
		}

		port := AttachedPort{
			FixedIPs: fixedIps,
		}
		v.Ports = append(v.Ports, port)
	}

	return nil
}

// Set details of secutrity groups and ports
func (v *Vps) PopulateSecurityGroups(os *OpenStack) error {
	var err error

	// Security Groups
	result := servers.GetResult{}
	url := os.Compute.ServiceURL("servers", v.ID, "os-security-groups")
	_, err = os.Compute.Get(url, &result.Body, nil)
	if err != nil {
		return err
	}

	var resp struct {
		SecurityGroups []secgroups.SecurityGroup `mapstructure:"security_groups"`
	}

	if err = mapstructure.Decode(result.Body, &resp); err != nil {
		return err
	}
	v.SecurityGroups = resp.SecurityGroups

	return nil
}

func (v *Vps) PopulatePorts(os *OpenStack) error {
	var err error

	result := servers.GetResult{}
	url := os.Compute.ServiceURL("servers", v.ID, "os-interface")
	_, err = os.Compute.Get(url, &result.Body, nil)
	if err != nil {
		return err
	}

	var resp struct {
		Ports []AttachedPort `mapstructure:"interfaceAttachments" json:"ports"`
	}

	if err = mapstructure.Decode(result.Body, &resp); err != nil {
		return err
	}
	v.Ports = resp.Ports

	return nil
}

func (v *Vps) GetPort(queryIp string) *AttachedPort {
	for _, p := range v.Ports {
		for _, ip := range p.FixedIPs {
			if queryIp == ip.IPAddress {
				return &p
			}
		}
	}
	return nil
}
func (v *Vps) HasIpAddress(queryIp string) bool {
	return v.GetPort(queryIp) != nil
}

// "version" argument is one of "all", "ipv4" and "ipv6
func (v *Vps) AllIpAddresses(version string) []string {
	addresses := make([]string, 0, 10)
	for _, p := range v.Ports {
		for _, ip := range p.FixedIPs {
			if version == "all" {
				addresses = append(addresses, ip.IPAddress)
			} else if version == "ipv6" && strings.Index(ip.IPAddress, ":") >= 0 {
				addresses = append(addresses, ip.IPAddress)
			} else if version == "ipv4" && strings.Index(ip.IPAddress, ":") < 0 {
				addresses = append(addresses, ip.IPAddress)
			}

		}
	}
	return addresses
}

func (v *Vps) String() string {
	return fmt.Sprintf("%s %s %s", v.ID, v.NameTag, strings.Join(v.AllIpAddresses("all"), ","))
}

func GetVps(os *OpenStack, query string) (*Vps, error) {
	query = strings.ToLower(query)

	condition := func(vps Vps) bool {
		if strings.ToLower(vps.ID) == query ||
			strings.ToLower(vps.NameTag) == query ||
			vps.HasIpAddress(query) {
			return true
		}
		return false
	}

	vpss, err := ListVps(os, condition)
	if err != nil {
		return nil, err
	} else if len(vpss) != 1 {
		return nil, nil
	} else {
		return &vpss[0], nil
	}
}

func ListVps(os *OpenStack, condition func(vps Vps) (match bool)) ([]Vps, error) {
	if condition == nil {
		condition = func(vps Vps) bool { return true }
	}

	var pager pagination.Pager

	opts := servers.ListOpts{}
	pager = servers.List(os.Compute, opts)

	vpss := make([]Vps, 0)
	err := pager.EachPage(func(pages pagination.Page) (bool, error) {
		ss, err := servers.ExtractServers(pages)
		if err != nil {
			return false, err
		}

		for _, s := range ss {
			vps := Vps{}
			if err := vps.FromServer(s); err != nil {
				return false, err
			}

			if condition(vps) {
				vpss = append(vpss, vps)
			}
		}
		return true, err
	})
	if err != nil {
		return nil, err
	}

	return vpss, nil
}
