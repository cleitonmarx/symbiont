# TodoApp - A Complete Symbiont Example

TodoApp is a comprehensive example application demonstrating the capabilities of Symbiont. It's a full-stack todo application with AI-powered board summaries and event-driven architecture.

## Features

- 📝 **Todo Management**: Create, update, and track todos with due dates
- 🤖 **AI Chat & Summaries**: Interactive LLM-powered chat that streams responses and generates intelligent board summaries from your current todos
- 🔔 **Event-Driven**: Pub/Sub architecture for asynchronous processing
- 🔒 **Secrets Management**: HashiCorp Vault integration for secure configuration
- 📊 **Observability**: OpenTelemetry tracing with Jaeger
- 🗄️ **PostgreSQL**: Persistent storage with migrations
- 🎨 **Modern UI**: React + TypeScript frontend

## Architecture

### Components

- **HTTP API**: RESTful API for todo operations (OpenAPI 3.0)
- **Worker**: Background worker for summary generation
- **PostgreSQL**: Database for todos and board summaries
- **Pub/Sub Emulator**: Google Cloud Pub/Sub emulator for event messaging
- **Vault**: Secret management for sensitive configuration
- **Docker Model Runner**: Local LLM inference for board summary generation
- **Jaeger**: Distributed tracing and monitoring using OpenTelemetry

### 🔗 Dependency Graph (Initializers, Runners, Dependencies, Configs)

