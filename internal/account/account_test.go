package account

import "testing"

func TestOfflineUUID(t *testing.T) {
	cases := map[string]string{
		"Notch": "b50ad385-829d-3141-a216-7e7d7539ba7f",
		"Steve": "5627dd98-e6be-3c21-b8a8-e92344183641",
		"jeb_":  "a762f560-4fce-3236-812a-b80efff0b62b",
	}
	for name, want := range cases {
		if got := offlineUUID(name); got != want {
			t.Errorf("offlineUUID(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestOfflineUUIDIsVersion3(t *testing.T) {
	uuid := offlineUUID("Steve")
	if uuid[14] != '3' {
		t.Errorf("version nibble = %c, want 3 (UUID v3); uuid=%s", uuid[14], uuid)
	}
	switch uuid[19] {
	case '8', '9', 'a', 'b':
	default:
		t.Errorf("variant nibble = %c, want one of 8/9/a/b; uuid=%s", uuid[19], uuid)
	}
}

func TestOfflineDeterministic(t *testing.T) {
	first := Offline("Alex").UUID
	second := Offline("Alex").UUID
	if first != second {
		t.Error("same username produced different UUIDs")
	}
	if Offline("Alex").UUID == Offline("Notch").UUID {
		t.Error("different usernames produced the same UUID")
	}
}

func TestOfflineFields(t *testing.T) {
	a := Offline("Steve")
	if a.Username != "Steve" {
		t.Errorf("Username = %q, want Steve", a.Username)
	}
	if a.AccessToken != "0" {
		t.Errorf("AccessToken = %q, want 0", a.AccessToken)
	}
	if a.UserType != "legacy" {
		t.Errorf("UserType = %q, want legacy", a.UserType)
	}
}
