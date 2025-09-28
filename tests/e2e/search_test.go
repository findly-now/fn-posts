package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSearchNearbyPosts(t *testing.T) {
	t.Run("should find nearby posts within radius", func(t *testing.T) {
		// Create posts at different NYC locations
		centralParkPost := CreateTestPostAt(t, TestLocations.CentralPark.Latitude, TestLocations.CentralPark.Longitude, "Central Park Post")
		defer CleanupPost(t, centralParkPost.ID)

		timesSquarePost := CreateTestPostAt(t, TestLocations.TimesSquare.Latitude, TestLocations.TimesSquare.Longitude, "Times Square Post")
		defer CleanupPost(t, timesSquarePost.ID)

		brooklynBridgePost := CreateTestPostAt(t, TestLocations.BrooklynBridge.Latitude, TestLocations.BrooklynBridge.Longitude, "Brooklyn Bridge Post")
		defer CleanupPost(t, brooklynBridgePost.ID)

		// Search from Central Park with 5km radius (should find Central Park and Times Square)
		endpoint := fmt.Sprintf("/posts/nearby?lat=%f&lng=%f&radius=5000",
			TestLocations.CentralPark.Latitude,
			TestLocations.CentralPark.Longitude)

		resp := makeRequest(t, "GET", endpoint, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var searchResp map[string]interface{}
		parseResponse(t, resp, &searchResp)

		posts := searchResp["posts"].([]interface{})
		count := int(searchResp["count"].(float64))

		require.GreaterOrEqual(t, count, 2, "Should find at least Central Park and Times Square posts")

		// Verify the posts are within the search area
		foundCentralPark := false
		foundTimesSquare := false

		for _, p := range posts {
			post := p.(map[string]interface{})
			title := post["title"].(string)

			if title == "Central Park Post" {
				foundCentralPark = true
			} else if title == "Times Square Post" {
				foundTimesSquare = true
			}
		}

		require.True(t, foundCentralPark, "Should find Central Park post")
		require.True(t, foundTimesSquare, "Should find Times Square post")
	})

	t.Run("should respect radius limits", func(t *testing.T) {
		// Create a post at Central Park
		post := CreateTestPostAt(t, TestLocations.CentralPark.Latitude, TestLocations.CentralPark.Longitude, "Central Park Post")
		defer CleanupPost(t, post.ID)

		// Search from Central Park with very small radius (100m)
		endpoint := fmt.Sprintf("/posts/nearby?lat=%f&lng=%f&radius=100",
			TestLocations.CentralPark.Latitude,
			TestLocations.CentralPark.Longitude)

		resp := makeRequest(t, "GET", endpoint, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var searchResp map[string]interface{}
		parseResponse(t, resp, &searchResp)

		posts := searchResp["posts"].([]interface{})
		count := int(searchResp["count"].(float64))

		// Should find the Central Park post
		require.GreaterOrEqual(t, count, 1, "Should find at least the Central Park post")

		// Search from Times Square with same small radius - should not find Central Park post
		endpoint = fmt.Sprintf("/posts/nearby?lat=%f&lng=%f&radius=100",
			TestLocations.TimesSquare.Latitude,
			TestLocations.TimesSquare.Longitude)

		resp = makeRequest(t, "GET", endpoint, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		parseResponse(t, resp, &searchResp)
		posts = searchResp["posts"].([]interface{})

		// Should not find the Central Park post
		for _, p := range posts {
			post := p.(map[string]interface{})
			title := post["title"].(string)
			require.NotEqual(t, "Central Park Post", title, "Should not find Central Park post from Times Square with small radius")
		}
	})

	t.Run("should filter by post type", func(t *testing.T) {
		// Create lost and found posts at the same location
		lostPost := CreateTestPostAt(t, TestLocations.EmpireState.Latitude, TestLocations.EmpireState.Longitude, "Lost Item at Empire State")
		defer CleanupPost(t, lostPost.ID)

		foundReq := CreatePostRequest{
			Title:        "Found Item at Empire State",
			Description:  "Found something at Empire State",
			Location:     TestLocations.EmpireState,
			RadiusMeters: 1000,
			Type:         "found",
		}
		foundPost := CreateTestPost(t, foundReq)
		defer CleanupPost(t, foundPost.ID)

		// Search for only lost items
		endpoint := fmt.Sprintf("/posts/nearby?lat=%f&lng=%f&radius=1000&type=lost",
			TestLocations.EmpireState.Latitude,
			TestLocations.EmpireState.Longitude)

		resp := makeRequest(t, "GET", endpoint, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var searchResp map[string]interface{}
		parseResponse(t, resp, &searchResp)

		posts := searchResp["posts"].([]interface{})

		// All returned posts should be lost type
		for _, p := range posts {
			post := p.(map[string]interface{})
			postType := post["type"].(string)
			require.Equal(t, "lost", postType, "All posts should be lost type")
		}

		// Search for only found items
		endpoint = fmt.Sprintf("/posts/nearby?lat=%f&lng=%f&radius=1000&type=found",
			TestLocations.EmpireState.Latitude,
			TestLocations.EmpireState.Longitude)

		resp = makeRequest(t, "GET", endpoint, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		parseResponse(t, resp, &searchResp)
		posts = searchResp["posts"].([]interface{})

		// All returned posts should be found type
		for _, p := range posts {
			post := p.(map[string]interface{})
			postType := post["type"].(string)
			require.Equal(t, "found", postType, "All posts should be found type")
		}
	})

	t.Run("should handle pagination", func(t *testing.T) {
		// Create multiple posts at the same location
		posts := make([]PostResponse, 5)
		for i := 0; i < 5; i++ {
			posts[i] = CreateTestPostAt(t,
				TestLocations.CentralPark.Latitude,
				TestLocations.CentralPark.Longitude,
				fmt.Sprintf("Central Park Post %d", i+1),
			)
			defer CleanupPost(t, posts[i].ID)
		}

		// Search with limit of 2
		endpoint := fmt.Sprintf("/posts/nearby?lat=%f&lng=%f&radius=1000&limit=2",
			TestLocations.CentralPark.Latitude,
			TestLocations.CentralPark.Longitude)

		resp := makeRequest(t, "GET", endpoint, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var searchResp map[string]interface{}
		parseResponse(t, resp, &searchResp)

		posts1 := searchResp["posts"].([]interface{})
		count := int(searchResp["count"].(float64))
		limit := int(searchResp["limit"].(float64))
		offset := int(searchResp["offset"].(float64))

		require.Equal(t, 2, count, "Should return 2 posts")
		require.Equal(t, 2, limit, "Limit should be 2")
		require.Equal(t, 0, offset, "Offset should be 0")

		// Search with offset
		endpoint = fmt.Sprintf("/posts/nearby?lat=%f&lng=%f&radius=1000&limit=2&offset=2",
			TestLocations.CentralPark.Latitude,
			TestLocations.CentralPark.Longitude)

		resp = makeRequest(t, "GET", endpoint, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		parseResponse(t, resp, &searchResp)
		posts2 := searchResp["posts"].([]interface{})
		offset = int(searchResp["offset"].(float64))

		require.Equal(t, 2, offset, "Offset should be 2")
		require.GreaterOrEqual(t, len(posts2), 1, "Should return more posts")

		// Posts should be different
		if len(posts1) > 0 && len(posts2) > 0 {
			post1 := posts1[0].(map[string]interface{})
			post2 := posts2[0].(map[string]interface{})
			require.NotEqual(t, post1["id"], post2["id"], "Posts should be different")
		}
	})

	t.Run("should validate search parameters", func(t *testing.T) {
		testCases := []struct {
			name     string
			endpoint string
		}{
			{"missing lat", "/posts/nearby?lng=-73.9665&radius=1000"},
			{"missing lng", "/posts/nearby?lat=40.7831&radius=1000"},
			{"invalid lat", "/posts/nearby?lat=invalid&lng=-73.9665&radius=1000"},
			{"invalid lng", "/posts/nearby?lat=40.7831&lng=invalid&radius=1000"},
			{"lat out of range", "/posts/nearby?lat=91.0&lng=-73.9665&radius=1000"},
			{"lng out of range", "/posts/nearby?lat=40.7831&lng=181.0&radius=1000"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				resp := makeRequest(t, "GET", tc.endpoint, nil)
				require.Equal(t, http.StatusBadRequest, resp.StatusCode)
			})
		}
	})

	t.Run("should use default radius when not specified", func(t *testing.T) {
		// Create a post
		post := CreateTestPostAt(t, TestLocations.CentralPark.Latitude, TestLocations.CentralPark.Longitude, "Central Park Post")
		defer CleanupPost(t, post.ID)

		// Search without radius parameter (should default to 1km)
		endpoint := fmt.Sprintf("/posts/nearby?lat=%f&lng=%f",
			TestLocations.CentralPark.Latitude,
			TestLocations.CentralPark.Longitude)

		resp := makeRequest(t, "GET", endpoint, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var searchResp map[string]interface{}
		parseResponse(t, resp, &searchResp)

		posts := searchResp["posts"].([]interface{})
		require.GreaterOrEqual(t, len(posts), 1, "Should find at least one post with default radius")
	})
}
