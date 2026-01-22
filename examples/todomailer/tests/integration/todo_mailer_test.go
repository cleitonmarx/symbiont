//------go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	httpapi "github.com/cleitonmarx/symbiont/examples/todomailer/internal/adapters/inbound/http"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/app"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/usecases"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
)

func TestTodoMailer_Integration(t *testing.T) {
	cleanupEnv := setTestEnvVars()
	defer cleanupEnv()

	todoMailerApp := app.NewTodoMailerApp(
		&InitDockerCompose{},
	)

	queue := make(usecases.CompletedTodoEmailQueue, 5)
	depend.Register(queue)

	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownCh := todoMailerApp.RunAsync(cancelCtx)

	err := todoMailerApp.WaitForReadiness(cancelCtx, 20000000*time.Second)
	if err != nil {
		cancel()
		t.Fatalf("TodoMailer app failed to become ready: %v", err)
	}

	apiCli, err := httpapi.NewClientWithResponses("http://localhost:8080")
	assert.NoError(t, err, "failed to create TodoMailer API client")

	t.Run("create-todos", func(t *testing.T) {
		for i := range 5 {
			createResp, err := apiCli.CreateTodoWithResponse(cancelCtx, httpapi.CreateTodoJSONRequestBody{
				Title:   fmt.Sprintf("Test Todo %d", i+1),
				DueDate: types.Date{Time: time.Now().Add(24 * time.Hour)},
			})
			assert.NoError(t, err, "failed to call CreateTodo endpoint")
			assert.NotNil(t, createResp.JSON201, "expected non-nil response for CreateTodo")
		}
	})

	var todos []httpapi.Todo
	t.Run("list-created-todos", func(t *testing.T) {
		resp, err := apiCli.ListTodosWithResponse(cancelCtx, &httpapi.ListTodosParams{
			Page:     1,
			Pagesize: 10,
		})

		assert.NoError(t, err, "failed to call ListTodos endpoint")
		assert.NotNil(t, resp.JSON200, "expected non-nil response for ListTodos")
		assert.Equal(t, 5, len(resp.JSON200.Items), "expected 5 todos in the list")

		todos = resp.JSON200.Items
	})

	t.Run("update-todos-and-check-emails", func(t *testing.T) {
		statusDone := httpapi.DONE
		for _, todo := range todos {
			updateResp, err := apiCli.UpdateTodoWithResponse(cancelCtx, todo.Id, httpapi.UpdateTodoJSONRequestBody{
				Status: &statusDone,
			})
			assert.NoError(t, err, "failed to call UpdateTodo endpoint")
			assert.NotNil(t, updateResp.JSON200, "expected non-nil response for UpdateTodo")
			assert.Equal(t, httpapi.DONE, updateResp.JSON200.Status, "expected todo status to be 'completed'")
			select {
			case emailedTodo := <-queue:
				assert.Equal(t, todo.Id, types.UUID(emailedTodo.ID), "expected emailed todo ID to match updated todo ID")
				assert.Equal(t, domain.EmailStatus_SENT, emailedTodo.EmailStatus, "expected emailed todo status to be 'SENT'")
			case <-time.After(5 * time.Second):
				t.Fatalf("Timed out waiting for emailed todo in queue for todo ID: %s", todo.Id.String())
			}
		}
	})

	cancel()

	select {
	case <-time.After(10 * time.Second):
		t.Fatalf("TodoMailer app did not shut down in time")
	case err = <-shutdownCh:
		if err != nil {
			t.Fatalf("TodoMailer app shutdown with error: %v", err)
		} else {
			t.Logf("TodoMailer app shut down gracefully")
		}
	}
}

func setTestEnvVars() func() {
	// Set any necessary environment variables for the integration tests here.
	os.Setenv("VAULT_ADDR", "http://localhost:8200")
	os.Setenv("VAULT_TOKEN", "root-token")
	os.Setenv("VAULT_MOUNT_PATH", "secret")
	os.Setenv("VAULT_SECRET_PATH", "todomailer")
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_NAME", "todomailerdb")
	os.Setenv("EMAIL_SENDER_INTERVAL", "1s")
	os.Setenv("PUBSUB_EMULATOR_HOST", "localhost:8681")
	os.Setenv("PUBSUB_PROJECT_ID", "local-dev")
	os.Setenv("PUBSUB_TOPIC_ID", "Todo")
	os.Setenv("PUBSUB_SUBSCRIPTION_ID", "todo_summary_generator")
	os.Setenv("DOCKER_MODEL_HOST", "http://localhost:12434")

	return func() {
		// Unset the environment variables after the test.
		os.Unsetenv("VAULT_ADDR")
		os.Unsetenv("VAULT_TOKEN")
		os.Unsetenv("VAULT_MOUNT_PATH")
		os.Unsetenv("VAULT_SECRET_PATH")
		os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("EMAIL_SENDER_INTERVAL")
		os.Unsetenv("PUBSUB_EMULATOR_HOST")
		os.Unsetenv("PUBSUB_PROJECT_ID")
		os.Unsetenv("PUBSUB_TOPIC_ID")
		os.Unsetenv("PUBSUB_SUBSCRIPTION_ID")
		os.Unsetenv("DOCKER_MODEL_HOST")
	}
}
