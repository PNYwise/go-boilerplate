package validation

import "testing"

func TestGetValidatorSingleton(t *testing.T) {
	v1 := GetValidator()
	v2 := GetValidator()
	if v1 == nil || v2 == nil || v1 != v2 {
		t.Fatalf("GetValidator should return a singleton instance")
	}
}
