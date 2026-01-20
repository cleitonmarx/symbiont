# Todo Completion Email Demo UI

This project is a developer-friendly UI for the Todo Completion Email Demo API. It is built using React and TypeScript, and it provides a simple interface for managing todos, including creating, updating, and listing them with their email delivery statuses.

## Project Structure

```
webapp
├── src
│   ├── main.tsx               # Entry point of the application
│   ├── App.tsx                # Main application component with routing
│   ├── vite-env.d.ts          # TypeScript definitions for Vite environment variables
│   ├── components              # Contains reusable components
│   │   ├── TodoList.tsx       # Component to display the list of todos
│   │   ├── TodoItem.tsx       # Component for individual todo items
│   │   ├── CreateTodoForm.tsx # Form for creating new todos
│   │   └── ErrorMessage.tsx    # Component for displaying error messages
│   ├── services                # API client setup
│   │   └── api.ts             # Functions for interacting with the API
│   ├── types                   # TypeScript types and interfaces
│   │   └── index.ts           # Definitions for todos and API responses
│   ├── hooks                   # Custom hooks
│   │   └── useTodos.ts        # Hook for fetching and managing todos
│   └── styles                  # Global styles
│       └── index.css          # CSS styles for the application
├── public                      # Static assets (currently empty)
├── index.html                 # Main HTML file
├── package.json               # Project metadata and dependencies
├── tsconfig.json              # TypeScript configuration
├── tsconfig.node.json         # Node.js specific TypeScript configuration
├── vite.config.ts             # Vite configuration
└── README.md                  # Project documentation
```

## Getting Started

To run the application locally, follow these steps:

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd webapp
   ```

2. **Install dependencies**:
   ```bash
   npm install
   ```

3. **Set the API base URL**:
   Update the API base URL in the `src/services/api.ts` file to point to your Todo Completion Email Demo API.

4. **Run the application**:
   ```bash
   npm run dev
   ```

5. **Open your browser**:
   Navigate to `http://localhost:3000` to view the application.

## Demo Script

To test the application, you can use the following demo script:

1. Create a new todo by entering a title in the "Create Todo" form and clicking "Add Todo".
2. View the list of todos and their statuses.
3. Update a todo by clicking on the "Complete" or "Rename" buttons next to each todo.
4. Observe the email delivery status updates as you complete todos.

## License

This project is licensed under the MIT License.