```mermaid
graph TD
 	usecases_GetBoardSummaryImpl["<b><span style='font-size:16px'>usecases.GetBoardSummary</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.GetBoardSummaryImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitGetBoardSummary.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/get_board_summary.go:46)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
 	usecases_CreateTodoImpl["<b><span style='font-size:16px'>usecases.CreateTodo</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.CreateTodoImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitCreateTodo.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/create_todo.go:94)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
 	usecases_UpdateTodoImpl["<b><span style='font-size:16px'>usecases.UpdateTodo</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.UpdateTodoImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitUpdateTodo.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/update_todo.go:113)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
 	usecases_CompletedSummaryQueue["<b><span style='font-size:16px'>usecases.CompletedSummaryQueue</span></b>"]
 	LLM_MODEL_API_KEY["<b><span style='font-size:16px'>LLM_MODEL_API_KEY</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
 	DB_HOST["<b><span style='font-size:16px'>DB_HOST</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
 	FETCH_OUTBOX_INTERVAL["<b><span style='font-size:16px'>FETCH_OUTBOX_INTERVAL</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
 	PUBSUB_TOPIC_ID["<b><span style='font-size:16px'>PUBSUB_TOPIC_ID</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
 	SUMMARY_BATCH_SIZE["<b><span style='font-size:16px'>SUMMARY_BATCH_SIZE</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
 	llm_BoardSummaryGenerator["<b><span style='font-size:16px'>domain.BoardSummaryGenerator</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 llm.BoardSummaryGenerator</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ llm.InitBoardSummaryGenerator.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(llm/board_summary_generator.go:225)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
 	DB_PASS["<b><span style='font-size:16px'>DB_PASS</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 config.VaultProvider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
 	DB_PORT["<b><span style='font-size:16px'>DB_PORT</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
 	DB_USER["<b><span style='font-size:16px'>DB_USER</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 config.VaultProvider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
 	HTTP_PORT["<b><span style='font-size:16px'>HTTP_PORT</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
 	PUBSUB_SUBSCRIPTION_ID["<b><span style='font-size:16px'>PUBSUB_SUBSCRIPTION_ID</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
 	SUMMARY_BATCH_INTERVAL["<b><span style='font-size:16px'>SUMMARY_BATCH_INTERVAL</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
 	PUBSUB_PROJECT_ID["<b><span style='font-size:16px'>PUBSUB_PROJECT_ID</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
 	ptr_postgres_UnitOfWork["<b><span style='font-size:16px'>domain.UnitOfWork</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 *postgres.UnitOfWork</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ postgres.InitUnitOfWork.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(postgres/unit_work.go:74)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
 	postgres_BoardSummaryRepository["<b><span style='font-size:16px'>domain.BoardSummaryRepository</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 postgres.BoardSummaryRepository</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ postgres.InitBoardSummaryRepository.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(postgres/board_summary_repository.go:129)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
 	time_CurrentTimeProvider["<b><span style='font-size:16px'>domain.CurrentTimeProvider</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 time.CurrentTimeProvider</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ time.InitCurrentTimeProvider.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(time/current_time_provider.go:25)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
 	LLM_MODEL_HOST["<b><span style='font-size:16px'>LLM_MODEL_HOST</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
 	ptr_http_Client["<b><span style='font-size:16px'>*http.Client</span></b><br/><span style='color:darkblue;font-size:11px;'>🏗️ tracing.InitHttpClient.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(tracing/tracing.go:153)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
 	ptr_sql_DB["<b><span style='font-size:16px'>*sql.DB</span></b><br/><span style='color:darkblue;font-size:11px;'>🏗️ postgres.(*InitDB).Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(postgres/init_db.go:60)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
 	usecases_ListTodosImpl["<b><span style='font-size:16px'>usecases.ListTodos</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.ListTodosImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitListTodos.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/list_todos.go:47)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
 	DB_NAME["<b><span style='font-size:16px'>DB_NAME</span></b><br/><span style='color:green;font-size:11px;'>🫴🏽 config.EnvVarProvider</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
 	LLM_MODEL["<b><span style='font-size:16px'>LLM_MODEL</span></b><br/><span style='color:green;font-size:11px;'>default</span><br/><span style='color:green;font-size:11px;'>🔑 <b>Config</b></span>"]
 	ptr_log_Logger["<b><span style='font-size:16px'>*log.Logger</span></b><br/><span style='color:darkblue;font-size:11px;'>🏗️ log.InitLogger.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(log/logger.go:16)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
 	postgres_TodoRepository["<b><span style='font-size:16px'>domain.TodoRepository</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 postgres.TodoRepository</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ postgres.InitTodoRepository.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(postgres/todo_repository.go:187)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
 	ptr_pubsub_Client["<b><span style='font-size:16px'>*pubsub.Client</span></b><br/><span style='color:darkblue;font-size:11px;'>🏗️ pubsub.(*InitClient).Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(pubsub/client.go:25)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
 	usecases_GenerateBoardSummaryImpl["<b><span style='font-size:16px'>usecases.GenerateBoardSummary</span></b><br/><span style='color:darkgray;font-size:11px;'>🧩 usecases.GenerateBoardSummaryImpl</span><br/><span style='color:darkblue;font-size:11px;'>🏗️ usecases.InitGenerateBoardSummary.Initialize</span><br/><span style='color:gray;font-size:11px;'>📍(usecases/generate_board_summary.go:80)</span><br/><span style='color:green;font-size:11px;'>💉 <b>Dependency</b></span>"]
 	ptr_postgres_InitTodoRepository["<b><span style='font-size:15px'>*postgres.InitTodoRepository</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
 	ptr_usecases_InitGenerateBoardSummary["<b><span style='font-size:15px'>*usecases.InitGenerateBoardSummary</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
 	ptr_tracing_InitOpenTelemetry["<b><span style='font-size:15px'>*tracing.InitOpenTelemetry</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
 	ptr_config_InitVaultProvider["<b><span style='font-size:16px'>*config.InitVaultProvider</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
 	ptr_usecases_InitCreateTodo["<b><span style='font-size:15px'>*usecases.InitCreateTodo</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
 	ptr_llm_InitBoardSummaryGenerator["<b><span style='font-size:15px'>*llm.InitBoardSummaryGenerator</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
 	ptr_pubsub_InitClient["<b><span style='font-size:15px'>*pubsub.InitClient</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
 	ptr_log_InitLogger["<b><span style='font-size:16px'>*log.InitLogger</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
 	ptr_postgres_InitUnitOfWork["<b><span style='font-size:15px'>*postgres.InitUnitOfWork</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
 	ptr_postgres_InitBoardSummaryRepository["<b><span style='font-size:15px'>*postgres.InitBoardSummaryRepository</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
 	ptr_postgres_InitDB["<b><span style='font-size:15px'>*postgres.InitDB</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
 	ptr_time_InitCurrentTimeProvider["<b><span style='font-size:16px'>*time.InitCurrentTimeProvider</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
 	ptr_usecases_InitUpdateTodo["<b><span style='font-size:15px'>*usecases.InitUpdateTodo</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
 	ptr_usecases_InitGetBoardSummary["<b><span style='font-size:15px'>*usecases.InitGetBoardSummary</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
 	ptr_tracing_InitHttpClient["<b><span style='font-size:16px'>*tracing.InitHttpClient</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
 	ptr_usecases_InitListTodos["<b><span style='font-size:15px'>*usecases.InitListTodos</span></b><br/><span style='color:green;font-size:11px;'>📦 <b>Initializer</b></span>"]
 	ptr_app_ReportLoggerIntrospector["<b><span style='font-size:15px'>*app.ReportLoggerIntrospector</span></b><br/><span style='color:gray;font-size:11px;'>📍(build/symbiont.go:93)</span>"]
 	ptr_worker_TodoEventSubscriber["<b><span style='font-size:16px'>*worker.TodoEventSubscriber</span></b><br/><span style='color:green;font-size:11px;'>⚙️ <b>Runnable</b></span>"]
 	ptr_http_TodoMailerServer["<b><span style='font-size:16px'>*http.TodoMailerServer</span></b><br/><span style='color:green;font-size:11px;'>⚙️ <b>Runnable</b></span>"]
 	ptr_worker_OutboxPublisher["<b><span style='font-size:16px'>*worker.OutboxPublisher</span></b><br/><span style='color:green;font-size:11px;'>⚙️ <b>Runnable</b></span>"]
 	SymbiontApp["<b><span style='font-size:20px;color:white'>🚀 Symbiont App</span></b>"]
     ptr_http_Client -.-> ptr_llm_InitBoardSummaryGenerator
     ptr_http_TodoMailerServer --- SymbiontApp
     ptr_llm_InitBoardSummaryGenerator --o llm_BoardSummaryGenerator
     ptr_log_InitLogger --o ptr_log_Logger
     ptr_log_Logger -.-> ptr_app_ReportLoggerIntrospector
     ptr_log_Logger -.-> ptr_http_TodoMailerServer
     ptr_log_Logger -.-> ptr_postgres_InitDB
     ptr_log_Logger -.-> ptr_pubsub_InitClient
     ptr_log_Logger -.-> ptr_tracing_InitOpenTelemetry
     ptr_log_Logger -.-> ptr_worker_OutboxPublisher
     ptr_log_Logger -.-> ptr_worker_TodoEventSubscriber
     ptr_postgres_InitBoardSummaryRepository --o postgres_BoardSummaryRepository
     ptr_postgres_InitDB --o ptr_sql_DB
     ptr_postgres_InitTodoRepository --o postgres_TodoRepository
     ptr_postgres_InitUnitOfWork --o ptr_postgres_UnitOfWork
     ptr_postgres_UnitOfWork -.-> ptr_usecases_InitCreateTodo
     ptr_postgres_UnitOfWork -.-> ptr_usecases_InitUpdateTodo
     ptr_pubsub_Client -.-> ptr_worker_OutboxPublisher
     ptr_pubsub_Client -.-> ptr_worker_TodoEventSubscriber
     ptr_pubsub_InitClient --o ptr_pubsub_Client
     ptr_sql_DB -.-> ptr_postgres_InitBoardSummaryRepository
     ptr_sql_DB -.-> ptr_postgres_InitTodoRepository
     ptr_sql_DB -.-> ptr_postgres_InitUnitOfWork
     ptr_sql_DB -.-> ptr_worker_OutboxPublisher
     ptr_time_InitCurrentTimeProvider --o time_CurrentTimeProvider
     ptr_tracing_InitHttpClient --o ptr_http_Client
     ptr_usecases_InitCreateTodo --o usecases_CreateTodoImpl
     ptr_usecases_InitGenerateBoardSummary --o usecases_GenerateBoardSummaryImpl
     ptr_usecases_InitGetBoardSummary --o usecases_GetBoardSummaryImpl
     ptr_usecases_InitListTodos --o usecases_ListTodosImpl
     ptr_usecases_InitUpdateTodo --o usecases_UpdateTodoImpl
     ptr_worker_OutboxPublisher --- SymbiontApp
     ptr_worker_TodoEventSubscriber --- SymbiontApp
     DB_HOST -.-> ptr_postgres_InitDB
     DB_NAME -.-> ptr_postgres_InitDB
     DB_PASS -.-> ptr_postgres_InitDB
     DB_PORT -.-> ptr_postgres_InitDB
     DB_USER -.-> ptr_postgres_InitDB
     FETCH_OUTBOX_INTERVAL -.-> ptr_worker_OutboxPublisher
     HTTP_PORT -.-> ptr_http_TodoMailerServer
     LLM_MODEL -.-> ptr_llm_InitBoardSummaryGenerator
     LLM_MODEL_API_KEY -.-> ptr_llm_InitBoardSummaryGenerator
     LLM_MODEL_HOST -.-> ptr_llm_InitBoardSummaryGenerator
     PUBSUB_PROJECT_ID -.-> ptr_pubsub_InitClient
     PUBSUB_SUBSCRIPTION_ID -.-> ptr_worker_TodoEventSubscriber
     PUBSUB_TOPIC_ID -.-> ptr_worker_OutboxPublisher
     SUMMARY_BATCH_INTERVAL -.-> ptr_worker_TodoEventSubscriber
     SUMMARY_BATCH_SIZE -.-> ptr_worker_TodoEventSubscriber
     llm_BoardSummaryGenerator -.-> ptr_usecases_InitGenerateBoardSummary
     postgres_BoardSummaryRepository -.-> ptr_usecases_InitGenerateBoardSummary
     postgres_BoardSummaryRepository -.-> ptr_usecases_InitGetBoardSummary
     postgres_TodoRepository -.-> ptr_usecases_InitGenerateBoardSummary
     postgres_TodoRepository -.-> ptr_usecases_InitListTodos
     time_CurrentTimeProvider -.-> ptr_llm_InitBoardSummaryGenerator
     time_CurrentTimeProvider -.-> ptr_usecases_InitCreateTodo
     time_CurrentTimeProvider -.-> ptr_usecases_InitUpdateTodo
     usecases_CompletedSummaryQueue -.-> ptr_usecases_InitGenerateBoardSummary
     usecases_CreateTodoImpl -.-> ptr_http_TodoMailerServer
     usecases_GenerateBoardSummaryImpl -.-> ptr_worker_TodoEventSubscriber
     usecases_GetBoardSummaryImpl -.-> ptr_http_TodoMailerServer
     usecases_ListTodosImpl -.-> ptr_http_TodoMailerServer
     usecases_UpdateTodoImpl -.-> ptr_http_TodoMailerServer
     style usecases_GetBoardSummaryImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
     style usecases_CreateTodoImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
     style usecases_UpdateTodoImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
     style usecases_CompletedSummaryQueue fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
     style LLM_MODEL_API_KEY fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
     style DB_HOST fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
     style FETCH_OUTBOX_INTERVAL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
     style PUBSUB_TOPIC_ID fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
     style SUMMARY_BATCH_SIZE fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
     style llm_BoardSummaryGenerator fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
     style DB_PASS fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
     style DB_PORT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
     style DB_USER fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
     style HTTP_PORT fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
     style PUBSUB_SUBSCRIPTION_ID fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
     style SUMMARY_BATCH_INTERVAL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
     style PUBSUB_PROJECT_ID fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
     style ptr_postgres_UnitOfWork fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
     style postgres_BoardSummaryRepository fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
     style time_CurrentTimeProvider fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
     style LLM_MODEL_HOST fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
     style ptr_http_Client fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
     style ptr_sql_DB fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
     style usecases_ListTodosImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
     style DB_NAME fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
     style LLM_MODEL fill:#f1f7d2,stroke:#a7c957,stroke-width:2px,color:#222222
     style ptr_log_Logger fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
     style postgres_TodoRepository fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
     style ptr_pubsub_Client fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
     style usecases_GenerateBoardSummaryImpl fill:#d6fff9,stroke:#2ec4b6,stroke-width:2px,color:#222222
     style ptr_postgres_InitTodoRepository fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
     style ptr_usecases_InitGenerateBoardSummary fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
     style ptr_tracing_InitOpenTelemetry fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
     style ptr_config_InitVaultProvider fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
     style ptr_usecases_InitCreateTodo fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
     style ptr_llm_InitBoardSummaryGenerator fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
     style ptr_pubsub_InitClient fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
     style ptr_log_InitLogger fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
     style ptr_postgres_InitUnitOfWork fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
     style ptr_postgres_InitBoardSummaryRepository fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
     style ptr_postgres_InitDB fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
     style ptr_time_InitCurrentTimeProvider fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
     style ptr_usecases_InitUpdateTodo fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
     style ptr_usecases_InitGetBoardSummary fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
     style ptr_tracing_InitHttpClient fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
     style ptr_usecases_InitListTodos fill:#f0f0f0,stroke:#373636,stroke-width:1px,color:#222222,font-weight:bold
     style ptr_app_ReportLoggerIntrospector fill:#fff3e0,stroke:#f57c00,stroke-width:2px,color:#222222
     style ptr_worker_TodoEventSubscriber fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
     style ptr_http_TodoMailerServer fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
     style ptr_worker_OutboxPublisher fill:#f1e8ff,stroke:#7b2cbf,stroke-width:2px,color:#222222
     style SymbiontApp fill:#0f56c4,stroke:#68a4eb,stroke-width:6px,color:#ffffff,font-weight:bold
```


