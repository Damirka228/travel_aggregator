package main

import "log"

func main() {
	app := NewApp()
	defer app.Shutdown()
	if err := app.Run(); err != nil {
		log.Fatalf("error start server, error: %s", err)
	}
}
