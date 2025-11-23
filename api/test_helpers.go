package api

import (
	"context"

	"backend_v3/domain/analysis"
	"backend_v3/domain/enrichment"
	"backend_v3/domain/submission"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
)

// TestHandler is a test version of Handler that allows mocking
type TestHandler struct {
	Handler
	MockSubmissionSvc *MockSubmissionService
	MockEnrichmentSvc *MockEnrichmentService
	MockAnalysisSvc   *MockAnalysisService
	MockReportSvc     *MockReportService
	MockAsynqClient   *MockAsynqClient
}

// NewTestHandler creates a handler for testing with all mocks
func NewTestHandler() *TestHandler {
	gin.SetMode(gin.TestMode)

	th := &TestHandler{
		MockSubmissionSvc: new(MockSubmissionService),
		MockEnrichmentSvc: new(MockEnrichmentService),
		MockAnalysisSvc:   new(MockAnalysisService),
		MockReportSvc:     new(MockReportService),
		MockAsynqClient:   new(MockAsynqClient),
	}

	// Create real services with mock repositories
	th.Handler = Handler{
		logger: zerolog.Nop(),
	}

	return th
}

// ==================== SUBMISSION SERVICE MOCK ====================

// SubmissionServiceInterface defines the submission service interface
type SubmissionServiceInterface interface {
	SubmitForm(ctx context.Context, req *submission.SubmitRequest) (*submission.Submission, error)
	GetByID(ctx context.Context, id interface{}) (*submission.Submission, error)
	ListAll(ctx context.Context, opts *submission.ListOptions) ([]*submission.Submission, int, error)
	GetAnalytics(ctx context.Context) (*submission.AnalyticsData, error)
}

type MockSubmissionService struct {
	mock.Mock
}

func (m *MockSubmissionService) SubmitForm(ctx context.Context, req *submission.SubmitRequest) (*submission.Submission, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*submission.Submission), args.Error(1)
}

func (m *MockSubmissionService) GetByID(ctx context.Context, id interface{}) (*submission.Submission, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*submission.Submission), args.Error(1)
}

func (m *MockSubmissionService) ListAll(ctx context.Context, opts *submission.ListOptions) ([]*submission.Submission, int, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*submission.Submission), args.Int(1), args.Error(2)
}

func (m *MockSubmissionService) GetAnalytics(ctx context.Context) (*submission.AnalyticsData, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*submission.AnalyticsData), args.Error(1)
}

// ==================== ENRICHMENT SERVICE MOCK ====================

type EnrichmentServiceInterface interface {
	GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*enrichment.Enrichment, error)
	GetByID(ctx context.Context, id uuid.UUID) (*enrichment.Enrichment, error)
	UpdateFields(ctx context.Context, id uuid.UUID, fields map[string]interface{}) (*enrichment.Enrichment, error)
	Approve(ctx context.Context, id uuid.UUID) error
}

type MockEnrichmentService struct {
	mock.Mock
}

func (m *MockEnrichmentService) GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*enrichment.Enrichment, error) {
	args := m.Called(ctx, submissionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*enrichment.Enrichment), args.Error(1)
}

func (m *MockEnrichmentService) GetByID(ctx context.Context, id uuid.UUID) (*enrichment.Enrichment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*enrichment.Enrichment), args.Error(1)
}

func (m *MockEnrichmentService) UpdateFields(ctx context.Context, id uuid.UUID, fields map[string]interface{}) (*enrichment.Enrichment, error) {
	args := m.Called(ctx, id, fields)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*enrichment.Enrichment), args.Error(1)
}

func (m *MockEnrichmentService) Approve(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// ==================== ANALYSIS SERVICE MOCK ====================

type AnalysisServiceInterface interface {
	GetBySubmissionID(ctx context.Context, submissionID string) (*analysis.Analysis, error)
	GetByID(ctx context.Context, id string) (*analysis.Analysis, error)
	UpdateFields(ctx context.Context, id string, fields map[string]interface{}) (*analysis.Analysis, error)
	CreateVersion(ctx context.Context, parentID string, edits map[string]interface{}) (*analysis.Analysis, error)
	Approve(ctx context.Context, id string) error
	Send(ctx context.Context, id string, userEmail string) error
}

type MockAnalysisService struct {
	mock.Mock
}

func (m *MockAnalysisService) GetBySubmissionID(ctx context.Context, submissionID string) (*analysis.Analysis, error) {
	args := m.Called(ctx, submissionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*analysis.Analysis), args.Error(1)
}

func (m *MockAnalysisService) GetByID(ctx context.Context, id string) (*analysis.Analysis, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*analysis.Analysis), args.Error(1)
}

func (m *MockAnalysisService) UpdateFields(ctx context.Context, id string, fields map[string]interface{}) (*analysis.Analysis, error) {
	args := m.Called(ctx, id, fields)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*analysis.Analysis), args.Error(1)
}

func (m *MockAnalysisService) CreateVersion(ctx context.Context, parentID string, edits map[string]interface{}) (*analysis.Analysis, error) {
	args := m.Called(ctx, parentID, edits)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*analysis.Analysis), args.Error(1)
}

func (m *MockAnalysisService) Approve(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAnalysisService) Send(ctx context.Context, id string, userEmail string) error {
	args := m.Called(ctx, id, userEmail)
	return args.Error(0)
}

// ==================== REPORT SERVICE MOCK ====================

type ReportServiceInterface interface {
	GeneratePreview(ctx context.Context, submissionID string, analysisID string) (map[string]string, error)
	Publish(ctx context.Context, submissionID string, analysisID string) (string, error)
}

type MockReportService struct {
	mock.Mock
}

func (m *MockReportService) GeneratePreview(ctx context.Context, submissionID string, analysisID string) (map[string]string, error) {
	args := m.Called(ctx, submissionID, analysisID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockReportService) Publish(ctx context.Context, submissionID string, analysisID string) (string, error) {
	args := m.Called(ctx, submissionID, analysisID)
	return args.String(0), args.Error(1)
}

// ==================== ASYNQ CLIENT MOCK ====================

type AsynqClientInterface interface {
	Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
	Close() error
}

type MockAsynqClient struct {
	mock.Mock
}

func (m *MockAsynqClient) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	args := m.Called(task, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*asynq.TaskInfo), args.Error(1)
}

func (m *MockAsynqClient) Close() error {
	args := m.Called()
	return args.Error(0)
}
