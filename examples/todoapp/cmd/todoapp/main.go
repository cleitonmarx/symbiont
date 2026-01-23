package main

import "github.com/cleitonmarx/symbiont/examples/todoapp/internal/app"

func main() {
	err := app.NewTodoMailerApp().
		Instrospect(&app.ReportLoggerIntrospector{}).
		Run()
	if err != nil {
		panic(err)
	}
}
