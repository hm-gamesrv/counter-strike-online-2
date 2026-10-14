package kerlong

import (
	"encoding/binary"
	"errors"
	"net/netip"
)

// IPToUint32 converts an IPv4 address to its network-order integer.
func IPToUint32(value string) (uint32, error) {
	address, err := netip.ParseAddr(value)
	if err != nil || !address.Unmap().Is4() {
		return 0, errors.New("invalid IPv4 address")
	}
	bytes := address.Unmap().As4()
	return binary.BigEndian.Uint32(bytes[:]), nil
}
