package detect

import "net"

type Interface struct {
	Name      string   `json:"name"`
	Hardware  string   `json:"hardware,omitempty"`
	Addresses []string `json:"addresses,omitempty"`
	Up        bool     `json:"up"`
	Loopback  bool     `json:"loopback"`
}

func Interfaces() ([]Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	result := make([]Interface, 0, len(interfaces))
	for _, item := range interfaces {
		addresses, _ := item.Addrs()
		values := make([]string, 0, len(addresses))
		for _, address := range addresses {
			values = append(values, address.String())
		}
		result = append(result, Interface{
			Name:      item.Name,
			Hardware:  item.HardwareAddr.String(),
			Addresses: values,
			Up:        item.Flags&net.FlagUp != 0,
			Loopback:  item.Flags&net.FlagLoopback != 0,
		})
	}

	return result, nil
}

func PrimaryIP() string {
	interfaces, err := Interfaces()
	if err != nil {
		return ""
	}
	for _, item := range interfaces {
		if !item.Up || item.Loopback {
			continue
		}
		for _, address := range item.Addresses {
			ip, _, err := net.ParseCIDR(address)
			if err != nil || ip == nil || ip.To4() == nil {
				continue
			}
			return ip.String()
		}
	}
	return ""
}
