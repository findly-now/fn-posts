package e2e

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPhotoUpload(t *testing.T) {
	t.Run("should upload single photo successfully", func(t *testing.T) {
		// Create a test post
		post := CreateTestPostWithDefaults(t)
		defer CleanupPost(t, post.ID)

		// Create test image file
		testImageData := []byte("fake-jpeg-header-data-for-testing")
		testFile := "/tmp/test-single.jpg"
		err := os.WriteFile(testFile, testImageData, 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		// Upload photo
		fields := map[string]string{
			"caption_0": "Test single photo caption",
		}
		files := map[string]string{
			"photos": testFile,
		}

		resp := makeMultipartRequest(t, fmt.Sprintf("/posts/%s/photos", post.ID), fields, files)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var uploadResp map[string]interface{}
		parseResponse(t, resp, &uploadResp)

		uploadedPhotos := uploadResp["uploaded_photos"].([]interface{})
		require.Len(t, uploadedPhotos, 1)

		successCount := int(uploadResp["success_count"].(float64))
		totalCount := int(uploadResp["total_count"].(float64))

		require.Equal(t, 1, successCount)
		require.Equal(t, 1, totalCount)

		// Verify photo details
		photo := uploadedPhotos[0].(map[string]interface{})
		photoDetails := photo["photo"].(map[string]interface{})

		require.NotEmpty(t, photoDetails["id"])
		require.NotEmpty(t, photoDetails["url"])
		require.Equal(t, "Test single photo caption", photoDetails["caption"])
		require.Equal(t, float64(1), photoDetails["display_order"])
	})

	t.Run("should upload multiple photos successfully", func(t *testing.T) {
		// Create a test post
		post := CreateTestPostWithDefaults(t)
		defer CleanupPost(t, post.ID)

		// Create multiple test image files
		testFiles := make([]string, 3)
		for i := 0; i < 3; i++ {
			testImageData := []byte(fmt.Sprintf("fake-jpeg-data-%d", i))
			testFile := fmt.Sprintf("/tmp/test-multi-%d.jpg", i)
			err := os.WriteFile(testFile, testImageData, 0644)
			require.NoError(t, err)
			defer os.Remove(testFile)
			testFiles[i] = testFile
		}

		// Create multipart request with multiple files
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)

		// Add captions
		for i := 0; i < 3; i++ {
			err := writer.WriteField(fmt.Sprintf("caption_%d", i), fmt.Sprintf("Caption for photo %d", i+1))
			require.NoError(t, err)
		}

		// Add files
		for i, testFile := range testFiles {
			file, err := os.Open(testFile)
			require.NoError(t, err)
			defer file.Close()

			part, err := writer.CreateFormFile("photos", fmt.Sprintf("test-photo-%d.jpg", i+1))
			require.NoError(t, err)

			_, err = io.Copy(part, file)
			require.NoError(t, err)
		}

		err := writer.Close()
		require.NoError(t, err)

		// Make request
		req, err := http.NewRequest("POST", BaseURL+fmt.Sprintf("/posts/%s/photos", post.ID), &body)
		require.NoError(t, err)

		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("X-User-ID", TestUserID)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var uploadResp map[string]interface{}
		parseResponse(t, resp, &uploadResp)

		uploadedPhotos := uploadResp["uploaded_photos"].([]interface{})
		require.Len(t, uploadedPhotos, 3)

		successCount := int(uploadResp["success_count"].(float64))
		totalCount := int(uploadResp["total_count"].(float64))

		require.Equal(t, 3, successCount)
		require.Equal(t, 3, totalCount)

		// Verify display order
		for i, photoInterface := range uploadedPhotos {
			photo := photoInterface.(map[string]interface{})
			photoDetails := photo["photo"].(map[string]interface{})
			require.Equal(t, float64(i+1), photoDetails["display_order"])
		}
	})

	t.Run("should validate photo count limit", func(t *testing.T) {
		// Create a test post
		post := CreateTestPostWithDefaults(t)
		defer CleanupPost(t, post.ID)

		// Try to upload 11 photos (exceeds limit of 10)
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)

		// Create 11 fake files
		for i := 0; i < 11; i++ {
			testImageData := []byte(fmt.Sprintf("fake-image-data-%d", i))
			part, err := writer.CreateFormFile("photos", fmt.Sprintf("test-%d.jpg", i))
			require.NoError(t, err)
			_, err = part.Write(testImageData)
			require.NoError(t, err)
		}

		err := writer.Close()
		require.NoError(t, err)

		req, err := http.NewRequest("POST", BaseURL+fmt.Sprintf("/posts/%s/photos", post.ID), &body)
		require.NoError(t, err)

		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("X-User-ID", TestUserID)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		errorMsg := parseErrorResponse(t, resp)
		require.Contains(t, errorMsg, "Maximum 10 photos allowed")
	})

	t.Run("should validate file formats", func(t *testing.T) {
		// Create a test post
		post := CreateTestPostWithDefaults(t)
		defer CleanupPost(t, post.ID)

		// Create an invalid file (txt instead of image)
		testTextData := []byte("this is not an image file")
		testFile := "/tmp/test-invalid.txt"
		err := os.WriteFile(testFile, testTextData, 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		// Try to upload invalid file
		fields := map[string]string{}
		files := map[string]string{
			"photos": testFile,
		}

		resp := makeMultipartRequest(t, fmt.Sprintf("/posts/%s/photos", post.ID), fields, files)
		// This should either fail completely or show in the errors array
		require.True(t, resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusPartialContent)

		if resp.StatusCode == http.StatusPartialContent {
			var uploadResp map[string]interface{}
			parseResponse(t, resp, &uploadResp)

			errors := uploadResp["errors"].([]interface{})
			require.Greater(t, len(errors), 0, "Should have errors for invalid file format")

			successCount := int(uploadResp["success_count"].(float64))
			require.Equal(t, 0, successCount, "Should have no successful uploads")
		}
	})

	t.Run("should require photos in request", func(t *testing.T) {
		// Create a test post
		post := CreateTestPostWithDefaults(t)
		defer CleanupPost(t, post.ID)

		// Make request without any files
		fields := map[string]string{}
		files := map[string]string{} // Empty files

		resp := makeMultipartRequest(t, fmt.Sprintf("/posts/%s/photos", post.ID), fields, files)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		errorMsg := parseErrorResponse(t, resp)
		require.Contains(t, errorMsg, "No photos provided")
	})

	t.Run("should require valid post ID", func(t *testing.T) {
		// Create test image file
		testImageData := []byte("fake-jpeg-data")
		testFile := "/tmp/test-invalid-post.jpg"
		err := os.WriteFile(testFile, testImageData, 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		// Try to upload to non-existent post
		fields := map[string]string{}
		files := map[string]string{
			"photos": testFile,
		}

		resp := makeMultipartRequest(t, "/posts/550e8400-e29b-41d4-a716-446655440404/photos", fields, files)
		require.Equal(t, http.StatusInternalServerError, resp.StatusCode) // Post service will fail to find post
	})

	t.Run("should require authentication", func(t *testing.T) {
		// Create a test post
		post := CreateTestPostWithDefaults(t)
		defer CleanupPost(t, post.ID)

		// Create test image file
		testImageData := []byte("fake-jpeg-data")
		testFile := "/tmp/test-auth.jpg"
		err := os.WriteFile(testFile, testImageData, 0644)
		require.NoError(t, err)
		defer os.Remove(testFile)

		// Make request without user ID header
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)

		file, err := os.Open(testFile)
		require.NoError(t, err)
		defer file.Close()

		part, err := writer.CreateFormFile("photos", "test.jpg")
		require.NoError(t, err)

		_, err = io.Copy(part, file)
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		req, err := http.NewRequest("POST", BaseURL+fmt.Sprintf("/posts/%s/photos", post.ID), &body)
		require.NoError(t, err)

		req.Header.Set("Content-Type", writer.FormDataContentType())
		// Don't set X-User-ID header

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		errorMsg := parseErrorResponse(t, resp)
		require.Contains(t, errorMsg, "not authenticated")
	})
}

