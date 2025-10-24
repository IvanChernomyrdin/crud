package main

import (
	"RestApiCRUD/internal/config"
	"RestApiCRUD/internal/config/db"
	"RestApiCRUD/internal/handlers"
	"log"
	"net/http"
)

func main() {
	cfg := config.NewConfig()

	var err error
	database, err := db.NewConnection(cfg.DatabasePostgres)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer database.Close()

	if err := database.Ping(); err != nil {
		log.Fatalf("Ошибка проверки соединения с БД: %v", err)
	}

	h := handlers.NewHandler(database)
	r := handlers.NewRouter(h)

	server := &http.Server{
		Addr:    cfg.Host,
		Handler: r,
	}

	log.Printf("Сервер запущен по адресу: %s", cfg.Host)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Сервер завершил работу с критической ошибкой: %v", err)
	}
}
