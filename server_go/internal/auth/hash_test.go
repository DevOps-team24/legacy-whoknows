package auth

import "testing"

func TestHashPassword_MatchesLegacyPython(t *testing.T) {
	// "password" -> MD5 from Python's hashlib.md5(b"password").hexdigest()
	// This is also the admin user hash in 001_init.sql.
	got := HashPassword("password")
	want := "5f4dcc3b5aa765d61d8327deb882cf99"
	if got != want {
		t.Errorf("HashPassword(%q) = %q, want %q", "password", got, want)
	}
}

func TestHashPassword_EmptyString(t *testing.T) {
	got := HashPassword("")
	want := "d41d8cd98f00b204e9800998ecf8427e"
	if got != want {
		t.Errorf("HashPassword(%q) = %q, want %q", "", got, want)
	}
}

func TestVerifyPassword_Correct(t *testing.T) {
	hash := HashPassword("mySecret123")
	if !VerifyPassword(hash, "mySecret123") {
		t.Error("VerifyPassword should return true for matching password")
	}
}

func TestVerifyPassword_Wrong(t *testing.T) {
	hash := HashPassword("mySecret123")
	if VerifyPassword(hash, "wrongPassword") {
		t.Error("VerifyPassword should return false for wrong password")
	}
}
