DROP TABLE IF EXISTS task_applications;
DROP TABLE IF EXISTS tasks;

CREATE TABLE tasks (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    original_id BIGINT,
    previous_id BIGINT REFERENCES tasks(id),
    description TEXT NOT NULL,
    note TEXT NOT NULL,
    version INT DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE TABLE task_applications (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    application_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_tasks_id ON tasks(id);
CREATE INDEX idx_tasks_created_at ON tasks(created_at);
CREATE INDEX idx_tasks_deleted_at ON tasks(deleted_at);
CREATE INDEX idx_tasks_original_id ON tasks(original_id);
CREATE INDEX idx_tasks_previous_id ON tasks(previous_id);

CREATE INDEX idx_task_applications_task_id ON task_applications(task_id);
CREATE INDEX idx_task_applications_name ON task_applications(application_name);
CREATE INDEX idx_task_applications_deleted_at ON task_applications(deleted_at);