package external

import (
	"testing"

	"github.com/jetsetilly/test7800/test"
)

func TestSN2_transformDataNormal(t *testing.T) {
	ext := SN2{}

	for i := range uint8(0xff) {
		v := ext.transformDataNormal(i)
		test.DemandEquality(t, v, i)
	}
}

func TestSN2_transformData320(t *testing.T) {
	ext := SN2{}

	var v uint8

	// identity
	v = ext.transformData320(0x00)
	test.DemandEquality(t, v, 0x00)
	v = ext.transformData320(0xff)
	test.DemandEquality(t, v, 0xff)

	// single bit
	v = ext.transformData320(0x01)
	test.DemandEquality(t, v, 0x80)
	v = ext.transformData320(0x02)
	test.DemandEquality(t, v, 0x40)
	v = ext.transformData320(0x04)
	test.DemandEquality(t, v, 0x20)
	v = ext.transformData320(0x08)
	test.DemandEquality(t, v, 0x10)
	v = ext.transformData320(0x08)
	test.DemandEquality(t, v, 0x10)
	v = ext.transformData320(0x10)
	test.DemandEquality(t, v, 0x08)
	v = ext.transformData320(0x20)
	test.DemandEquality(t, v, 0x04)
	v = ext.transformData320(0x40)
	test.DemandEquality(t, v, 0x02)
	v = ext.transformData320(0x80)
	test.DemandEquality(t, v, 0x01)

	// multiple bits
	v = ext.transformData320(0x03)
	test.DemandEquality(t, v, 0xc0)
	v = ext.transformData320(0x23)
	test.DemandEquality(t, v, 0xc4)
}

func TestSN2_transformAddressNormal(t *testing.T) {
	ext := SN2{}

	for i := range uint16(0xffff) {
		v := ext.transformAddressNormal(i)
		test.DemandEquality(t, v, i)
	}
}

func TestSN2_transformAddressReverse(t *testing.T) {
	ext := SN2{}

	var v uint16

	v = ext.transformAddressReverse(0xff00)
	test.DemandEquality(t, v, 0xffff)
	v = ext.transformAddressReverse(0xffff)
	test.DemandEquality(t, v, 0xff00)
	v = ext.transformAddressReverse(0xff01)
	test.DemandEquality(t, v, 0xfffe)
}
