package kerlong

import "testing"

func TestIPToUint32(t *testing.T) {
	for _, value := range []string{"127.0.0.1", "::ffff:127.0.0.1"} {
		got, err := IPToUint32(value)
		if err != nil || got != 0x7f000001 {
			t.Errorf("%s: got %x, %v", value, got, err)
		}
	}
	for _, value := range []string{"invalid", "256.1.1.1", "::1"} {
		if _, err := IPToUint32(value); err == nil {
			t.Errorf("accepted %s", value)
		}
	}
}
