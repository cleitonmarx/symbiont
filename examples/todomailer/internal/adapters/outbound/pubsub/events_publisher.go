package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"cloud.google.com/go/pubsub/v2"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/tracing"
)

// TodoEventPublisher implements domain.TodoEventPublisher using Google Cloud Pub/Sub.
type TodoEventPublisher struct {
	client  *pubsub.Client
	topicID string
}

// NewTodoEventPublisher creates a new TodoEventPublisher instance.
func NewTodoEventPublisher(client *pubsub.Client, topicID string) *TodoEventPublisher {
	return &TodoEventPublisher{
		client:  client,
		topicID: topicID,
	}
}

// PublishEvent publishes a todo event to the pub/sub topic.
func (p *TodoEventPublisher) PublishEvent(ctx context.Context, event domain.TodoEvent) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	result := p.client.Publisher(p.topicID).Publish(spanCtx, &pubsub.Message{
		Data: eventData,
		Attributes: map[string]string{
			"event_type": string(event.Type),
			"todo_id":    event.TodoID.String(),
		},
	})

	// Block until the result is returned
	_, err = result.Get(spanCtx)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

type InitClient struct {
	Logger    *log.Logger `resolve:""`
	ProjectID string      `config:"PUBSUB_PROJECT_ID"`
	client    *pubsub.Client
}

func (i *InitClient) Initialize(ctx context.Context) (context.Context, error) {
	client, err := pubsub.NewClient(ctx, i.ProjectID)
	if err != nil {
		return ctx, fmt.Errorf("failed to create pubsub client: %w", err)
	}
	i.client = client

	depend.Register[*pubsub.Client](client)

	return ctx, nil
}

func (i *InitClient) Close() {
	if err := i.client.Close(); err != nil {
		i.Logger.Printf("InitClient:failed to close pubsub client: %v", err)
	}
}

// InitTodoEventPublisher is the initializer for TodoEventPublisher.
type InitTodoEventPublisher struct {
	Logger    *log.Logger    `resolve:""`
	ProjectID string         `config:"PUBSUB_PROJECT_ID"`
	TopicID   string         `config:"PUBSUB_TOPIC_ID" default:"todo-events"`
	Client    *pubsub.Client `resolve:""`
}

// Initialize registers the TodoEventPublisher in the dependency container.
func (i *InitTodoEventPublisher) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.TodoEventPublisher](NewTodoEventPublisher(i.Client, i.TopicID))

	return ctx, nil
}
