package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(h *Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	r.Route("/api/v1/tasks", func(r chi.Router) {
		r.Get("/", h.GetAllTasks)
		r.Post("/", h.CreateTask)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetTaskByID)
			r.Put("/", h.UpdateTaskByID)
			r.Delete("/", h.DeleteTaskByID)
		})
	})
	return r
}
