package user

import "testing"

func TestPasswordMD5IsFixedLowercase32Characters(t *testing.T) {
	got := passwordMD5("password")
	const want = "5f4dcc3b5aa765d61d8327deb882cf99"
	if got != want || len(got) != 32 {
		t.Fatalf("passwordMD5() = %q, want %q", got, want)
	}
	if !passwordMatches("password", want) {
		t.Fatal("passwordMatches() rejected lowercase MD5")
	}
	if passwordMatches("password", "5F4DCC3B5AA765D61D8327DEB882CF99") {
		t.Fatal("passwordMatches() accepted uppercase MD5")
	}
}
