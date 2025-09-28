package domain

import (
	"fmt"

	"github.com/google/uuid"
)

type PostID struct {
	value uuid.UUID
}

func NewPostID() PostID {
	return PostID{value: uuid.New()}
}

func PostIDFromString(s string) (PostID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return PostID{}, fmt.Errorf("invalid post ID: %w", err)
	}
	return PostID{value: id}, nil
}

func (p PostID) String() string {
	return p.value.String()
}

func (p PostID) UUID() uuid.UUID {
	return p.value
}

func (p PostID) IsZero() bool {
	return p.value == uuid.Nil
}

func (p PostID) Equals(other PostID) bool {
	return p.value == other.value
}

type UserID struct {
	value uuid.UUID
}

func NewUserID() UserID {
	return UserID{value: uuid.New()}
}

func UserIDFromString(s string) (UserID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return UserID{}, fmt.Errorf("invalid user ID: %w", err)
	}
	return UserID{value: id}, nil
}

func UserIDFromUUID(id uuid.UUID) UserID {
	return UserID{value: id}
}

func (u UserID) String() string {
	return u.value.String()
}

func (u UserID) UUID() uuid.UUID {
	return u.value
}

func (u UserID) IsZero() bool {
	return u.value == uuid.Nil
}

func (u UserID) Equals(other UserID) bool {
	return u.value == other.value
}

type OrganizationID struct {
	value uuid.UUID
}

func NewOrganizationID() OrganizationID {
	return OrganizationID{value: uuid.New()}
}

func OrganizationIDFromString(s string) (OrganizationID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return OrganizationID{}, fmt.Errorf("invalid organization ID: %w", err)
	}
	return OrganizationID{value: id}, nil
}

func OrganizationIDFromUUID(id uuid.UUID) OrganizationID {
	return OrganizationID{value: id}
}

func (o OrganizationID) String() string {
	return o.value.String()
}

func (o OrganizationID) UUID() uuid.UUID {
	return o.value
}

func (o OrganizationID) IsZero() bool {
	return o.value == uuid.Nil
}

func (o OrganizationID) Equals(other OrganizationID) bool {
	return o.value == other.value
}

type PhotoID struct {
	value uuid.UUID
}

func NewPhotoID() PhotoID {
	return PhotoID{value: uuid.New()}
}

func PhotoIDFromString(s string) (PhotoID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return PhotoID{}, fmt.Errorf("invalid photo ID: %w", err)
	}
	return PhotoID{value: id}, nil
}

func PhotoIDFromUUID(id uuid.UUID) PhotoID {
	return PhotoID{value: id}
}

func (p PhotoID) String() string {
	return p.value.String()
}

func (p PhotoID) UUID() uuid.UUID {
	return p.value
}

func (p PhotoID) IsZero() bool {
	return p.value == uuid.Nil
}

func (p PhotoID) Equals(other PhotoID) bool {
	return p.value == other.value
}
