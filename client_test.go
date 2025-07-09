package clavata

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	gatewayv1 "github.com/clavataai/monorail/libs/protobufs/gateway/v1"
	sharedv1 "github.com/clavataai/monorail/libs/protobufs/shared/v1"
)

// mockGatewayClient is a manual mock implementation of GatewayServiceClient
type mockGatewayClient struct {
	// Functions for each method
	createJobFunc func(ctx context.Context, in *gatewayv1.CreateJobRequest, opts ...grpc.CallOption) (*gatewayv1.CreateJobResponse, error)
	getJobFunc    func(ctx context.Context, in *gatewayv1.GetJobRequest, opts ...grpc.CallOption) (*gatewayv1.GetJobResponse, error)
	listJobsFunc  func(ctx context.Context, in *gatewayv1.ListJobsRequest, opts ...grpc.CallOption) (*gatewayv1.ListJobsResponse, error)
	evaluateFunc  func(ctx context.Context, in *gatewayv1.EvaluateRequest, opts ...grpc.CallOption) (gatewayv1.GatewayService_EvaluateClient, error)
}

func (m *mockGatewayClient) CreateJob(ctx context.Context, in *gatewayv1.CreateJobRequest, opts ...grpc.CallOption) (*gatewayv1.CreateJobResponse, error) {
	if m.createJobFunc != nil {
		return m.createJobFunc(ctx, in, opts...)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockGatewayClient) GetJob(ctx context.Context, in *gatewayv1.GetJobRequest, opts ...grpc.CallOption) (*gatewayv1.GetJobResponse, error) {
	if m.getJobFunc != nil {
		return m.getJobFunc(ctx, in, opts...)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockGatewayClient) ListJobs(ctx context.Context, in *gatewayv1.ListJobsRequest, opts ...grpc.CallOption) (*gatewayv1.ListJobsResponse, error) {
	if m.listJobsFunc != nil {
		return m.listJobsFunc(ctx, in, opts...)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockGatewayClient) Evaluate(ctx context.Context, in *gatewayv1.EvaluateRequest, opts ...grpc.CallOption) (gatewayv1.GatewayService_EvaluateClient, error) {
	if m.evaluateFunc != nil {
		return m.evaluateFunc(ctx, in, opts...)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// mockEvaluateClient implements the grpc.ClientStream interface for testing
type mockEvaluateClient struct {
	responses []*gatewayv1.EvaluateResponse
	index     int
	err       error
}

func (m *mockEvaluateClient) Recv() (*gatewayv1.EvaluateResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.index >= len(m.responses) {
		return nil, io.EOF
	}
	resp := m.responses[m.index]
	m.index++
	return resp, nil
}

func (m *mockEvaluateClient) Header() (metadata.MD, error) { return nil, nil }
func (m *mockEvaluateClient) Trailer() metadata.MD         { return nil }
func (m *mockEvaluateClient) CloseSend() error             { return nil }
func (m *mockEvaluateClient) Context() context.Context     { return context.Background() }
func (m *mockEvaluateClient) SendMsg(interface{}) error    { return nil }
func (m *mockEvaluateClient) RecvMsg(interface{}) error    { return nil }

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		options []option
		wantErr bool
		errMsg  string
	}{
		{
			name:    "missing API key",
			options: []option{},
			wantErr: true,
			errMsg:  "API key is required",
		},
		{
			name: "with API key",
			options: []option{
				WithAPIKey("test-key"),
			},
			wantErr: false,
		},
		{
			name: "with custom endpoint",
			options: []option{
				WithAPIKey("test-key"),
				WithEndpoint("custom.endpoint:443"),
			},
			wantErr: false,
		},
		{
			name: "with insecure",
			options: []option{
				WithAPIKey("test-key"),
				WithInsecure(true),
			},
			wantErr: false,
		},
		{
			name: "with custom timeout",
			options: []option{
				WithAPIKey("test-key"),
				WithTimeout(60 * time.Second),
			},
			wantErr: false,
		},
		{
			name: "with custom retry config",
			options: []option{
				WithAPIKey("test-key"),
				WithRetryConfig(RetryConfig{
					MaxRetries:          10,
					InitialInterval:     1 * time.Second,
					MaxInterval:         60 * time.Second,
					Multiplier:          3.0,
					RandomizationFactor: 0.5,
				}),
			},
			wantErr: false,
		},
		{
			name: "with retry disabled",
			options: []option{
				WithAPIKey("test-key"),
				WithDisableRetry(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.options...)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.NotNil(t, client.conn)
				assert.NotNil(t, client.gateway)
				// Clean up
				client.Close()
			}
		})
	}
}

func TestClient_CreateJob(t *testing.T) {
	mockTime := time.Now()

	tests := []struct {
		name      string
		request   *CreateJobRequest
		mockSetup func(*mockGatewayClient)
		wantJob   *Job
		wantErr   bool
		err       error
	}{
		{
			name: "successful job creation",
			request: &CreateJobRequest{
				PolicyID: "policy-123",
				Content: []Contenter{
					NewTextContent("test content"),
				},
				Options: JobOptions{
					Threshold: 0.8,
					Expedited: true,
				},
			},
			mockSetup: func(m *mockGatewayClient) {
				m.createJobFunc = func(ctx context.Context, req *gatewayv1.CreateJobRequest, opts ...grpc.CallOption) (*gatewayv1.CreateJobResponse, error) {
					// Verify request
					assert.Equal(t, "policy-123", req.PolicyId)
					assert.Len(t, req.ContentData, 1)
					assert.NotNil(t, req.Threshold)
					assert.Equal(t, float64(0.8), *req.Threshold)
					assert.NotNil(t, req.Expedited)
					assert.True(t, *req.Expedited)

					return &gatewayv1.CreateJobResponse{
						Job: &sharedv1.Job{
							JobUuid:    "job-123",
							CustomerId: "customer-123",
							Status:     sharedv1.JobStatus_JOB_STATUS_PENDING,
							Created:    timestamppb.New(mockTime),
							Updated:    timestamppb.New(mockTime),
							PolicyId:   &[]string{"policy-123"}[0],
							Threshold:  0.8,
						},
					}, nil
				}
			},
			wantJob: &Job{
				ID:         "job-123",
				CustomerID: "customer-123",
				Status:     JobStatusPending,
				CreatedAt:  mockTime,
				UpdatedAt:  mockTime,
			},
			wantErr: false,
		},
		{
			name: "with webhook",
			request: &CreateJobRequest{
				PolicyID:   "policy-123",
				Content:    []Contenter{NewTextContent("test")},
				WebhookURL: "https://example.com/webhook",
			},
			mockSetup: func(m *mockGatewayClient) {
				m.createJobFunc = func(ctx context.Context, req *gatewayv1.CreateJobRequest, opts ...grpc.CallOption) (*gatewayv1.CreateJobResponse, error) {
					// Verify webhook was set
					assert.NotNil(t, req.Webhook)
					assert.Equal(t, "https://example.com/webhook", req.Webhook.Url)

					return &gatewayv1.CreateJobResponse{
						Job: &sharedv1.Job{
							JobUuid: "job-123",
							Status:  sharedv1.JobStatus_JOB_STATUS_PENDING,
							Created: timestamppb.New(mockTime),
							Updated: timestamppb.New(mockTime),
						},
					}, nil
				}
			},
			wantJob: &Job{
				ID:        "job-123",
				Status:    JobStatusPending,
				CreatedAt: mockTime,
				UpdatedAt: mockTime,
			},
			wantErr: false,
		},
		{
			name: "grpc error",
			request: &CreateJobRequest{
				PolicyID: "policy-123",
				Content:  []Contenter{NewTextContent("test")},
			},
			mockSetup: func(m *mockGatewayClient) {
				m.createJobFunc = func(ctx context.Context, req *gatewayv1.CreateJobRequest, opts ...grpc.CallOption) (*gatewayv1.CreateJobResponse, error) {
					return nil, status.Error(codes.InvalidArgument, "invalid policy ID")
				}
			},
			wantErr: true,
			err:     status.Error(codes.InvalidArgument, "invalid policy ID"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockGatewayClient{}
			tt.mockSetup(mockClient)

			client := &Client{
				config: &cfg{
					apiKey: "test-key",
				},
				gateway: mockClient,
			}

			job, err := client.CreateJob(context.Background(), tt.request)
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.err)
				assert.Nil(t, job)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantJob.ID, job.ID)
				assert.Equal(t, tt.wantJob.Status, job.Status)
			}
		})
	}
}