## Prerequisites

- Docker and Docker Compose
- Go 1.25 or higher (for local development)
- Node.js 18+ (for webapp development)

## Quick Start

### 1. Start the Application

```bash
docker compose up -d
```

The application will be available at:
- **Web UI**: http://localhost:8080
- **API**: http://localhost:8080/api/v1
- **Jaeger Tracing**: http://localhost:16686

## Configuration

### Vault Secrets

The application stores sensitive configuration in Vault at `secret/data/todoapp`:


### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8080` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_NAME` | Database name | `todoappdb` |
| `VAULT_ADDR` | Vault server address | `http://localhost:8200` |
| `VAULT_TOKEN` | Vault authentication token | `root-token` |
| `VAULT_MOUNT_PATH` | Vault mount path | `secret` |
| `VAULT_SECRET_PATH` | Vault secret path | `data/todoapp` |
| `PUBSUB_EMULATOR_HOST` | Pub/Sub emulator host | `localhost:8681` |
| `PUBSUB_PROJECT_ID` | Pub/Sub project ID | `local-dev` |
| `PUBSUB_TOPIC_ID` | Pub/Sub topic name | `todo` |
| `LLM_MODEL_HOST` | Docker Model Runner endpoint | `http://localhost:12434` |
| `LLM_MODEL` | Docker Model Runnee LLM Model | `ai/gpt-oss` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry endpoint | `http://localhost:4318` |

