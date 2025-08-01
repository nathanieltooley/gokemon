package networking

import "net"

func GetAllBroadcastAddrs() []net.IP {
	masks := make([]net.IPMask, 0)
	ips := make([]net.IP, 0)

	ints, _ := net.Interfaces()
	for _, ifs := range ints {
		addrs, _ := ifs.Addrs()
		for _, addr := range addrs {
			switch a := addr.(type) {
			case *net.IPNet:
				netAddr := a.IP.To4()

				if netAddr != nil {
					masks = append(masks, a.Mask)
					ips = append(ips, a.IP.To4())
				}
			case *net.IPAddr:
				defMask := a.IP.DefaultMask()
				if defMask != nil {
					masks = append(masks, defMask)
					ips = append(ips, a.IP.To4())
				}
			}
		}
	}

	broadIps := make([]net.IP, 0)
	for i, m := range masks {
		broadIp := net.ParseIP("0.0.0.0").To4()

		// Bitwise OR each byte of the IP addr with the Bitwise NOT of the mask byte
		for j := range 4 {
			broadIp[j] = ips[i][j] | ^m[j]
		}

		broadIps = append(broadIps, broadIp)
	}

	return broadIps
}
