package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
)

// =================================================================
// In-Memory Asynq Testing Helpers
// =================================================================

// MockAsynqClient simulates asynq.Client for testing
type MockAsynqClient struct {
	EnqueuedTasks []*asynq.Task
	mu            sync.Mutex
}

// NewTestAsynqClient creates an in-memory asynq client for testing
func NewTestAsynqClient() *MockAsynqClient {
	return &MockAsynqClient{
		EnqueuedTasks: make([]*asynq.Task, 0),
	}
}

// Enqueue simulates job enqueueing
func (c *MockAsynqClient) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.EnqueuedTasks = append(c.EnqueuedTasks, task)

	// Return mock TaskInfo
	info := &asynq.TaskInfo{
		ID:       fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Queue:    "default",
		Type:     task.Type(),
		Payload:  task.Payload(),
		State:    asynq.TaskStatePending,
		MaxRetry: 3,
		Retried:  0,
	}

	return info, nil
}

// Close simulates client cleanup
func (c *MockAsynqClient) Close() error {
	return nil
}

// GetEnqueuedTasks returns all enqueued tasks (for verification)
func (c *MockAsynqClient) GetEnqueuedTasks() []*asynq.Task {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.EnqueuedTasks
}

// ClearEnqueuedTasks clears the enqueued tasks list
func (c *MockAsynqClient) ClearEnqueuedTasks() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.EnqueuedTasks = make([]*asynq.Task, 0)
}

// =================================================================
// MockAsynqServer - In-memory task execution
// =================================================================

// MockAsynqServer simulates asynq.Server for testing
type MockAsynqServer struct {
	handlers      map[string]asynq.Handler
	processedTasks []*asynq.Task
	mu            sync.Mutex
}

// NewTestAsynqServer creates an in-memory asynq server for testing
func NewTestAsynqServer() *MockAsynqServer {
	return &MockAsynqServer{
		handlers:      make(map[string]asynq.Handler),
		processedTasks: make([]*asynq.Task, 0),
	}
}

// RegisterHandler registers a task handler
func (s *MockAsynqServer) RegisterHandler(taskType string, handler asynq.Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers[taskType] = handler
}

// ProcessTask simulates task processing
func (s *MockAsynqServer) ProcessTask(ctx context.Context, task *asynq.Task) error {
	s.mu.Lock()
	handler, ok := s.handlers[task.Type()]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("no handler registered for task type: %s", task.Type())
	}

	// Execute handler
	err := handler.ProcessTask(ctx, task)

	s.mu.Lock()
	s.processedTasks = append(s.processedTasks, task)
	s.mu.Unlock()

	return err
}

// GetProcessedTasks returns all processed tasks
func (s *MockAsynqServer) GetProcessedTasks() []*asynq.Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.processedTasks
}

// =================================================================
// MockAsynqInspector - Task inspection
// =================================================================

// MockAsynqInspector simulates asynq.Inspector for testing
type MockAsynqInspector struct {
	EnqueuedTasks []*asynq.TaskInfo
	mu            sync.Mutex
}

// NewTestAsynqInspector creates a mock inspector
func NewTestAsynqInspector() *MockAsynqInspector {
	return &MockAsynqInspector{
		EnqueuedTasks: make([]*asynq.TaskInfo, 0),
	}
}

// AddTask adds a task to the inspector
func (i *MockAsynqInspector) AddTask(task *asynq.Task) {
	i.mu.Lock()
	defer i.mu.Unlock()

	info := &asynq.TaskInfo{
		ID:       fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Queue:    "default",
		Type:     task.Type(),
		Payload:  task.Payload(),
		State:    asynq.TaskStatePending,
		MaxRetry: 3,
		Retried:  0,
	}

	i.EnqueuedTasks = append(i.EnqueuedTasks, info)
}

// GetQueueInfo returns queue statistics
func (i *MockAsynqInspector) GetQueueInfo(queueName string) (*asynq.QueueInfo, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Count tasks by state
	pending := 0
	for _, task := range i.EnqueuedTasks {
		if task.State == asynq.TaskStatePending {
			pending++
		}
	}

	return &asynq.QueueInfo{
		Queue:   queueName,
		Pending: pending,
		Active:  0,
		Size:    len(i.EnqueuedTasks),
	}, nil
}

