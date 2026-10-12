package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"

	"github.com/choria-io/fisk"
	"github.com/vjanelle/go-planka/pkg/planka"
	"github.com/zalando/go-keyring"
)

type Config struct {
	URL      string `json:"url"`
	Username string `json:"username"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".planka", "config.json")
}

func loadConfig() *Config {
	cfg := &Config{}
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, cfg)
	return cfg
}

func saveConfig(cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(configPath()), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0o600)
}

func keychainGet(key string) (string, error) {
	return keyring.Get("planka", key)
}

func keychainSet(key, value string) error {
	return keyring.Set("planka", key, value)
}

func keychainClear() error {
	for _, key := range []string{"url", "token", "api_key", "user_id"} {
		keyring.Delete("planka", key)
	}
	return nil
}

func getClient(cfg *Config) *planka.ClientWithResponses {
	url := cfg.URL
	token, _ := keychainGet("token")
	apiKey, _ := keychainGet("api_key")

	var c *planka.ClientWithResponses
	var err error
	if token != "" {
		c, err = planka.AuthClient(url, token)
	} else if apiKey != "" {
		c, err = planka.APIKeyClient(url, apiKey)
	} else {
		c, err = planka.NewClientWithResponses(url + "/api")
	}
	if err != nil {
		fisk.Fatalf("error creating client: %v", err)
	}
	return c
}

func main() {
	ctx := context.Background()

	app := fisk.New("planka", "Planka Kanban CLI")
	app.Version("0.1.0")
	app.UsageWriter(os.Stdout)

	cfg := loadConfig()

	// ---- auth ----
	authCmd := app.Command("auth", "Authentication")

	var loginURL, loginUser, loginPass string
	loginCmd := authCmd.Command("login", "Login with username and password")
	loginCmd.Arg("url", "Planka URL").Required().StringVar(&loginURL)
	loginCmd.Arg("username", "Username").Required().StringVar(&loginUser)
	loginCmd.Arg("password", "Password").Required().StringVar(&loginPass)
	loginCmd.Action(func(_ *fisk.ParseContext) error {
		token, err := planka.Login(ctx, loginURL, loginUser, loginPass)
		if err != nil {
			return err
		}
		cfg.URL = loginURL
		cfg.Username = loginUser
		keychainSet("url", loginURL)
		keychainSet("token", token)
		return saveConfig(cfg)
	})

	var apiKeyURL, apiKey string
	apiKeyCmd := authCmd.Command("api-key", "Use API key")
	apiKeyCmd.Arg("url", "Planka URL").Required().StringVar(&apiKeyURL)
	apiKeyCmd.Arg("key", "API key").Required().StringVar(&apiKey)
	apiKeyCmd.Action(func(_ *fisk.ParseContext) error {
		cfg.URL = apiKeyURL
		keychainSet("url", apiKeyURL)
		keychainSet("api_key", apiKey)
		keychainSet("token", "")
		return saveConfig(cfg)
	})

	authCmd.Command("whoami", "Show current user").Action(func(_ *fisk.ParseContext) error {
		client := getClient(cfg)
		resp, err := client.GetUsersWithResponse(ctx)
		if err != nil {
			return err
		}
		if resp.JSON200 == nil {
			return fmt.Errorf("not authenticated: %s", string(resp.Body))
		}
		userID, _ := keychainGet("user_id")
		for _, u := range resp.JSON200.Items {
			if u.Id == userID {
				if u.Username != nil {
					cfg.Username = *u.Username
					keychainSet("user_id", u.Id)
				}
				return saveConfig(cfg)
			}
		}
		fmt.Printf("Username: %s\n", cfg.Username)
		return nil
	})

	authCmd.Command("logout", "Clear authentication").Action(func(_ *fisk.ParseContext) error {
		if err := keychainClear(); err != nil {
			return err
		}
		cfg.URL = ""
		cfg.Username = ""
		return saveConfig(cfg)
	})

	// ---- projects ----
	projectsCmd := app.Command("projects", "Manage projects")

	projectsCmd.Command("list", "List projects").Action(func(_ *fisk.ParseContext) error {
		client := getClient(cfg)
		resp, err := client.GetProjectsWithResponse(ctx)
		if err != nil {
			return err
		}
		if resp.JSON200 == nil {
			return fmt.Errorf("error: %s", string(resp.Body))
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID	NAME\n")
		for _, p := range resp.JSON200.Items {
			fmt.Fprintf(w, "%s	%s\n", p.Id, p.Name)
		}
		return w.Flush()
	})

	// ---- boards ----
	boardsCmd := app.Command("boards", "Manage boards")

	var boardListProject string
	boardListCmd := boardsCmd.Command("list", "List boards")
	boardListCmd.Arg("project", "Filter by project").StringVar(&boardListProject)
	boardListCmd.Action(func(_ *fisk.ParseContext) error {
		client := getClient(cfg)
		resp, err := client.GetProjectsWithResponse(ctx)
		if err != nil {
			return err
		}
		if resp.JSON200 == nil {
			return fmt.Errorf("error: %s", string(resp.Body))
		}
		rows := make([][]string, 0)
		for _, b := range resp.JSON200.Included.Boards {
			if boardListProject != "" && b.ProjectId != boardListProject {
				continue
			}
			rows = append(rows, []string{b.Id, b.Name, b.ProjectId})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i][1] < rows[j][1] })
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID	NAME	PROJECT\n")
		for _, row := range rows {
			fmt.Fprintf(w, "%s	%s	%s\n", row[0], row[1], row[2])
		}
		return w.Flush()
	})

	var boardID string
	boardGetCmd := boardsCmd.Command("get", "Get board details")
	boardGetCmd.Arg("board", "Board ID").Required().StringVar(&boardID)
	boardGetCmd.Action(func(_ *fisk.ParseContext) error {
		client := getClient(cfg)
		resp, err := client.GetBoardWithResponse(ctx, boardID, nil)
		if err != nil {
			return err
		}
		if resp.JSON200 == nil {
			return fmt.Errorf("error: %s", string(resp.Body))
		}
		b := resp.JSON200.Item
		fmt.Printf("Board: %s (project: %s)\n", b.Name, b.ProjectId)
		fmt.Printf("\nLists:\n")
		for _, l := range resp.JSON200.Included.Lists {
			name := "???"
			if l.Name != nil {
				name = *l.Name
			}
			fmt.Printf("  %s  %s (%s)\n", l.Id, name, l.Type)
		}
		return nil
	})

	// ---- lists ----
	listsCmd := app.Command("lists", "Manage lists")

	var listsBoard string
	listsListCmd := listsCmd.Command("list", "List lists in a board")
	listsListCmd.Arg("board", "Board ID").Required().StringVar(&listsBoard)
	listsListCmd.Action(func(_ *fisk.ParseContext) error {
		client := getClient(cfg)
		resp, err := client.GetBoardWithResponse(ctx, listsBoard, nil)
		if err != nil {
			return err
		}
		if resp.JSON200 == nil {
			return fmt.Errorf("error: %s", string(resp.Body))
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID	NAME	TYPE\n")
		for _, l := range resp.JSON200.Included.Lists {
			name := "???"
			if l.Name != nil {
				name = *l.Name
			}
			fmt.Fprintf(w, "%s	%s	%s\n", l.Id, name, l.Type)
		}
		return w.Flush()
	})

	// ---- cards ----
	cardsCmd := app.Command("cards", "Manage cards")

	var cardsList string
	cardsListCmd := cardsCmd.Command("list", "List cards in a list")
	cardsListCmd.Arg("list", "List ID").Required().StringVar(&cardsList)
	cardsListCmd.Action(func(_ *fisk.ParseContext) error {
		client := getClient(cfg)
		resp, err := client.GetCardsWithResponse(ctx, cardsList, nil)
		if err != nil {
			return err
		}
		if resp.JSON200 == nil {
			return fmt.Errorf("error: %s", string(resp.Body))
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID	NAME\n")
		for _, c := range resp.JSON200.Items {
			fmt.Fprintf(w, "%s	%s\n", c.Id, c.Name)
		}
		return w.Flush()
	})

	var cardCreateList, cardCreateName string
	cardCreateCmd := cardsCmd.Command("create", "Create card")
	cardCreateCmd.Arg("list", "List ID").Required().StringVar(&cardCreateList)
	cardCreateCmd.Arg("name", "Card name").Required().StringVar(&cardCreateName)
	cardCreateCmd.Action(func(_ *fisk.ParseContext) error {
		client := getClient(cfg)
		resp, err := client.CreateCardWithResponse(ctx, cardCreateList, planka.CreateCardJSONRequestBody{
			Name: cardCreateName,
		})
		if err != nil {
			return err
		}
		if resp.JSON200 == nil {
			return fmt.Errorf("error: %s", string(resp.Body))
		}
		fmt.Printf("Created: %s (%s)\n", resp.JSON200.Item.Name, resp.JSON200.Item.Id)
		return nil
	})

	var cardUpdateID string
	var cardUpdateName, cardUpdateList, cardUpdateDesc string
	cardUpdateCmd := cardsCmd.Command("update", "Update card")
	cardUpdateCmd.Arg("card", "Card ID").Required().StringVar(&cardUpdateID)
	cardUpdateCmd.Flag("name", "Card name").StringVar(&cardUpdateName)
	cardUpdateCmd.Flag("list", "Move to list").StringVar(&cardUpdateList)
	cardUpdateCmd.Flag("description", "Card description").StringVar(&cardUpdateDesc)
	cardUpdateCmd.Action(func(_ *fisk.ParseContext) error {
		client := getClient(cfg)
		body := planka.UpdateCardJSONRequestBody{}
		if cardUpdateName != "" {
			body.Name = &cardUpdateName
		}
		if cardUpdateList != "" {
			body.ListId = &cardUpdateList
		}
		if cardUpdateDesc != "" {
			body.Description = &cardUpdateDesc
		}
		resp, err := client.UpdateCardWithResponse(ctx, cardUpdateID, body)
		if err != nil {
			return err
		}
		if resp.JSON200 == nil {
			return fmt.Errorf("error: %s", string(resp.Body))
		}
		fmt.Printf("Updated: %s (%s)\n", resp.JSON200.Item.Name, resp.JSON200.Item.Id)
		return nil
	})

	var cardDeleteID string
	cardDeleteCmd := cardsCmd.Command("delete", "Delete card")
	cardDeleteCmd.Arg("card", "Card ID").Required().StringVar(&cardDeleteID)
	cardDeleteCmd.Action(func(_ *fisk.ParseContext) error {
		client := getClient(cfg)
		resp, err := client.DeleteCardWithResponse(ctx, cardDeleteID)
		if err != nil {
			return err
		}
		if resp.JSON200 == nil {
			return fmt.Errorf("error: %s", string(resp.Body))
		}
		fmt.Printf("Deleted: %s\n", cardDeleteID)
		return nil
	})

	// ---- comments ----
	commentsCmd := app.Command("comments", "Manage comments")

	var commentsCard string
	commentsListCmd := commentsCmd.Command("list", "List comments")
	commentsListCmd.Arg("card", "Card ID").Required().StringVar(&commentsCard)
	commentsListCmd.Action(func(_ *fisk.ParseContext) error {
		client := getClient(cfg)
		resp, err := client.GetCommentsWithResponse(ctx, commentsCard, nil)
		if err != nil {
			return err
		}
		if resp.JSON200 == nil {
			return fmt.Errorf("error: %s", string(resp.Body))
		}
		for _, c := range resp.JSON200.Items {
			userID := "unknown"
			if c.UserId != nil {
				userID = *c.UserId
			}
			fmt.Printf("[%s] %s\n", userID, c.Text)
		}
		return nil
	})

	var commentCard, commentText string
	commentCreateCmd := commentsCmd.Command("create", "Create comment")
	commentCreateCmd.Arg("card", "Card ID").Required().StringVar(&commentCard)
	commentCreateCmd.Arg("text", "Comment text").Required().StringVar(&commentText)
	commentCreateCmd.Action(func(_ *fisk.ParseContext) error {
		client := getClient(cfg)
		resp, err := client.CreateCommentWithResponse(ctx, commentCard, planka.CreateCommentJSONRequestBody{
			Text: commentText,
		})
		if err != nil {
			return err
		}
		if resp.JSON200 == nil {
			return fmt.Errorf("error: %s", string(resp.Body))
		}
		fmt.Printf("Created: %s\n", resp.JSON200.Item.Id)
		return nil
	})

	// ---- users ----
	usersCmd := app.Command("users", "Manage users")

	usersCmd.Command("list", "List users").Action(func(_ *fisk.ParseContext) error {
		client := getClient(cfg)
		resp, err := client.GetUsersWithResponse(ctx)
		if err != nil {
			return err
		}
		if resp.JSON200 == nil {
			return fmt.Errorf("error: %s", string(resp.Body))
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID	USERNAME	EMAIL	ROLE\n")
		for _, u := range resp.JSON200.Items {
			username := "???"
			if u.Username != nil {
				username = *u.Username
			}
			email := "???"
			if u.Email != nil {
				email = string(*u.Email)
			}
			fmt.Fprintf(w, "%s	%s	%s	%s\n", u.Id, username, email, u.Role)
		}
		return w.Flush()
	})

	// ---- notifications ----
	notifsCmd := app.Command("notifications", "Manage notifications")

	notifListCmd := notifsCmd.Command("list", "List notifications")
	notifListCmd.Action(func(_ *fisk.ParseContext) error {
		client := getClient(cfg)
		resp, err := client.GetNotificationsWithResponse(ctx)
		if err != nil {
			return err
		}
		if resp.JSON200 == nil {
			return fmt.Errorf("error: %s", string(resp.Body))
		}
		for _, n := range resp.JSON200.Items {
			status := "read"
			if !n.IsRead {
				status = "unread"
			}
			fmt.Printf("[%s] %s: %s (%s)\n", n.Id, status, n.Type, n.UpdatedAt)
		}
		return nil
	})

	_, err := app.Parse(os.Args[1:])
	if err != nil {
		fisk.Fatalf("Error: %v", err)
	}
}