func TestClient_GetJob(t *testing.T) {
	tests := []struct {
		name      string
		jobID     string
		mockSetup func(*mockGatewayClient)
		wantJob   *Job
		wantErr   bool
	}{
		{
			name:  "successful get",
			jobID: "job-123",
			mockSetup: func(m *mockGatewayClient) {
				m.getJobFunc = func(ctx context.Context, req *gatewayv1.GetJobRequest, opts ...grpc.CallOption) (*gatewayv1.GetJobResponse, error) {
					assert.Equal(t, "job-123", req.JobUuid)

					return &gatewayv1.GetJobResponse{
						Job: &sharedv1.Job{
							JobUuid:    "job-123",
							CustomerId: "customer-123",
							Status:     sharedv1.JobStatus_JOB_STATUS_COMPLETED,
							Results: []*sharedv1.JobResult{
								{
									Report: &sharedv1.PolicyEvaluationReport{
										PolicyId: "policy-123",
										ReviewResult: &sharedv1.PolicyEvaluationReport_ReviewResult{
											Outcome: sharedv1.Outcome_OUTCOME_TRUE,
											Score:   0.95,
										},
									},
								},
							},
						},
					}, nil
				}
			},
			wantJob: &Job{
				ID:         "job-123",
				CustomerID: "customer-123",
				Status:     JobStatusCompleted,
			},
			wantErr: false,
		},
		{
			name:  "job not found",
			jobID: "nonexistent",
			mockSetup: func(m *mockGatewayClient) {
				m.getJobFunc = func(ctx context.Context, req *gatewayv1.GetJobRequest, opts ...grpc.CallOption) (*gatewayv1.GetJobResponse, error) {
					return nil, status.Error(codes.NotFound, "job not found")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockGatewayClient{}
			tt.mockSetup(mockClient)

			client := &Client{
				config: &cfg{
					apiKey: "test-key",
				},
				gateway: mockClient,
			}

			job, err := client.GetJob(context.Background(), tt.jobID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantJob.ID, job.ID)
				assert.Equal(t, tt.wantJob.Status, job.Status)
				if tt.wantJob.Status == JobStatusCompleted {
					assert.NotEmpty(t, job.Results)
				}
			}
		})
	}
}

