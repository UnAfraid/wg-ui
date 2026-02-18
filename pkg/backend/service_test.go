package backend

import "testing"

func TestProcessUpdateBackendKeepsExistingPasswordWhenRedacted(t *testing.T) {
	b := &Backend{
		Url: "routeros://admin:secret@router.example.com:443/rest",
	}

	options := &UpdateOptions{
		Url: "routeros://admin:***@router.example.com:443/rest?insecureSkipVerify=true",
	}
	fieldMask := &UpdateFieldMask{Url: true}

	if err := processUpdateBackend(b, options, fieldMask, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "routeros://admin:secret@router.example.com:443/rest?insecureSkipVerify=true"
	if b.Url != expected {
		t.Fatalf("expected %q, got %q", expected, b.Url)
	}
}

func TestProcessCreateBackendRejectsRedactedPasswordPlaceholder(t *testing.T) {
	_, err := processCreateBackend(&CreateOptions{
		Name: "routeros",
		Url:  "routeros://admin:***@router.example.com:443/rest",
	}, "user-id")
	if err == nil {
		t.Fatalf("expected create backend to reject redacted password placeholder")
	}
}