func TestPhotoDelete(t *testing.T) {
	t.Run("should delete photo successfully", func(t *testing.T) {
		// Create a test post
		post := CreateTestPostWithDefaults(t)
		defer CleanupPost(t, post.ID)

		// Upload a photo first
		UploadTestPhoto(t, post.ID)

		// Get the post to find the photo ID
		resp := makeRequest(t, "GET", fmt.Sprintf("/posts/%s", post.ID), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var postResp PostResponse
		parseResponse(t, resp, &postResp)
		require.Len(t, postResp.Photos, 1)

		photoID := postResp.Photos[0].ID

		// Delete the photo
		resp = makeRequest(t, "DELETE", fmt.Sprintf("/posts/%s/photos/%s", post.ID, photoID), nil)
		require.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify photo is deleted by getting post again
		resp = makeRequest(t, "GET", fmt.Sprintf("/posts/%s", post.ID), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		parseResponse(t, resp, &postResp)
		require.Len(t, postResp.Photos, 0, "Photo should be deleted")
	})

	t.Run("should return 404 for non-existent photo", func(t *testing.T) {
		// Create a test post
		post := CreateTestPostWithDefaults(t)
		defer CleanupPost(t, post.ID)

		// Try to delete non-existent photo
		resp := makeRequest(t, "DELETE", fmt.Sprintf("/posts/%s/photos/550e8400-e29b-41d4-a716-446655440404", post.ID), nil)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("should validate post and photo IDs", func(t *testing.T) {
		testCases := []struct {
			name     string
			endpoint string
		}{
			{"invalid post ID", "/posts/invalid/photos/550e8400-e29b-41d4-a716-446655440001"},
			{"invalid photo ID", "/posts/550e8400-e29b-41d4-a716-446655440001/photos/invalid"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				resp := makeRequest(t, "DELETE", tc.endpoint, nil)
				require.Equal(t, http.StatusBadRequest, resp.StatusCode)
			})
		}
	})
}