func TestClient_Evaluate(t *testing.T) {
	tests := []struct {
		name      string
		request   *EvaluateRequest
		mockSetup func(*mockGatewayClient)
		wantCount int
		wantErr   bool
	}{
		{
			name: "successful stream",
			request: &EvaluateRequest{
				PolicyID: "policy-123",
				Content: []Contenter{
					NewTextContent("content 1"),
					NewTextContent("content 2"),
				},
			},
			mockSetup: func(m *mockGatewayClient) {
				m.evaluateFunc = func(ctx context.Context, req *gatewayv1.EvaluateRequest, opts ...grpc.CallOption) (gatewayv1.GatewayService_EvaluateClient, error) {
					stream := &mockEvaluateClient{
						responses: []*gatewayv1.EvaluateResponse{
							{
								JobUuid:     "job-123",
								ContentHash: "hash1",
								PolicyEvaluationReport: &sharedv1.PolicyEvaluationReport{
									PolicyId: "policy-123",
									ReviewResult: &sharedv1.PolicyEvaluationReport_ReviewResult{
										Outcome: sharedv1.Outcome_OUTCOME_FALSE,
										Score:   0.2,
									},
								},
							},
							{
								JobUuid:     "job-123",
								ContentHash: "hash2",
								PolicyEvaluationReport: &sharedv1.PolicyEvaluationReport{
									PolicyId: "policy-123",
									ReviewResult: &sharedv1.PolicyEvaluationReport_ReviewResult{
										Outcome: sharedv1.Outcome_OUTCOME_TRUE,
										Score:   0.9,
									},
								},
							},
						},
					}
					return stream, nil
				}
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "stream error",
			request: &EvaluateRequest{
				PolicyID: "policy-123",
				Content:  []Contenter{NewTextContent("test")},
			},
			mockSetup: func(m *mockGatewayClient) {
				m.evaluateFunc = func(ctx context.Context, req *gatewayv1.EvaluateRequest, opts ...grpc.CallOption) (gatewayv1.GatewayService_EvaluateClient, error) {
					return nil, status.Error(codes.Internal, "stream error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockGatewayClient{}
			tt.mockSetup(mockClient)

			client := &Client{
				config: &cfg{
					apiKey: "test-key",
				},
				gateway: mockClient,
			}

			ch, err := client.Evaluate(context.Background(), tt.request)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ch)

				// Collect results
				var results []EvaluateResult
				for result := range ch {
					results = append(results, result)
				}

				assert.Len(t, results, tt.wantCount)
				for _, result := range results {
					if result.Error != nil {
						assert.Fail(t, "unexpected error in result", result.Error)
					} else {
						assert.NotNil(t, result.Report)
					}
				}
			}
		})
	}
}

func TestClient_EvaluateOne(t *testing.T) {
	tests := []struct {
		name      string
		policyID  string
		content   Contenter
		options   JobOptions
		mockSetup func(*mockGatewayClient)
		wantErr   bool
	}{
		{
			name:     "successful single evaluation",
			policyID: "policy-123",
			content:  NewTextContent("test content"),
			options:  JobOptions{Threshold: 0.7},
			mockSetup: func(m *mockGatewayClient) {
				m.evaluateFunc = func(ctx context.Context, req *gatewayv1.EvaluateRequest, opts ...grpc.CallOption) (gatewayv1.GatewayService_EvaluateClient, error) {
					stream := &mockEvaluateClient{
						responses: []*gatewayv1.EvaluateResponse{
							{
								JobUuid:     "job-123",
								ContentHash: "hash123",
								PolicyEvaluationReport: &sharedv1.PolicyEvaluationReport{
									PolicyId:  "policy-123",
									Threshold: 0.7,
									ReviewResult: &sharedv1.PolicyEvaluationReport_ReviewResult{
										Outcome: sharedv1.Outcome_OUTCOME_TRUE,
										Score:   0.85,
									},
									SectionEvaluationReports: []*sharedv1.PolicyEvaluationReport_SectionEvaluationReport{
										{
											Name:    "inappropriate",
											Message: "block",
											ReviewResult: &sharedv1.PolicyEvaluationReport_ReviewResult{
												Outcome: sharedv1.Outcome_OUTCOME_TRUE,
												Score:   0.85,
											},
										},
									},
								},
							},
						},
					}
					return stream, nil
				}
			},
			wantErr: false,
		},
		{
			name:     "no results",
			policyID: "policy-123",
			content:  NewTextContent("test"),
			mockSetup: func(m *mockGatewayClient) {
				m.evaluateFunc = func(ctx context.Context, req *gatewayv1.EvaluateRequest, opts ...grpc.CallOption) (gatewayv1.GatewayService_EvaluateClient, error) {
					stream := &mockEvaluateClient{
						responses: []*gatewayv1.EvaluateResponse{},
					}
					return stream, nil
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockGatewayClient{}
			tt.mockSetup(mockClient)

			client := &Client{
				config: &cfg{
					apiKey: "test-key",
				},
				gateway: mockClient,
			}

			report, err := client.EvaluateOne(context.Background(), tt.policyID, tt.content, tt.options)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, report)
				assert.Equal(t, tt.policyID, report.PolicyID)
			}
		})
	}
}