## Local Development

### Running Locally

```bash
docker compose -f docker-compose.deps.yml  up -d 
```

```bash
# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=todoappdb
export DB_PASSWORD=todoapppass
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=root-token
export PUBSUB_EMULATOR_HOST=localhost:8681
export PUBSUB_PROJECT_ID=local-dev
export LLM_MODEL_HOST=http://localhost:12434
export LLM_MODEL=ai/gpt-oss
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318

# Run the application
go run cmd/todoapp/main.go
```

The application automatically:
- Runs database migrations on startup
- Starts the HTTP server on port 8080
- Starts the background worker
- Connects to all dependencies

### Running Tests

#### Unit Tests

```bash
# Run all unit tests
go test ./...
```

#### Integration Tests

Integration tests use testcontainers to automatically spin up and manage all required dependencies (PostgreSQL, Vault, Pub/Sub, Docker Model Runner). No manual setup needed.

```bash
# Run integration tests
go test -tags=integration ./tests/integration/... -v

# Run with timeout
go test -tags=integration -timeout=5m ./tests/integration/... -v
```

Testcontainers automatically:
- Pulls and starts required Docker images
- Waits for services to be healthy
- Cleans up containers after tests complete
- Isolates tests from each other

### Generate Mocks 

```bash
go generate ./...
```

