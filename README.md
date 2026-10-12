# go-planka

A Go client library and CLI for [Planka](https://plankanban.github.io/planka/) Kanban boards.

## Library

The client library is generated from Planka's OpenAPI specification using [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen).

### Installation

```bash
go get github.com/vjanelle/go-planka
```

### Usage

```go
import (
    "context"
    "github.com/vjanelle/go-planka/pkg/planka"
)

// Authenticate with email/password
client, err := planka.AuthClient("https://your-planka-instance.com", "your-token")
if err != nil {
    panic(err)
}

// Use API key authentication
client, err := planka.APIKeyClient("https://your-planka-instance.com", "your-api-key")
if err != nil {
    panic(err)
}

// List projects
ctx := context.Background()
resp, err := client.GetProjectsWithResponse(ctx)
if err != nil {
    panic(err)
}
for _, project := range resp.JSON200.Items {
    fmt.Println(project.Name)
}

// List boards
resp, err = client.GetProjectsWithResponse(ctx)
for _, board := range resp.JSON200.Included.Boards {
    fmt.Println(board.Name)
}

// Get cards from a list
resp, err = client.GetCardsWithResponse(ctx, "list-id", nil)
for _, card := range resp.JSON200.Items {
    fmt.Println(card.Name)
}

// Create a card
resp, err = client.CreateCardWithResponse(ctx, "list-id", planka.CreateCardJSONRequestBody{
    Name: "My new card",
})
fmt.Println(resp.JSON200.Item.Id)

// Update a card
name := "Updated name"
resp, err = client.UpdateCardWithResponse(ctx, "card-id", planka.UpdateCardJSONRequestBody{
    Name: &name,
})

// Delete a card
resp, err = client.DeleteCardWithResponse(ctx, "card-id")
```

## CLI

The CLI is built with [fisk](https://github.com/choria-io/fisk).

### Installation

```bash
go install github.com/vjanelle/go-planka/cmd/planka
```

### Usage

```bash
# Login
planka auth login https://your-planka-instance.com admin admin

# Or use API key
planka auth api-key https://your-planka-instance.com your-api-key

# List projects
planka projects list

# List boards
planka boards list
planka boards list p1  # filter by project

# Get board details
planka boards get b1

# List lists
planka lists list b1

# List cards
planka cards list l1

# Create card
planka cards create l1 "My new card"

# Update card
planka cards update c1 --name "Updated name"
planka cards update c1 --list l2  # move to another list

# Delete card
planka cards delete c1

# Comments
planka comments list c1
planka comments create c1 "My comment"

# Users
planka users list

# Notifications
planka notifications list
```

### Configuration

The CLI stores credentials in `~/.planka/config.json`.

## Development

### Regenerating the client

```bash
oapi-codegen --package planka --generate types,client -o pkg/planka/planka_api.go swagger.json
```

### Running tests

```bash
ginkgo run ./pkg/...
```

## License

MIT
