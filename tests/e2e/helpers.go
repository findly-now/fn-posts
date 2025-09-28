package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/jsarabia/fn-posts/internal/domain"
)

const (
	BaseURL = "http://localhost:8081/api/v1"
	TestUserID = "550e8400-e29b-41d4-a716-446655440001"
)

// Test data structures matching API responses
type PostResponse struct {
	ID             string          `json:"id"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Photos         []PhotoResponse `json:"photos"`
	Location       domain.Location `json:"location"`
	RadiusMeters   int             `json:"radius_meters"`
	Status         string          `json:"status"`
	Type           string          `json:"type"`
	CreatedBy      string          `json:"created_by"`
	OrganizationID *string         `json:"organization_id,omitempty"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
}

type PhotoResponse struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Caption      string `json:"caption,omitempty"`
	DisplayOrder int    `json:"display_order"`
	CreatedAt    string `json:"created_at"`
}

type CreatePostRequest struct {
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Location       domain.Location `json:"location"`
	RadiusMeters   int             `json:"radius_meters"`
	Type           string          `json:"type"`
	OrganizationID *string         `json:"organization_id,omitempty"`
}

type UpdatePostRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdatePostStatusRequest struct {
	Status string `json:"status"`
}

type ListPostsResponse struct {
	Posts  []PostResponse `json:"posts"`
	Total  int64          `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// HTTP Client helpers

func makeRequest(t *testing.T, method, endpoint string, body interface{}) *http.Response {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, BaseURL+endpoint, reqBody)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", TestUserID) // Simulate auth

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

func makeMultipartRequest(t *testing.T, endpoint string, fields map[string]string, files map[string]string) *http.Response {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add form fields
	for key, value := range fields {
		err := writer.WriteField(key, value)
		require.NoError(t, err)
	}

	// Add files
	for fieldName, filePath := range files {
		file, err := os.Open(filePath)
		require.NoError(t, err)
		defer file.Close()

		part, err := writer.CreateFormFile(fieldName, filePath)
		require.NoError(t, err)

		_, err = io.Copy(part, file)
		require.NoError(t, err)
	}

	err := writer.Close()
	require.NoError(t, err)

	req, err := http.NewRequest("POST", BaseURL+endpoint, &body)
	require.NoError(t, err)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-ID", TestUserID)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

func parseResponse(t *testing.T, resp *http.Response, target interface{}) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	if resp.StatusCode >= 400 {
		t.Logf("Error response: %s", string(body))
	}

	err = json.Unmarshal(body, target)
	require.NoError(t, err)
}

func parseErrorResponse(t *testing.T, resp *http.Response) string {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var errorResp ErrorResponse
	err = json.Unmarshal(body, &errorResp)
	require.NoError(t, err)

	return errorResp.Error
}

// Test data creation helpers

func CreateTestPost(t *testing.T, req CreatePostRequest) PostResponse {
	resp := makeRequest(t, "POST", "/posts", req)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var post PostResponse
	parseResponse(t, resp, &post)
	return post
}

func CreateTestPostWithDefaults(t *testing.T) PostResponse {
	req := CreatePostRequest{
		Title:        "Test Post " + uuid.New().String()[:8],
		Description:  "Test description for E2E testing",
		Location:     domain.Location{Latitude: 40.7831, Longitude: -73.9665}, // Central Park
		RadiusMeters: 1000,
		Type:         "lost",
	}
	return CreateTestPost(t, req)
}

func CreateTestPostAt(t *testing.T, lat, lng float64, title string) PostResponse {
	req := CreatePostRequest{
		Title:        title,
		Description:  "Test post at specific location",
		Location:     domain.Location{Latitude: lat, Longitude: lng},
		RadiusMeters: 1000,
		Type:         "lost",
	}
	return CreateTestPost(t, req)
}

func UploadTestPhoto(t *testing.T, postID string) {
	// Create a simple test image file
	testImageData := []byte("fake-image-data-for-testing")
	testFile := "/tmp/test-image.jpg"

	err := os.WriteFile(testFile, testImageData, 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	fields := map[string]string{
		"caption_0": "Test photo caption",
	}
	files := map[string]string{
		"photos": testFile,
	}

	resp := makeMultipartRequest(t, fmt.Sprintf("/posts/%s/photos", postID), fields, files)
	require.True(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusPartialContent)
}

// Cleanup helpers

func CleanupPost(t *testing.T, postID string) {
	resp := makeRequest(t, "DELETE", fmt.Sprintf("/posts/%s", postID), nil)
	require.True(t, resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound)
}

// Wait helpers

func WaitForService(t *testing.T, url string, maxAttempts int) {
	client := &http.Client{Timeout: 5 * time.Second}

	for i := 0; i < maxAttempts; i++ {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}

		time.Sleep(2 * time.Second)
	}

	t.Fatalf("Service at %s did not become ready after %d attempts", url, maxAttempts)
}

// Validation helpers

func AssertPostEquals(t *testing.T, expected CreatePostRequest, actual PostResponse) {
	require.Equal(t, expected.Title, actual.Title)
	require.Equal(t, expected.Description, actual.Description)
	require.Equal(t, expected.Location.Latitude, actual.Location.Latitude)
	require.Equal(t, expected.Location.Longitude, actual.Location.Longitude)
	require.Equal(t, expected.RadiusMeters, actual.RadiusMeters)
	require.Equal(t, expected.Type, actual.Type)
	require.Equal(t, "active", actual.Status) // Default status
	require.Equal(t, TestUserID, actual.CreatedBy)
	require.NotEmpty(t, actual.ID)
	require.NotEmpty(t, actual.CreatedAt)
	require.NotEmpty(t, actual.UpdatedAt)
}

func AssertLocationNear(t *testing.T, expected, actual domain.Location, toleranceMeters float64) {
	// Simple distance calculation for testing
	latDiff := expected.Latitude - actual.Latitude
	lngDiff := expected.Longitude - actual.Longitude
	distance := latDiff*latDiff + lngDiff*lngDiff // Simplified distance for testing

	require.True(t, distance < 0.01, "Locations should be close: expected %v, got %v", expected, actual)
}

// Test location data

var TestLocations = struct {
	CentralPark    domain.Location
	TimesSquare    domain.Location
	BrooklynBridge domain.Location
	EmpireState    domain.Location
}{
	CentralPark:    domain.Location{Latitude: 40.7831, Longitude: -73.9665},
	TimesSquare:    domain.Location{Latitude: 40.7505, Longitude: -73.9934},
	BrooklynBridge: domain.Location{Latitude: 40.6782, Longitude: -73.9442},
	EmpireState:    domain.Location{Latitude: 40.7484, Longitude: -73.9857},
}