package debugger

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jetsetilly/test7800/hardware"
)

type mappedAddress struct {
	address uint16
	area    hardware.Area
	idx     uint16

	// this struct doesn't contain the actual mapped address. it would be nice
	// to have for the purpose of user-feedback but it's not necessary. the area
	// and index fields are the real requirements
}

func (m debugger) parseAddress(address string) (mappedAddress, error) {
	var ma mappedAddress

	if strings.HasPrefix(address, "$") {
		address = fmt.Sprintf("0x%s", address[1:])
	}

	addr, err := strconv.ParseUint(address, 0, 16)
	if err != nil {
		return ma, fmt.Errorf("address is not valid: %s", address)
	}
	ma.address = uint16(addr)

	ma.idx, ma.area = m.console.Mem.MapAddress(ma.address, true)
	if ma.area == nil {
		return ma, fmt.Errorf("address is not mapped: %s", address)
	}

	return ma, nil
}