func TestClient_ListJobs(t *testing.T) {
	tests := []struct {
		name      string
		request   *ListJobsRequest
		mockSetup func(*mockGatewayClient)
		wantJobs  int
		wantErr   bool
	}{
		{
			name: "successful list without query",
			request: &ListJobsRequest{
				PageSize: 10,
			},
			mockSetup: func(m *mockGatewayClient) {
				m.listJobsFunc = func(ctx context.Context, req *gatewayv1.ListJobsRequest, opts ...grpc.CallOption) (*gatewayv1.ListJobsResponse, error) {
					assert.Equal(t, int32(10), req.PageSize)

					return &gatewayv1.ListJobsResponse{
						Jobs: []*sharedv1.Job{
							{
								JobUuid: "job-1",
								Status:  sharedv1.JobStatus_JOB_STATUS_COMPLETED,
							},
							{
								JobUuid: "job-2",
								Status:  sharedv1.JobStatus_JOB_STATUS_PENDING,
							},
						},
						NextPageToken: "next-token",
					}, nil
				}
			},
			wantJobs: 2,
			wantErr:  false,
		},
		{
			name: "with query filters",
			request: &ListJobsRequest{
				Query: ListJobsQuery{
					PolicyID: "policy-123",
					Status:   JobStatusCompleted,
				},
				PageSize: 20,
			},
			mockSetup: func(m *mockGatewayClient) {
				m.listJobsFunc = func(ctx context.Context, req *gatewayv1.ListJobsRequest, opts ...grpc.CallOption) (*gatewayv1.ListJobsResponse, error) {
					assert.NotNil(t, req.Query)
					assert.NotNil(t, req.Query.PolicyId)
					assert.Equal(t, "policy-123", *req.Query.PolicyId)
					assert.Equal(t, sharedv1.JobStatus_JOB_STATUS_COMPLETED, req.Query.Status)

					return &gatewayv1.ListJobsResponse{
						Jobs: []*sharedv1.Job{
							{
								JobUuid:  "job-3",
								Status:   sharedv1.JobStatus_JOB_STATUS_COMPLETED,
								PolicyId: &[]string{"policy-123"}[0],
							},
						},
					}, nil
				}
			},
			wantJobs: 1,
			wantErr:  false,
		},
		{
			name: "with pagination token",
			request: &ListJobsRequest{
				PageToken: "page-token-123",
			},
			mockSetup: func(m *mockGatewayClient) {
				m.listJobsFunc = func(ctx context.Context, req *gatewayv1.ListJobsRequest, opts ...grpc.CallOption) (*gatewayv1.ListJobsResponse, error) {
					assert.Equal(t, "page-token-123", req.PageToken)

					return &gatewayv1.ListJobsResponse{
						Jobs: []*sharedv1.Job{},
					}, nil
				}
			},
			wantJobs: 0,
			wantErr:  false,
		},
		{
			name:    "error from server",
			request: &ListJobsRequest{},
			mockSetup: func(m *mockGatewayClient) {
				m.listJobsFunc = func(ctx context.Context, req *gatewayv1.ListJobsRequest, opts ...grpc.CallOption) (*gatewayv1.ListJobsResponse, error) {
					return nil, status.Error(codes.Internal, "server error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockGatewayClient{}
			tt.mockSetup(mockClient)

			client := &Client{
				config: &cfg{
					apiKey: "test-key",
				},
				gateway: mockClient,
			}

			resp, err := client.ListJobs(context.Background(), tt.request)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, resp.Jobs, tt.wantJobs)
			}
		})
	}
}

