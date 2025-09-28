package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jsarabia/fn-posts/internal/domain"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// HandleError maps domain errors to appropriate HTTP responses
func HandleError(c *gin.Context, err error) {
	var postErr domain.PostError

	switch {
	case errors.As(err, &postErr):
		handlePostError(c, postErr)
	default:
		// Generic error handling
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Internal server error",
			Code:  "INTERNAL_ERROR",
		})
	}
}

func handlePostError(c *gin.Context, err domain.PostError) {
	switch err.Code {
	case "POST_NOT_FOUND":
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: err.Message,
			Code:  string(err.Code),
		})
	case "INVALID_POST_TYPE", "INVALID_POST_STATUS", "INVALID_PHOTO_COUNT",
		"INVALID_TITLE", "INVALID_LOCATION", "INVALID_STATUS_TRANSITION",
		"TOO_MANY_PHOTOS", "INVALID_PHOTO_URL", "INVALID_PHOTO_FORMAT",
		"INVALID_DISPLAY_ORDER", "INVALID_LATITUDE", "INVALID_LONGITUDE":
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: err.Message,
			Code:  string(err.Code),
		})
	case "PHOTO_NOT_FOUND":
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: err.Message,
			Code:  string(err.Code),
		})
	case "UNAUTHORIZED":
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: err.Message,
			Code:  string(err.Code),
		})
	default:
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Internal server error",
			Code:  "INTERNAL_ERROR",
		})
	}
}
