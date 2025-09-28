package domain

import (
	"fmt"
)

type PostErrorCode string

const (
	// Post validation errors
	PostErrorInvalidType      PostErrorCode = "POST_INVALID_TYPE"
	PostErrorInvalidStatus    PostErrorCode = "POST_INVALID_STATUS"
	PostErrorInvalidTitle     PostErrorCode = "POST_INVALID_TITLE"
	PostErrorInvalidLocation  PostErrorCode = "POST_INVALID_LOCATION"
	PostErrorCannotTransition PostErrorCode = "POST_CANNOT_TRANSITION_STATUS"

	// Photo validation errors
	PhotoErrorInvalidCount  PostErrorCode = "PHOTO_INVALID_COUNT"
	PhotoErrorInvalidURL    PostErrorCode = "PHOTO_INVALID_URL"
	PhotoErrorInvalidFormat PostErrorCode = "PHOTO_INVALID_FORMAT"
	PhotoErrorInvalidOrder  PostErrorCode = "PHOTO_INVALID_DISPLAY_ORDER"
	PhotoErrorNotFound      PostErrorCode = "PHOTO_NOT_FOUND"

	// Location validation errors
	LocationErrorInvalidLatitude  PostErrorCode = "LOCATION_INVALID_LATITUDE"
	LocationErrorInvalidLongitude PostErrorCode = "LOCATION_INVALID_LONGITUDE"

	// Business rule errors
	BusinessErrorPostNotFound PostErrorCode = "BUSINESS_POST_NOT_FOUND"
	BusinessErrorUnauthorized PostErrorCode = "BUSINESS_UNAUTHORIZED"
	BusinessErrorPostExpired  PostErrorCode = "BUSINESS_POST_EXPIRED"

	// Repository errors
	RepositoryErrorNotFound   PostErrorCode = "REPOSITORY_NOT_FOUND"
	RepositoryErrorDuplicate  PostErrorCode = "REPOSITORY_DUPLICATE"
	RepositoryErrorConnection PostErrorCode = "REPOSITORY_CONNECTION"
)

type PostError struct {
	Code    PostErrorCode
	Message string
	Details map[string]interface{}
	Cause   error
}

func (e PostError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e PostError) Unwrap() error {
	return e.Cause
}

func (e PostError) WithDetail(key string, value interface{}) PostError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

func (e PostError) WithCause(cause error) PostError {
	e.Cause = cause
	return e
}

func NewPostError(code PostErrorCode, message string) PostError {
	return PostError{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

func ErrInvalidPostType(providedType string) PostError {
	return NewPostError(
		PostErrorInvalidType,
		"Post type must be 'lost' or 'found'",
	).WithDetail("provided_type", providedType)
}

func ErrInvalidPostStatus(providedStatus string) PostError {
	return NewPostError(
		PostErrorInvalidStatus,
		"Post status is not valid",
	).WithDetail("provided_status", providedStatus)
}

func ErrInvalidTitle() PostError {
	return NewPostError(
		PostErrorInvalidTitle,
		"Post title cannot be empty and must be between 1 and 200 characters",
	)
}

func ErrInvalidLocation(latitude, longitude float64) PostError {
	return NewPostError(
		PostErrorInvalidLocation,
		"Location coordinates are invalid",
	).WithDetail("latitude", latitude).WithDetail("longitude", longitude)
}

func ErrCannotTransitionStatus(currentStatus, newStatus PostStatus) PostError {
	return NewPostError(
		PostErrorCannotTransition,
		"Cannot transition to the requested status",
	).WithDetail("current_status", string(currentStatus)).WithDetail("new_status", string(newStatus))
}

func ErrInvalidPhotoCount(currentCount int) PostError {
	return NewPostError(
		PhotoErrorInvalidCount,
		"Post must have between 1 and 10 photos",
	).WithDetail("current_count", currentCount)
}

func ErrInvalidPhotoURL(url string) PostError {
	return NewPostError(
		PhotoErrorInvalidURL,
		"Photo URL is invalid or empty",
	).WithDetail("url", url)
}

func ErrInvalidPhotoFormat(format string) PostError {
	return NewPostError(
		PhotoErrorInvalidFormat,
		"Photo format is not supported. Allowed formats: jpg, jpeg, png, webp",
	).WithDetail("format", format)
}

func ErrInvalidDisplayOrder(order int) PostError {
	return NewPostError(
		PhotoErrorInvalidOrder,
		"Photo display order must be between 1 and 10",
	).WithDetail("display_order", order)
}

func ErrPhotoNotFound(photoID PhotoID) PostError {
	return NewPostError(
		PhotoErrorNotFound,
		"Photo not found",
	).WithDetail("photo_id", photoID.String())
}

func ErrInvalidLatitude(latitude float64) PostError {
	return NewPostError(
		LocationErrorInvalidLatitude,
		"Latitude must be between -90 and 90 degrees",
	).WithDetail("latitude", latitude)
}

func ErrInvalidLongitude(longitude float64) PostError {
	return NewPostError(
		LocationErrorInvalidLongitude,
		"Longitude must be between -180 and 180 degrees",
	).WithDetail("longitude", longitude)
}

func ErrPostNotFound(postID PostID) PostError {
	return NewPostError(
		BusinessErrorPostNotFound,
		"Post not found",
	).WithDetail("post_id", postID.String())
}

func ErrUnauthorizedOperation(userID UserID, operation string) PostError {
	return NewPostError(
		BusinessErrorUnauthorized,
		"User is not authorized to perform this operation",
	).WithDetail("user_id", userID.String()).WithDetail("operation", operation)
}

func ErrRepositoryNotFound(entityType string, id string) PostError {
	return NewPostError(
		RepositoryErrorNotFound,
		"Entity not found in repository",
	).WithDetail("entity_type", entityType).WithDetail("id", id)
}

func ErrRepositoryConnection(operation string) PostError {
	return NewPostError(
		RepositoryErrorConnection,
		"Repository connection error",
	).WithDetail("operation", operation)
}

func IsPostError(err error) bool {
	_, ok := err.(PostError)
	return ok
}

func IsPostErrorCode(err error, code PostErrorCode) bool {
	if postErr, ok := err.(PostError); ok {
		return postErr.Code == code
	}
	return false
}

func GetPostErrorCode(err error) PostErrorCode {
	if postErr, ok := err.(PostError); ok {
		return postErr.Code
	}
	return ""
}
