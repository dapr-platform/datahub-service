package authbridge

import "testing"

func TestNormalizeRedirectURI(t *testing.T) {
	cases := map[string]string{
		"https://sjdz.menghaikechuang.com/datahub/#/sso/callback": "https://sjdz.menghaikechuang.com/datahub/",
		"https://sjdz.menghaikechuang.com/datahub/":               "https://sjdz.menghaikechuang.com/datahub/",
		" https://a.com/x#y ":                                     "https://a.com/x",
	}
	for in, want := range cases {
		got := NormalizeRedirectURI(in)
		if got != want {
			t.Fatalf("NormalizeRedirectURI(%q)=%q want %q", in, got, want)
		}
	}
}

func TestBuildSaSign(t *testing.T) {
	params := map[string]string{
		"client":    "yqsjdz-client",
		"msgType":   "userinfo",
		"loginId":   "1000001",
		"timestamp": "1789029175834",
		"nonce":     "6e2f3ef65d5d4c90bf5a8adacdf4b512",
	}
	got := BuildSaSign(params, "SJDZ-YQfyZtAmDbYHTBaHPSx3GZeX7x2ip7ik")
	want := "69f9e0f7c79fabd233fbc754e8cfcf80"
	if got != want {
		t.Fatalf("sign=%s, want %s", got, want)
	}
}
