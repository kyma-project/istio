package httpbin

import (
	"strings"
	"testing"
)

func TestGenerateManifestIPv4Default(t *testing.T) {
	t.Setenv("TEST_IP_FAMILY", "")
	b := NewBuilder().WithName("hb")
	m, err := b.generateManifest()
	if err != nil {
		t.Fatal(err)
	}
	s := string(m)
	if !strings.Contains(s, "ipFamilyPolicy: SingleStack") {
		t.Errorf("expected SingleStack, got:\n%s", s)
	}
	if !strings.Contains(s, "- IPv4") {
		t.Errorf("expected IPv4 in families, got:\n%s", s)
	}
	if strings.Contains(s, "- IPv6") {
		t.Errorf("did not expect IPv6 in ipv4 mode, got:\n%s", s)
	}
}

func TestGenerateManifestDualStack(t *testing.T) {
	t.Setenv("TEST_IP_FAMILY", "dualstack")
	b := NewBuilder().WithName("hb")
	m, err := b.generateManifest()
	if err != nil {
		t.Fatal(err)
	}
	s := string(m)
	if !strings.Contains(s, "ipFamilyPolicy: PreferDualStack") {
		t.Errorf("expected PreferDualStack, got:\n%s", s)
	}
	if !strings.Contains(s, "- IPv6") || !strings.Contains(s, "- IPv4") {
		t.Errorf("expected both families, got:\n%s", s)
	}
	if strings.Index(s, "- IPv6") > strings.Index(s, "- IPv4") {
		t.Error("expected IPv6 listed before IPv4 in dualstack mode")
	}
}
