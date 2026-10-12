package planka_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vjanelle/go-planka/pkg/planka"
)

var _ = Describe("AuthClient", func() {
	It("creates a client with bearer token auth", func() {
		client, err := planka.AuthClient("https://example.com", "test-token")
		Expect(err).ToNot(HaveOccurred())
		Expect(client).ToNot(BeNil())
	})
})

var _ = Describe("APIKeyClient", func() {
	It("creates a client with API key auth", func() {
		client, err := planka.APIKeyClient("https://example.com", "test-key")
		Expect(err).ToNot(HaveOccurred())
		Expect(client).ToNot(BeNil())
	})
})

var _ = Describe("Login", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Expect(r.URL.Path).To(Equal("/api/access-tokens"))
			Expect(r.Method).To(Equal(http.MethodPost))

			var body map[string]string
			err := json.NewDecoder(r.Body).Decode(&body)
			Expect(err).ToNot(HaveOccurred())
			Expect(body["emailOrUsername"]).To(Equal("testuser"))
			Expect(body["password"]).To(Equal("testpass"))

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"item": "test-token-12345",
			})
		}))
	})

	AfterEach(func() {
		server.Close()
	})

	It("logs in and returns a token", func() {
		token, err := planka.Login(context.Background(), server.URL, "testuser", "testpass")
		Expect(err).ToNot(HaveOccurred())
		Expect(token).To(Equal("test-token-12345"))
	})

	It("returns an error on invalid credentials", func() {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Invalid credentials",
			})
		}))
		defer server.Close()

		_, err := planka.Login(context.Background(), server.URL, "testuser", "wrongpass")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("Client methods", func() {
	var server *httptest.Server
	var client *planka.ClientWithResponses
	ctx := context.Background()

	BeforeEach(func() {
		mux := http.NewServeMux()

		mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]string{
					{"id": "p1", "name": "Project One"},
					{"id": "p2", "name": "Project Two"},
				},
				"included": map[string]interface{}{
					"boards": []map[string]string{
						{"id": "b1", "name": "Board A", "projectId": "p1"},
					},
				},
			})
		})

		mux.HandleFunc("/api/boards/b1", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"item": map[string]string{
					"id": "b1", "name": "Board A", "projectId": "p1",
				},
				"included": map[string]interface{}{
					"lists": []map[string]interface{}{
						{"id": "l1", "name": "To Do", "type": "active"},
						{"id": "l2", "name": "Done", "type": "active"},
					},
				},
			})
		})

		mux.HandleFunc("/api/lists/l1/cards", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				var body map[string]string
				json.NewDecoder(r.Body).Decode(&body)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"item": map[string]string{
						"id": "c1", "name": body["name"], "listId": "l1", "boardId": "b1",
					},
				})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]string{
					{"id": "c1", "name": "Card 1", "listId": "l1"},
					{"id": "c2", "name": "Card 2", "listId": "l1"},
				},
			})
		})

		mux.HandleFunc("/api/cards/c1", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"item": map[string]string{"id": "c1", "name": "Card 1"},
				})
				return
			}
			if r.Method == http.MethodPatch {
				var body map[string]*string
				json.NewDecoder(r.Body).Decode(&body)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"item": map[string]string{"id": "c1", "name": *body["name"]},
				})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"item": map[string]string{"id": "c1", "name": "Card 1"},
			})
		})

		mux.HandleFunc("/api/cards/c1/comments", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				var body map[string]string
				json.NewDecoder(r.Body).Decode(&body)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"item": map[string]string{"id": "cm1", "cardId": "c1", "text": body["text"]},
				})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]string{
					{"id": "cm1", "cardId": "c1", "text": "First comment"},
					{"id": "cm2", "cardId": "c1", "text": "Second comment"},
				},
			})
		})

		mux.HandleFunc("/api/notifications", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{
					{"id": "n1", "type": "CARD_UPDATED", "isRead": false, "updatedAt": "2024-01-01T00:00:00Z"},
				},
			})
		})

		mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{
					{"id": "u1", "username": "admin", "email": "admin@example.com", "role": "admin"},
				},
			})
		})

		server = httptest.NewServer(mux)
		client, _ = planka.NewClientWithResponses(server.URL)
	})

	AfterEach(func() {
		server.Close()
	})

	It("gets projects", func() {
		resp, err := client.GetProjectsWithResponse(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.JSON200).ToNot(BeNil())
		Expect(len(resp.JSON200.Items)).To(Equal(2))
	})

	It("gets board", func() {
		resp, err := client.GetBoardWithResponse(ctx, "b1", nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.JSON200).ToNot(BeNil())
		Expect(resp.JSON200.Item.Name).To(Equal("Board A"))
		Expect(len(resp.JSON200.Included.Lists)).To(Equal(2))
	})

	It("gets cards", func() {
		resp, err := client.GetCardsWithResponse(ctx, "l1", nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.JSON200).ToNot(BeNil())
		Expect(len(resp.JSON200.Items)).To(Equal(2))
	})

	It("creates a card", func() {
		resp, err := client.CreateCardWithResponse(ctx, "l1", planka.CreateCardJSONRequestBody{
			Name: "New Card",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.JSON200).ToNot(BeNil())
		Expect(resp.JSON200.Item.Name).To(Equal("New Card"))
	})

	It("updates a card", func() {
		name := "Updated Card"
		resp, err := client.UpdateCardWithResponse(ctx, "c1", planka.UpdateCardJSONRequestBody{
			Name: &name,
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.JSON200).ToNot(BeNil())
		Expect(resp.JSON200.Item.Name).To(Equal("Updated Card"))
	})

	It("deletes a card", func() {
		resp, err := client.DeleteCardWithResponse(ctx, "c1")
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.JSON200).ToNot(BeNil())
	})

	It("gets comments", func() {
		resp, err := client.GetCommentsWithResponse(ctx, "c1", nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.JSON200).ToNot(BeNil())
		Expect(len(resp.JSON200.Items)).To(Equal(2))
	})

	It("creates a comment", func() {
		resp, err := client.CreateCommentWithResponse(ctx, "c1", planka.CreateCommentJSONRequestBody{
			Text: "Test comment",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.JSON200).ToNot(BeNil())
		Expect(resp.JSON200.Item.Id).To(Equal("cm1"))
	})

	It("gets notifications", func() {
		resp, err := client.GetNotificationsWithResponse(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.JSON200).ToNot(BeNil())
		Expect(len(resp.JSON200.Items)).To(Equal(1))
	})

	It("gets users", func() {
		resp, err := client.GetUsersWithResponse(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.JSON200).ToNot(BeNil())
		Expect(len(resp.JSON200.Items)).To(Equal(1))
		Expect(*resp.JSON200.Items[0].Username).To(Equal("admin"))
	})

	It("NewClient creates a client", func() {
		c, err := planka.NewClient(server.URL)
		Expect(err).ToNot(HaveOccurred())
		Expect(c).ToNot(BeNil())
	})

	It("NewClientWithResponses creates a client", func() {
		c, err := planka.NewClientWithResponses(server.URL)
		Expect(err).ToNot(HaveOccurred())
		Expect(c).ToNot(BeNil())
	})

	It("creates client with request editor", func() {
		editor := planka.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("X-Custom", "test")
			return nil
		})
		_, err := planka.NewClientWithResponses(server.URL, editor)
		Expect(err).ToNot(HaveOccurred())
	})
})