// =================================================================
// EnqueueAndWait - Helper to enqueue and wait for completion
// =================================================================

// EnqueueAndWait enqueues a job and waits for completion (for testing)
func EnqueueAndWait(
	t *testing.T,
	client *MockAsynqClient,
	server *MockAsynqServer,
	task *asynq.Task,
	timeout time.Duration,
) error {
	t.Helper()

	// Enqueue task
	_, err := client.Enqueue(task)
	require.NoError(t, err, "Failed to enqueue task")

	// Process task
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err = server.ProcessTask(ctx, task)
	if err != nil {
		return fmt.Errorf("task processing failed: %w", err)
	}

	return nil
}

// =================================================================
// Task Payload Helpers
// =================================================================

// CreateEnrichmentPayload creates a JSON payload for enrichment job
func CreateEnrichmentPayload(submissionID string) []byte {
	payload := map[string]string{
		"submission_id": submissionID,
	}
	data, _ := json.Marshal(payload)
	return data
}

// CreateAnalysisPayload creates a JSON payload for analysis job
func CreateAnalysisPayload(submissionID, enrichmentID string) []byte {
	payload := map[string]string{
		"submission_id": submissionID,
		"enrichment_id": enrichmentID,
	}
	data, _ := json.Marshal(payload)
	return data
}

// ParseEnrichmentPayload extracts submission ID from enrichment payload
func ParseEnrichmentPayload(t *testing.T, payload []byte) string {
	t.Helper()

	var p map[string]string
	err := json.Unmarshal(payload, &p)
	require.NoError(t, err, "Failed to parse enrichment payload")

	return p["submission_id"]
}

// ParseAnalysisPayload extracts IDs from analysis payload
func ParseAnalysisPayload(t *testing.T, payload []byte) (submissionID, enrichmentID string) {
	t.Helper()

	var p map[string]string
	err := json.Unmarshal(payload, &p)
	require.NoError(t, err, "Failed to parse analysis payload")

	return p["submission_id"], p["enrichment_id"]
}

// =================================================================
// Task Verification Helpers
// =================================================================

// AssertTaskEnqueued verifies a task was enqueued
func AssertTaskEnqueued(t *testing.T, client *MockAsynqClient, taskType string) {
	t.Helper()

	tasks := client.GetEnqueuedTasks()
	found := false

	for _, task := range tasks {
		if task.Type() == taskType {
			found = true
			break
		}
	}

	require.True(t, found, "Task of type '%s' was not enqueued", taskType)
}

// AssertTaskProcessed verifies a task was processed
func AssertTaskProcessed(t *testing.T, server *MockAsynqServer, taskType string) {
	t.Helper()

	tasks := server.GetProcessedTasks()
	found := false

	for _, task := range tasks {
		if task.Type() == taskType {
			found = true
			break
		}
	}

	require.True(t, found, "Task of type '%s' was not processed", taskType)
}

// GetTaskByType retrieves first task of given type
func GetTaskByType(tasks []*asynq.Task, taskType string) *asynq.Task {
	for _, task := range tasks {
		if task.Type() == taskType {
			return task
		}
	}
	return nil
}

// CountTasksByType counts tasks of a specific type
func CountTasksByType(tasks []*asynq.Task, taskType string) int {
	count := 0
	for _, task := range tasks {
		if task.Type() == taskType {
			count++
		}
	}
	return count
}

// =================================================================
// Helper Types for Testing
// =================================================================

// Task is a simplified task structure for testing examples
type Task struct {
	TaskType string
	Payload  []byte
}

// Type returns the task type
func (t *Task) Type() string {
	return t.TaskType
}

// NewAsynqTask converts a simple Task to asynq.Task
func NewAsynqTask(taskType string, payload []byte) *asynq.Task {
	return asynq.NewTask(taskType, payload)
}
