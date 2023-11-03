package ip

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"log"
	"net"
)

// GetLoadBalancerIp returns the IP of the load balancer from the load balancer ingress object
func GetLoadBalancerIp(loadBalancerIngress v1.LoadBalancerIngress) (net.IP, error) {
	loadBalancerIP, err := getIpBasedLoadBalancerIp(loadBalancerIngress)

	if err == nil {
		return loadBalancerIP, nil
	} else {
		log.Printf("Falling back to reading DNS based load balancer IP, because of: %s\n", err)
		return getDnsBasedLoadBalancerIp(loadBalancerIngress)
	}
}

func getIpBasedLoadBalancerIp(lbIngress v1.LoadBalancerIngress) (net.IP, error) {
	ip := net.ParseIP(lbIngress.IP)
	if ip == nil {
		return nil, fmt.Errorf("failed to parse IP from load balancer IP %s", lbIngress.IP)
	}

	return ip, nil
}

func getDnsBasedLoadBalancerIp(lbIngress v1.LoadBalancerIngress) (net.IP, error) {
	ips, err := net.LookupIP(lbIngress.Hostname)
	if err != nil || len(ips) < 1 {
		return nil, fmt.Errorf("could not get IPs by load balancer hostname: %v", err)
	}

	return ips[0], nil
}