var _ = Describe("Client methods with auth", func() {
	var server *httptest.Server

	BeforeEach(func() {
		mux := http.NewServeMux()

		mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{
					{"id": "u1", "username": "admin"},
				},
			})
		})

		mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-Api-Key")
			if apiKey != "test-key" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]string{},
			})
		})

		server = httptest.NewServer(mux)
	})

	AfterEach(func() {
		server.Close()
	})

	It("AuthClient sets bearer token", func() {
		client, err := planka.AuthClient(server.URL, "test-token")
		Expect(err).ToNot(HaveOccurred())
		resp, err := client.GetUsersWithResponse(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.HTTPResponse.StatusCode).To(Equal(http.StatusOK))
	})

	It("APIKeyClient sets API key header", func() {
		client, err := planka.APIKeyClient(server.URL, "test-key")
		Expect(err).ToNot(HaveOccurred())
		resp, err := client.GetProjectsWithResponse(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.HTTPResponse.StatusCode).To(Equal(http.StatusOK))
	})

	It("handles 401 error", func() {
		client, err := planka.AuthClient(server.URL, "wrong-token")
		Expect(err).ToNot(HaveOccurred())
		resp, err := client.GetUsersWithResponse(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.HTTPResponse.StatusCode).To(Equal(http.StatusUnauthorized))
		Expect(resp.JSON200).To(BeNil())
	})
})

var _ = Describe("Client methods edge cases", func() {
	var server *httptest.Server
	var client *planka.ClientWithResponses
	ctx := context.Background()

	BeforeEach(func() {
		mux := http.NewServeMux()

		mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []interface{}{},
				"included": map[string]interface{}{
					"boards": []interface{}{},
				},
			})
		})

		mux.HandleFunc("/api/lists/l1/cards", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []interface{}{},
			})
		})

		mux.HandleFunc("/api/cards/c1/comments", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []interface{}{},
			})
		})

		mux.HandleFunc("/api/notifications", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []interface{}{},
			})
		})

		mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []interface{}{},
			})
		})

		server = httptest.NewServer(mux)
		client, _ = planka.NewClientWithResponses(server.URL)
	})

	AfterEach(func() {
		server.Close()
	})

	It("handles empty lists", func() {
		resp, err := client.GetProjectsWithResponse(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(resp.JSON200.Items)).To(Equal(0))

		resp2, err := client.GetCardsWithResponse(ctx, "l1", nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(resp2.JSON200.Items)).To(Equal(0))
	})

	It("handles nil list name", func() {
		mux2 := http.NewServeMux()
		mux2.HandleFunc("/api/boards/b1", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"item": map[string]interface{}{"id": "b1", "name": "Board"},
				"included": map[string]interface{}{
					"lists": []map[string]interface{}{
						{"id": "l1", "name": nil, "type": "active"},
					},
				},
			})
		})
		s2 := httptest.NewServer(mux2)
		defer s2.Close()

		c2, _ := planka.NewClientWithResponses(s2.URL)
		resp, err := c2.GetBoardWithResponse(ctx, "b1", nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.JSON200).ToNot(BeNil())
	})
})
