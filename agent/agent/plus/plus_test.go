package plus_test

import (
	"encoding/json"
	"testing"
	"time"

	"agent/plus"
)

func TestNewPlus(t *testing.T) {
	p := plus.NewPlus()
	if p == nil {
		t.Fatal("NewPlus() returned nil")
	}
}

func TestStartAndStop(t *testing.T) {
	p := plus.NewPlus()

	// Start Plus component
	if err := p.Start(); err != nil {
		t.Fatalf("Failed to start Plus: %v", err)
	}

	// Give it some time to initialize
	time.Sleep(1 * time.Second)

	// Stop Plus component
	if err := p.Stop(); err != nil {
		t.Fatalf("Failed to stop Plus: %v", err)
	}
}

func TestTaskManager(t *testing.T) {
	// Test TaskManager functionality
	tm := plus.NewTaskManager(nil)

	// Start task manager
	if err := tm.Start(); err != nil {
		t.Fatalf("Failed to start task manager: %v", err)
	}

	// Create a test task
	testTask := &plus.Task{
		ID:          "test-task-1",
		Name:        "test-task",
		Description: "A test task",
		StepList:    []plus.Step{},
		CronConfig:  &plus.CronConfig{},
		Status:      plus.TaskStatusReceived,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Add task to manager (via UpdateTask)
	taskJSON, err := json.Marshal(testTask)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	if err := tm.UpdateTask(string(taskJSON)); err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// Start the task
	if err := tm.StartTask("test-task-1"); err != nil {
		t.Fatalf("Failed to start task: %v", err)
	}

	// Give it some time to start
	time.Sleep(100 * time.Millisecond)

	// Check if task is running or completed (since it's a simple task with no plugins)
	status, err := tm.GetTaskStatus("test-task-1")
	if err != nil {
		t.Fatalf("Failed to get task status: %v", err)
	}

	if status != plus.TaskStatusRunning && status != plus.TaskStatusCompleted {
		t.Fatalf("Expected task to be running or completed, got status: %s", status)
	}

	// Stop the task
	if err := tm.StopTask("test-task-1"); err != nil {
		t.Fatalf("Failed to stop task: %v", err)
	}

	// Give it some time to stop
	time.Sleep(1 * time.Second)

	// Check if task is stopped
	status, err = tm.GetTaskStatus("test-task-1")
	if err != nil {
		t.Fatalf("Failed to get task status: %v", err)
	}

	if status != plus.TaskStatusCancelled {
		t.Fatalf("Expected task to be cancelled, got status: %s", status)
	}

	// Stop task manager
	if err := tm.Stop(); err != nil {
		t.Fatalf("Failed to stop task manager: %v", err)
	}
}
