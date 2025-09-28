package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/jsarabia/fn-posts/internal/domain"
)

func TestCreatePost(t *testing.T) {
	t.Run("should create post successfully", func(t *testing.T) {
		req := CreatePostRequest{
			Title:        "Lost iPhone 14 Pro",
			Description:  "Black iPhone 14 Pro lost near Central Park",
			Location:     TestLocations.CentralPark,
			RadiusMeters: 1000,
			Type:         "lost",
		}

		post := CreateTestPost(t, req)
		defer CleanupPost(t, post.ID)

		AssertPostEquals(t, req, post)
	})

	t.Run("should create found post successfully", func(t *testing.T) {
		req := CreatePostRequest{
			Title:        "Found Keys",
			Description:  "Set of house keys found at Times Square",
			Location:     TestLocations.TimesSquare,
			RadiusMeters: 500,
			Type:         "found",
		}

		post := CreateTestPost(t, req)
		defer CleanupPost(t, post.ID)

		AssertPostEquals(t, req, post)
	})

	t.Run("should fail with invalid data", func(t *testing.T) {
		testCases := []struct {
			name string
			req  CreatePostRequest
		}{
			{
				name: "empty title",
				req: CreatePostRequest{
					Title:        "",
					Description:  "Valid description",
					Location:     TestLocations.CentralPark,
					RadiusMeters: 1000,
					Type:         "lost",
				},
			},
			{
				name: "invalid post type",
				req: CreatePostRequest{
					Title:        "Valid title",
					Description:  "Valid description",
					Location:     TestLocations.CentralPark,
					RadiusMeters: 1000,
					Type:         "invalid",
				},
			},
			{
				name: "invalid location",
				req: CreatePostRequest{
					Title:        "Valid title",
					Description:  "Valid description",
					Location:     domain.Location{Latitude: 91.0, Longitude: -200.0},
					RadiusMeters: 1000,
					Type:         "lost",
				},
			},
			{
				name: "radius too small",
				req: CreatePostRequest{
					Title:        "Valid title",
					Description:  "Valid description",
					Location:     TestLocations.CentralPark,
					RadiusMeters: 50, // Below minimum
					Type:         "lost",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				resp := makeRequest(t, "POST", "/posts", tc.req)
				require.Equal(t, http.StatusBadRequest, resp.StatusCode)
			})
		}
	})
}

