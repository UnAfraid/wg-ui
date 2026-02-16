package backend

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var namePattern = regexp.MustCompile(`^[a-zA-Z0-9.\-_]{1,64}$`)

type Backend struct {
	Id           string
	Name         string
	Description  string
	Url          string
	Enabled      bool
	CreateUserId string
	UpdateUserId string
	DeleteUserId string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

func (b *Backend) Type() string {
	parsed, err := ParseURL(b.Url)
	if err != nil {
		return ""
	}
	return parsed.Type
}

func (b *Backend) validate(fieldMask *UpdateFieldMask) error {
	if fieldMask == nil {
		if len(strings.TrimSpace(b.Name)) == 0 {
			return fmt.Errorf("name is required")
		}

		if !namePattern.MatchString(b.Name) {
			return fmt.Errorf("invalid name: must match pattern %s", namePattern.String())
		}
	}

	if fieldMask == nil || fieldMask.Description {
		if len(b.Description) > 255 {
			return fmt.Errorf("description must not be longer than 255 characters")
		}
	}

	if fieldMask == nil || fieldMask.Url {
		if len(strings.TrimSpace(b.Url)) == 0 {
			return fmt.Errorf("url is required")
		}

		if _, err := ParseURL(b.Url); err != nil {
			return fmt.Errorf("invalid url: %w", err)
		}
	}

	return nil
}

func (b *Backend) update(options *UpdateOptions, fieldMask *UpdateFieldMask) {
	if fieldMask.Name {
		b.Name = options.Name
	}

	if fieldMask.Description {
		b.Description = options.Description
	}

	if fieldMask.Url {
		b.Url = options.Url
	}

	if fieldMask.Enabled {
		b.Enabled = options.Enabled
	}

	if fieldMask.UpdateUserId {
		b.UpdateUserId = options.UpdateUserId
	}
}
