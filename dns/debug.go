package dns

import D "github.com/miekg/dns"

func DebugDnsClient(client []dnsClient) []string {
	ips := []string{}
	for _, c := range client {
		ips = append(ips, c.Address())
	}
	return ips
}
func DebugDnsMessageQueryName(m *D.Msg) []string {
	names := []string{}
	for _, name := range m.Question {
		names = append(names, name.Name)
	}
	return names
}