func TestGetPost(t *testing.T) {
	t.Run("should get existing post", func(t *testing.T) {
		// Create a test post
		post := CreateTestPostWithDefaults(t)
		defer CleanupPost(t, post.ID)

		// Get the post
		resp := makeRequest(t, "GET", fmt.Sprintf("/posts/%s", post.ID), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var retrievedPost PostResponse
		parseResponse(t, resp, &retrievedPost)

		require.Equal(t, post.ID, retrievedPost.ID)
		require.Equal(t, post.Title, retrievedPost.Title)
		require.Equal(t, post.Description, retrievedPost.Description)
	})

	t.Run("should return 404 for non-existent post", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/posts/550e8400-e29b-41d4-a716-446655440404", nil)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("should return 400 for invalid post ID", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/posts/invalid-id", nil)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestUpdatePost(t *testing.T) {
	t.Run("should update post successfully", func(t *testing.T) {
		// Create a test post
		post := CreateTestPostWithDefaults(t)
		defer CleanupPost(t, post.ID)

		// Update the post
		updateReq := UpdatePostRequest{
			Title:       "Updated Title",
			Description: "Updated Description",
		}

		resp := makeRequest(t, "PUT", fmt.Sprintf("/posts/%s", post.ID), updateReq)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var updatedPost PostResponse
		parseResponse(t, resp, &updatedPost)

		require.Equal(t, post.ID, updatedPost.ID)
		require.Equal(t, updateReq.Title, updatedPost.Title)
		require.Equal(t, updateReq.Description, updatedPost.Description)
		require.Equal(t, post.Location.Latitude, updatedPost.Location.Latitude) // Location unchanged
		require.Equal(t, post.Location.Longitude, updatedPost.Location.Longitude)
	})

	t.Run("should fail with invalid data", func(t *testing.T) {
		post := CreateTestPostWithDefaults(t)
		defer CleanupPost(t, post.ID)

		// Try to update with empty title
		updateReq := UpdatePostRequest{
			Title:       "",
			Description: "Valid description",
		}

		resp := makeRequest(t, "PUT", fmt.Sprintf("/posts/%s", post.ID), updateReq)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestUpdatePostStatus(t *testing.T) {
	t.Run("should update status successfully", func(t *testing.T) {
		// Create a test post
		post := CreateTestPostWithDefaults(t)
		defer CleanupPost(t, post.ID)

		// Update status to resolved
		statusReq := UpdatePostStatusRequest{Status: "resolved"}

		resp := makeRequest(t, "PATCH", fmt.Sprintf("/posts/%s/status", post.ID), statusReq)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var updatedPost PostResponse
		parseResponse(t, resp, &updatedPost)

		require.Equal(t, post.ID, updatedPost.ID)
		require.Equal(t, "resolved", updatedPost.Status)
	})

	t.Run("should fail with invalid status", func(t *testing.T) {
		post := CreateTestPostWithDefaults(t)
		defer CleanupPost(t, post.ID)

		statusReq := UpdatePostStatusRequest{Status: "invalid"}

		resp := makeRequest(t, "PATCH", fmt.Sprintf("/posts/%s/status", post.ID), statusReq)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestDeletePost(t *testing.T) {
	t.Run("should delete post successfully", func(t *testing.T) {
		// Create a test post
		post := CreateTestPostWithDefaults(t)

		// Delete the post
		resp := makeRequest(t, "DELETE", fmt.Sprintf("/posts/%s", post.ID), nil)
		require.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify post is gone
		resp = makeRequest(t, "GET", fmt.Sprintf("/posts/%s", post.ID), nil)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("should return 404 for non-existent post", func(t *testing.T) {
		resp := makeRequest(t, "DELETE", "/posts/550e8400-e29b-41d4-a716-446655440404", nil)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestListPosts(t *testing.T) {
	t.Run("should list posts with pagination", func(t *testing.T) {
		// Create multiple test posts
		posts := make([]PostResponse, 3)
		for i := 0; i < 3; i++ {
			posts[i] = CreateTestPostAt(t,
				40.7831+float64(i)*0.001, // Slightly different locations
				-73.9665+float64(i)*0.001,
				fmt.Sprintf("Test Post %d", i+1),
			)
			defer CleanupPost(t, posts[i].ID)
		}

		// List posts with limit
		resp := makeRequest(t, "GET", "/posts?limit=2", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp ListPostsResponse
		parseResponse(t, resp, &listResp)

		require.Len(t, listResp.Posts, 2)
		require.Equal(t, 2, listResp.Limit)
		require.Equal(t, 0, listResp.Offset)
		require.GreaterOrEqual(t, listResp.Total, int64(3))
	})

	t.Run("should filter posts by type", func(t *testing.T) {
		// Create lost and found posts
		lostPost := CreateTestPostAt(t, 40.7831, -73.9665, "Lost Item")
		defer CleanupPost(t, lostPost.ID)

		foundReq := CreatePostRequest{
			Title:        "Found Item",
			Description:  "Test found item",
			Location:     TestLocations.TimesSquare,
			RadiusMeters: 1000,
			Type:         "found",
		}
		foundPost := CreateTestPost(t, foundReq)
		defer CleanupPost(t, foundPost.ID)

		// Filter by lost type
		resp := makeRequest(t, "GET", "/posts?type=lost", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp ListPostsResponse
		parseResponse(t, resp, &listResp)

		// All returned posts should be lost type
		for _, post := range listResp.Posts {
			require.Equal(t, "lost", post.Type)
		}
	})

	t.Run("should filter posts by status", func(t *testing.T) {
		// Create a post and mark it as resolved
		post := CreateTestPostWithDefaults(t)
		defer CleanupPost(t, post.ID)

		statusReq := UpdatePostStatusRequest{Status: "resolved"}
		resp := makeRequest(t, "PATCH", fmt.Sprintf("/posts/%s/status", post.ID), statusReq)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		// Filter by resolved status
		resp = makeRequest(t, "GET", "/posts?status=resolved", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp ListPostsResponse
		parseResponse(t, resp, &listResp)

		// All returned posts should be resolved
		for _, p := range listResp.Posts {
			require.Equal(t, "resolved", p.Status)
		}
	})
}

func TestGetUserPosts(t *testing.T) {
	t.Run("should get posts for specific user", func(t *testing.T) {
		// Create test posts (they'll all have the same test user ID)
		posts := make([]PostResponse, 2)
		for i := 0; i < 2; i++ {
			posts[i] = CreateTestPostAt(t,
				40.7831+float64(i)*0.001,
				-73.9665+float64(i)*0.001,
				fmt.Sprintf("User Post %d", i+1),
			)
			defer CleanupPost(t, posts[i].ID)
		}

		// Get posts for test user
		resp := makeRequest(t, "GET", fmt.Sprintf("/users/%s/posts", TestUserID), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var userPostsResp map[string]interface{}
		parseResponse(t, resp, &userPostsResp)

		postsList := userPostsResp["posts"].([]interface{})
		require.GreaterOrEqual(t, len(postsList), 2)

		// All posts should belong to the test user
		for _, p := range postsList {
			post := p.(map[string]interface{})
			require.Equal(t, TestUserID, post["created_by"])
		}
	})

	t.Run("should return empty list for non-existent user", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/users/550e8400-e29b-41d4-a716-446655440404/posts", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var userPostsResp map[string]interface{}
		parseResponse(t, resp, &userPostsResp)

		postsList := userPostsResp["posts"].([]interface{})
		require.Len(t, postsList, 0)
	})
}