func TestContentTypes(t *testing.T) {
	t.Run("text content", func(t *testing.T) {
		content := NewTextContent("Hello, world!")
		proto, err := content.toProtoContentData()
		assert.NoError(t, err)
		assert.Equal(t, "Hello, world!", proto.GetText())
	})

	t.Run("image content from file", func(t *testing.T) {
		// Create a temporary image file for testing
		tmpFile, err := os.CreateTemp("", "test-image-*.png")
		assert.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		// Write some test data
		testData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header
		_, err = tmpFile.Write(testData)
		assert.NoError(t, err)
		tmpFile.Close()

		content := NewImageFileContent(tmpFile.Name())
		proto, err := content.toProtoContentData()
		assert.NoError(t, err)
		assert.Equal(t, testData, proto.GetImage())
	})

	t.Run("image content from bytes", func(t *testing.T) {
		imageData := []byte{0xFF, 0xD8, 0xFF} // JPEG header
		content := NewImageContent(imageData)

		proto, err := content.toProtoContentData()
		assert.NoError(t, err)
		assert.Equal(t, imageData, proto.GetImage())
	})

	t.Run("image content from non-existent file", func(t *testing.T) {
		content := NewImageFileContent("/non/existent/file.png")
		_, err := content.toProtoContentData()
		assert.Error(t, err)
	})
}