### Frontend Development

```bash
cd webapp

# Install dependencies
npm install

# Start dev server with hot reload
npm run dev

# Build for production
npm run build
```

The React app will be available at http://localhost:5173 in dev mode.

## Observability

### Application Logs

View real-time application logs:

```bash
# View logs from Docker container
docker-compose logs -f todoapp

# View logs with timestamps
docker-compose logs -f --timestamps todoapp

# View last 100 lines
docker-compose logs --tail=100 todoapp
```

### Jaeger Tracing

Access Jaeger UI at http://localhost:16686 to view:
- Request traces across services
- API call latencies
- Database query performance
- Pub/Sub message flow
- LLM generation traces

Search for traces by:
- Service name: `todoapp`
- Operation: `HTTP POST /api/v1/todos`, `GenerateBoardSummary`, etc.
- Tags: `http.status_code`, `db.statement`, etc.

## Technical Details

### Symbiont Features

This example demonstrates Symbiont's key features:

1. **Dependency Injection**: All components are wired through Symbiont's DI container
2. **Configuration Loading**: Type-safe configuration from multiple sources (environment variables, Vault, defaults)
3. **Introspection**: Auto-generated dependency graphs for visualizing application architecture
4. **Multi-Process Hosting**: Run multiple Runners/servers/workers within the same deployable for simplified ops
