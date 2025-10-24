package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{
		db: db,
	}
}

type Task struct {
	ID           int64     `json:"id"`
	Description  string    `json:"description"`
	Note         string    `json:"note"`
	Applications []string  `json:"applications"`
	CreatedAt    time.Time `json:"created_at"`
}

type ResponseTask struct {
	ID           int64    `json:"id"`
	OriginalID   int64    `json:"original_id,omitempty"`
	PreviousID   int64    `json:"previous_id,omitempty"`
	Version      int      `json:"version,omitempty"`
	Message      string   `json:"message"`
	Description  string   `json:"description"`
	Note         string   `json:"note"`
	Applications []string `json:"applications"`
}

func (h *Handler) GetAllTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, err := h.db.Query(`
        SELECT id, description, note, created_at 
        FROM tasks 
        WHERE deleted_at IS NULL
			AND updated_at IS NULL
        ORDER BY created_at DESC
    `)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get tasks"})
		return
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.Description, &task.Note, &task.CreatedAt); err != nil {
			continue
		}
		task.Applications, _ = h.getTaskApplications(task.ID)
		tasks = append(tasks, task)
	}

	json.NewEncoder(w).Encode(tasks)
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON: " + err.Error()})
		return
	}

	if err := validateTask(task); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to start transaction: " + err.Error()})
		return
	}
	defer tx.Rollback()

	var taskID int64
	err = tx.QueryRow(`
		INSERT INTO tasks (description, note) 
		VALUES ($1, $2) 
		RETURNING id
	`, task.Description, task.Note).Scan(&taskID)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create task: " + err.Error()})
		return
	}

	if len(task.Applications) > 0 {
		stmt, err := tx.Prepare("INSERT INTO task_applications (task_id, application_name) VALUES ($1, $2)")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to prepare statement: " + err.Error()})
			return
		}
		defer stmt.Close()

		for _, app := range task.Applications {
			if strings.TrimSpace(app) != "" {
				_, err := stmt.Exec(taskID, strings.TrimSpace(app))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": "Failed to insert application: " + err.Error()})
					return
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	response := &ResponseTask{
		ID:           taskID,
		Message:      "Task created successfully",
		Description:  task.Description,
		Note:         task.Note,
		Applications: task.Applications,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func validateTask(task Task) error {
	if strings.TrimSpace(task.Description) == "" {
		return fmt.Errorf("description is required")
	}
	if strings.TrimSpace(task.Note) == "" {
		return fmt.Errorf("note is required")
	}
	if len(task.Applications) == 0 {
		return fmt.Errorf("at least one application is required")
	}

	for i, app := range task.Applications {
		if strings.TrimSpace(app) == "" {
			return fmt.Errorf("application at index %d cannot be empty", i)
		}
	}

	return nil
}

func (h *Handler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task ID is required"})
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid task ID format"})
		return
	}

	var task Task
	err = h.db.QueryRow(`
		SELECT id, description, note, created_at 
		FROM tasks 
		WHERE id = $1 
			AND deleted_at IS NULL
			AND updated_at IS NULL
	`, id).Scan(&task.ID, &task.Description, &task.Note, &task.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get task: " + err.Error()})
		return
	}

	task.Applications, err = h.getTaskApplications(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get applications: " + err.Error()})
		return
	}

	json.NewEncoder(w).Encode(task)
}

func (h *Handler) getTaskApplications(taskID int64) ([]string, error) {
	var applications []string

	rows, err := h.db.Query(`
		SELECT application_name 
		FROM task_applications 
		WHERE task_id = $1
			AND deleted_at IS NULL
		ORDER BY id
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var app string
		if err := rows.Scan(&app); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		applications = append(applications, app)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return applications, nil
}

func (h *Handler) UpdateTaskByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task ID is required"})
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid task ID format"})
		return
	}

	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if err := validateTask(task); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	var currentVersion int
	var originalID int64
	err = tx.QueryRow(`
		SELECT COALESCE(version, 1), COALESCE(original_id, id) 
		FROM tasks WHERE id = $1 
			AND deleted_at IS NULL
			AND updated_at IS NULL
	`, id).Scan(&currentVersion, &originalID)

	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get task version: " + err.Error()})
		return
	}

	result, err := tx.Exec(`UPDATE tasks SET updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to mark old version as deleted: " + err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	}

	var newTaskID int64
	err = tx.QueryRow(`
		INSERT INTO tasks (original_id, previous_id, description, note, version) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id
	`, originalID, id, task.Description, task.Note, currentVersion+1).Scan(&newTaskID)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create new task version: " + err.Error()})
		return
	}

	if len(task.Applications) > 0 {
		stmt, err := tx.Prepare("INSERT INTO task_applications (task_id, application_name) VALUES ($1, $2)")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to prepare statement: " + err.Error()})
			return
		}
		defer stmt.Close()

		for _, app := range task.Applications {
			if strings.TrimSpace(app) != "" {
				_, err := stmt.Exec(newTaskID, strings.TrimSpace(app))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": "Failed to insert application: " + err.Error()})
					return
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	response := &ResponseTask{
		ID:           newTaskID,
		OriginalID:   originalID,
		PreviousID:   id,
		Version:      currentVersion + 1,
		Message:      "Task updated successfully (new version created)",
		Description:  task.Description,
		Note:         task.Note,
		Applications: task.Applications,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) DeleteTaskByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task ID is required"})
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid task ID format"})
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	resultTask, errTask := tx.Exec(`
		UPDATE tasks
		SET deleted_at = now()
		WHERE id = $1
			AND deleted_at IS NULL
		`, id)
	if errTask != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete task(update field deleted_at)"})
		return
	}
	rowAffected, _ := resultTask.RowsAffected()
	if rowAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task with id: \"" + idStr + "\" not found"})
		return
	}

	_, errTaskApplication := tx.Exec(`
		UPDATE task_applications
		SET deleted_at = now()
		WHERE task_id = $1
			AND deleted_at IS NULL
	`, id)
	if errTaskApplication != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete task_applications(update field deleted_at)"})
		return
	}

	if err := tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