func TestJobConversion(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		pbJob   *sharedv1.Job
		wantJob *Job
	}{
		{
			name: "completed job with results",
			pbJob: &sharedv1.Job{
				JobUuid:    "job-123",
				CustomerId: "customer-456",
				Status:     sharedv1.JobStatus_JOB_STATUS_COMPLETED,
				Created:    timestamppb.New(now),
				Updated:    timestamppb.New(now.Add(time.Minute)),
				Completed:  timestamppb.New(now.Add(2 * time.Minute)),
				PolicyId:   &[]string{"policy-789"}[0],
				Threshold:  0.75,
				Results: []*sharedv1.JobResult{
					{
						ContentHash: "hash1",
						Report: &sharedv1.PolicyEvaluationReport{
							PolicyId:  "policy-789",
							Threshold: 0.75,
							ReviewResult: &sharedv1.PolicyEvaluationReport_ReviewResult{
								Outcome: sharedv1.Outcome_OUTCOME_TRUE,
								Score:   0.85,
							},
						},
					},
				},
			},
			wantJob: &Job{
				ID:          "job-123",
				CustomerID:  "customer-456",
				Status:      JobStatusCompleted,
				CreatedAt:   now,
				UpdatedAt:   now.Add(time.Minute),
				CompletedAt: now.Add(2 * time.Minute),
			},
		},
		{
			name: "pending job without results",
			pbJob: &sharedv1.Job{
				JobUuid:    "job-456",
				CustomerId: "customer-789",
				Status:     sharedv1.JobStatus_JOB_STATUS_PENDING,
				Created:    timestamppb.New(now),
				Updated:    timestamppb.New(now),
			},
			wantJob: &Job{
				ID:         "job-456",
				CustomerID: "customer-789",
				Status:     JobStatusPending,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
		{
			name: "failed job",
			pbJob: &sharedv1.Job{
				JobUuid:    "job-789",
				CustomerId: "customer-123",
				Status:     sharedv1.JobStatus_JOB_STATUS_FAILED,
				Created:    timestamppb.New(now),
				Updated:    timestamppb.New(now),
			},
			wantJob: &Job{
				ID:         "job-789",
				CustomerID: "customer-123",
				Status:     JobStatusFailed,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := new(Job).fromProto(tt.pbJob)
			assert.Equal(t, tt.wantJob.ID, job.ID)
			assert.Equal(t, tt.wantJob.CustomerID, job.CustomerID)
			assert.Equal(t, tt.wantJob.Status, job.Status)
			assert.Equal(t, tt.wantJob.CreatedAt.Unix(), job.CreatedAt.Unix())
			assert.Equal(t, tt.wantJob.UpdatedAt.Unix(), job.UpdatedAt.Unix())

			if !tt.wantJob.CompletedAt.IsZero() {
				assert.Equal(t, tt.wantJob.CompletedAt.Unix(), job.CompletedAt.Unix())
			}

			if len(tt.pbJob.Results) > 0 {
				assert.Len(t, job.Results, len(tt.pbJob.Results))
			}
		})
	}
}

func TestReportConversion(t *testing.T) {
	report := &sharedv1.PolicyEvaluationReport{
		PolicyId:  "policy-123",
		Threshold: 0.7,
		ReviewResult: &sharedv1.PolicyEvaluationReport_ReviewResult{
			Outcome: sharedv1.Outcome_OUTCOME_TRUE,
			Score:   0.85,
		},
		SectionEvaluationReports: []*sharedv1.PolicyEvaluationReport_SectionEvaluationReport{
			{
				Name:    "toxicity",
				Message: "Content is toxic",
				ReviewResult: &sharedv1.PolicyEvaluationReport_ReviewResult{
					Outcome: sharedv1.Outcome_OUTCOME_TRUE,
					Score:   0.9,
				},
			},
			{
				Name:    "spam",
				Message: "Not spam",
				ReviewResult: &sharedv1.PolicyEvaluationReport_ReviewResult{
					Outcome: sharedv1.Outcome_OUTCOME_FALSE,
					Score:   0.1,
				},
			},
		},
	}

	converted := new(Report).fromProto(report)

	assert.Equal(t, "policy-123", converted.PolicyID)
	assert.Equal(t, 0.7, converted.Threshold)
	assert.Equal(t, OutcomeTrue, converted.Result)
	assert.Len(t, converted.LabelReports, 2)

	// Check label reports
	toxicity, ok := converted.LabelReports["toxicity"]
	assert.True(t, ok)
	assert.Equal(t, "toxicity", toxicity.LabelName)
	assert.Equal(t, "Content is toxic", toxicity.Actions)
	assert.Equal(t, OutcomeTrue, toxicity.Outcome)
	assert.Equal(t, 0.9, toxicity.Score)

	spam, ok := converted.LabelReports["spam"]
	assert.True(t, ok)
	assert.Equal(t, "spam", spam.LabelName)
	assert.Equal(t, "Not spam", spam.Actions)
	assert.Equal(t, OutcomeFalse, spam.Outcome)
	assert.Equal(t, 0.1, spam.Score)

	// Check matches - only labels that exceeded threshold
	assert.Len(t, converted.Matches, 1)
	assert.Equal(t, 0.9, converted.Matches["toxicity"])

	// Check actions
	assert.Contains(t, converted.Actions, "Content is toxic")
}

func TestRetryWithContext(t *testing.T) {
	t.Run("context timeout during retry", func(t *testing.T) {
		callCount := 0
		mockClient := &mockGatewayClient{
			createJobFunc: func(ctx context.Context, req *gatewayv1.CreateJobRequest, opts ...grpc.CallOption) (*gatewayv1.CreateJobResponse, error) {
				callCount++
				return nil, status.Error(codes.ResourceExhausted, "rate limited")
			},
		}

		client := &Client{
			config: &cfg{
				apiKey: "test-key",
				retryConfig: RetryConfig{
					MaxRetries:      5,
					InitialInterval: 100 * time.Millisecond,
					MaxInterval:     1 * time.Second,
					Multiplier:      2.0,
				},
			},
			gateway: mockClient,
		}

		// Create a context that times out quickly
		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()

		_, err := client.CreateJob(ctx, &CreateJobRequest{
			PolicyID: "test",
			Content:  []Contenter{NewTextContent("test")},
		})

		assert.Error(t, err)
		// The error could be either context deadline exceeded or the rate limit error
		// depending on timing
		assert.True(t,
			strings.Contains(err.Error(), "context deadline exceeded") ||
				strings.Contains(err.Error(), "rate limited"),
			"Expected either context deadline exceeded or rate limited error, got: %v", err)
		// Should have tried at least once but not all retries
		assert.GreaterOrEqual(t, callCount, 1)
		assert.Less(t, callCount, 5)
	})
}

func TestBuilders(t *testing.T) {
	t.Run("CreateJobRequestBuilder", func(t *testing.T) {
		req := NewCreateJobRequestBuilder().
			PolicyID("policy-123").
			AddContent(NewTextContent("hello"), NewTextContent("world")).
			Threshold(0.8).
			Expedited(true).
			Webhook("https://example.com/webhook").
			Build()

		assert.Equal(t, "policy-123", req.PolicyID)
		assert.Len(t, req.Content, 2)
		assert.Equal(t, 0.8, req.Options.Threshold)
		assert.True(t, req.Options.Expedited)
		assert.Equal(t, "https://example.com/webhook", req.WebhookURL)
	})

	t.Run("EvaluateRequestBuilder", func(t *testing.T) {
		req := NewEvaluateRequestBuilder().
			PolicyID("policy-456").
			AddContent(NewImageContent([]byte{0xFF, 0xD8})).
			Threshold(0.7).
			Build()

		assert.Equal(t, "policy-456", req.PolicyID)
		assert.Len(t, req.Content, 1)
		assert.Equal(t, 0.7, req.Options.Threshold)
	})

	t.Run("ListJobsRequestBuilder", func(t *testing.T) {
		req := NewListJobsRequestBuilder().
			Query(ListJobsQuery{
				PolicyID: "policy-789",
				Status:   JobStatusRunning,
			}).
			PageSize(50).
			PageToken("token-123").
			Build()

		assert.Equal(t, "policy-789", req.Query.PolicyID)
		assert.Equal(t, JobStatusRunning, req.Query.Status)
		assert.Equal(t, 50, req.PageSize)
		assert.Equal(t, "token-123", req.PageToken)
	})
}

// Add this to ensure proper import handling
func TestImports(t *testing.T) {
	// This test ensures all imports are used
	_ = os.Open
}
