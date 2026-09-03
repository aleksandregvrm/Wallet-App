package unit_tests

import (
	"errors"
	"testing"

	"go-task-wallet-service/auth-service/internal/domain"
	"go-task-wallet-service/auth-service/internal/utils"
)

func validUser() *domain.User {
	return &domain.User{
		Name:     "Jane Doe",
		Username: "janedoe",
		Email:    "jane@example.com",
		Password: "Hunter22!",
	}
}

func TestValidateUser_Valid(t *testing.T) {
	user := validUser()
	if err := utils.ValidateUser(user); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateUser_TrimsAndLowercasesFields(t *testing.T) {
	user := validUser()
	user.Name = "  Jane Doe  "
	user.Username = "  janedoe  "
	user.Email = "  JANE@EXAMPLE.COM  "

	if err := utils.ValidateUser(user); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Name != "Jane Doe" || user.Username != "janedoe" || user.Email != "jane@example.com" {
		t.Fatalf("expected fields to be trimmed/lowercased, got: %+v", user)
	}
}

func TestValidateUser_MissingName(t *testing.T) {
	user := validUser()
	user.Name = "   "
	if err := utils.ValidateUser(user); !errors.Is(err, utils.ErrNameRequired) {
		t.Fatalf("expected ErrNameRequired, got: %v", err)
	}
}

func TestValidateUser_MissingUsername(t *testing.T) {
	user := validUser()
	user.Username = "   "
	if err := utils.ValidateUser(user); !errors.Is(err, utils.ErrUsernameRequired) {
		t.Fatalf("expected ErrUsernameRequired, got: %v", err)
	}
}

func TestValidateUser_InvalidUsername(t *testing.T) {
	cases := []string{"jd", "this-username-is-way-too-long-to-be-valid", "invalid username", "invalid$name"}
	for _, username := range cases {
		user := validUser()
		user.Username = username
		if err := utils.ValidateUser(user); !errors.Is(err, utils.ErrUsernameInvalid) {
			t.Fatalf("username %q: expected ErrUsernameInvalid, got: %v", username, err)
		}
	}
}

func TestValidateUser_MissingEmail(t *testing.T) {
	user := validUser()
	user.Email = "   "
	if err := utils.ValidateUser(user); !errors.Is(err, utils.ErrEmailRequired) {
		t.Fatalf("expected ErrEmailRequired, got: %v", err)
	}
}

func TestValidateUser_InvalidEmail(t *testing.T) {
	cases := []string{"not-an-email", "missing-domain@", "@missing-local.com", "spaces in@email.com"}
	for _, email := range cases {
		user := validUser()
		user.Email = email
		if err := utils.ValidateUser(user); !errors.Is(err, utils.ErrEmailInvalid) {
			t.Fatalf("email %q: expected ErrEmailInvalid, got: %v", email, err)
		}
	}
}

func TestValidateUser_PasswordTooShort(t *testing.T) {
	user := validUser()
	user.Password = "Ab1!"
	if err := utils.ValidateUser(user); !errors.Is(err, utils.ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got: %v", err)
	}
}

func TestValidateUser_PasswordWeak(t *testing.T) {
	cases := map[string]string{
		"no uppercase": "hunter22!",
		"no lowercase": "HUNTER22!",
		"no number":    "HunterTwo!",
		"no special":   "Hunter2222",
	}
	for name, password := range cases {
		user := validUser()
		user.Password = password
		if err := utils.ValidateUser(user); !errors.Is(err, utils.ErrPasswordWeak) {
			t.Fatalf("%s (%q): expected ErrPasswordWeak, got: %v", name, password, err)
		}
	}
}
