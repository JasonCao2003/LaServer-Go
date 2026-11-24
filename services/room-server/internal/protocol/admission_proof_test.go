// 准入凭证解析测试

package protocol

import (
	"bytes"
	"testing"
)

func TestDecodeAdmissionProof(t *testing.T) {
	frame := joinFixture(284, 542, 170388, 0, 21)
	copy(frame[2:], []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff})
	copy(frame[18:50], bytes.Repeat([]byte{0xab}, 32))
	proof, err := DecodeAdmissionProof(frame)
	if err != nil {
		t.Fatal(err)
	}
	if proof.AccountID != "00112233-4455-6677-8899-aabbccddeeff" || proof.Signature != [32]byte(bytes.Repeat([]byte{0xab}, 32)) {
		t.Fatal("proof fields did not match the observed layout")
	}
	if _, err := DecodeAdmissionProof(joinFixture(284, 541, 169938, 0, 21)); err == nil {
		t.Fatal("used credential offsets on an unverified profile")
	}
}
