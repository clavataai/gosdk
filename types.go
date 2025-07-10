package clavata

import (
	"context"
	"fmt"
	"os"
	"time"

	gatewayv1 "github.com/clavataai/gosdk/internal/protobufs/gateway/v1"
	sharedv1 "github.com/clavataai/gosdk/internal/protobufs/shared/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// JobStatus represents the status of a job
type JobStatus string

const (
	// JobStatusUnspecified is the zero value. Should not be used.
	JobStatusUnspecified JobStatus = "UNSPECIFIED"
	// JobStatusPending means the job is in queue and will be evaluated soon.
	JobStatusPending JobStatus = "PENDING"
	// JobStatusRunning means the job is being evaluated.
	JobStatusRunning JobStatus = "RUNNING"
	// JobStatusCompleted means the job has been evaluated and the results are available.
	JobStatusCompleted JobStatus = "COMPLETED"
	// JobStatusFailed means the job failed to evaluate.
	JobStatusFailed JobStatus = "FAILED"
	// JobStatusCanceled means the job was canceled, typically by the user.
	JobStatusCanceled JobStatus = "CANCELED"
)

// String returns the string representation of the job status.
func (j JobStatus) String() string {
	return string(j)
}

func (j JobStatus) fromProto(proto sharedv1.JobStatus) JobStatus {
	switch proto {
	case sharedv1.JobStatus_JOB_STATUS_UNSPECIFIED:
		return JobStatusUnspecified
	case sharedv1.JobStatus_JOB_STATUS_PENDING:
		return JobStatusPending
	case sharedv1.JobStatus_JOB_STATUS_RUNNING:
		return JobStatusRunning
	case sharedv1.JobStatus_JOB_STATUS_COMPLETED:
		return JobStatusCompleted
	case sharedv1.JobStatus_JOB_STATUS_FAILED:
		return JobStatusFailed
	case sharedv1.JobStatus_JOB_STATUS_CANCELED:
		return JobStatusCanceled
	}
	return JobStatusUnspecified
}

func (j JobStatus) toProto() sharedv1.JobStatus {
	switch j {
	case JobStatusUnspecified:
		return sharedv1.JobStatus_JOB_STATUS_UNSPECIFIED
	case JobStatusPending:
		return sharedv1.JobStatus_JOB_STATUS_PENDING
	case JobStatusRunning:
		return sharedv1.JobStatus_JOB_STATUS_RUNNING
	case JobStatusCompleted:
		return sharedv1.JobStatus_JOB_STATUS_COMPLETED
	case JobStatusFailed:
		return sharedv1.JobStatus_JOB_STATUS_FAILED
	case JobStatusCanceled:
		return sharedv1.JobStatus_JOB_STATUS_CANCELED
	}
	return sharedv1.JobStatus_JOB_STATUS_UNSPECIFIED
}

// Contenter is an interface that represents a content input to send to the API.
type Contenter interface {
	// AddMetadata allows you to add a key/value pair to the content. This metadata will be included
	// in the `Report` for this piece of content, allowing you to include IDs or other information
	// that is meaningful to your system.
	AddMetadata(key, value string) Contenter

	// toProtoContentData prepares the content for transmission to the API.
	toProtoContentData() (*sharedv1.ContentData, error)
}

// NewTextContent can be used to create a text input to send to the API.
func NewTextContent(text string) Contenter {
	return &textContent{text: text}
}

type textContent struct {
	text     string
	metadata map[string]string
}

// AddMetadata allows you to add a single key/value pair to the content. It can be called multiple
// times to add multiple key/value pairs, but if two keys are the same, the last value supplied will
// take precedence.
func (t *textContent) AddMetadata(key, value string) Contenter {
	t.metadata[key] = value
	return t
}

func (t *textContent) toProtoContentData() (*sharedv1.ContentData, error) {
	return &sharedv1.ContentData{
		Content:  &sharedv1.ContentData_Text{Text: t.text},
		Metadata: t.metadata,
	}, nil
}

// NewImageContent can be used to create an image input to send to the API.
func NewImageContent(imageData []byte) Contenter {
	return &imageContent{imageData: imageData}
}

type imageContent struct {
	imageData []byte
	metadata  map[string]string
}

// AddMetadata allows you to add a single key/value pair to the content. It can be called multiple
// times to add multiple key/value pairs, but if two keys are the same, the last value supplied will
// take precedence.
func (i *imageContent) AddMetadata(key, value string) Contenter {
	i.metadata[key] = value
	return i
}

func (i *imageContent) toProtoContentData() (*sharedv1.ContentData, error) {
	return &sharedv1.ContentData{
		Content:  &sharedv1.ContentData_Image{Image: i.imageData},
		Metadata: i.metadata,
	}, nil
}

// NewImageFileContent can be used to create an image input from a file available on the local file
// system to send to the API.
func NewImageFileContent(imageFile string) Contenter {
	return &imageFileContent{imageFile: imageFile}
}

type imageFileContent struct {
	imageFile string
	metadata  map[string]string
}

// AddMetadata allows you to add a single key/value pair to the content. It can be called multiple
// times to add multiple key/value pairs, but if two keys are the same, the last value supplied will
// take precedence.
func (i *imageFileContent) AddMetadata(key, value string) Contenter {
	i.metadata[key] = value
	return i
}

func (i *imageFileContent) toProtoContentData() (*sharedv1.ContentData, error) {
	// Load the image file from the file system
	imageData, err := os.ReadFile(i.imageFile)
	if err != nil {
		return nil, err
	}
	return &sharedv1.ContentData{
		Content:  &sharedv1.ContentData_Image{Image: imageData},
		Metadata: i.metadata,
	}, nil
}

// NewImageURLContent can be used to create an image input from a URL to send to the API.
func NewImageURLContent(imageURL string) Contenter {
	return &imageURLContent{imageURL: imageURL}
}

type imageURLContent struct {
	imageURL string
	metadata map[string]string
}

// AddMetadata allows you to add a single key/value pair to the content. It can be called multiple
// times to add multiple key/value pairs, but if two keys are the same, the last value supplied will
// take precedence.
func (i *imageURLContent) AddMetadata(key, value string) Contenter {
	i.metadata[key] = value
	return i
}

func (i *imageURLContent) toProtoContentData() (*sharedv1.ContentData, error) {
	return &sharedv1.ContentData{
		Content:  &sharedv1.ContentData_ImageUrl{ImageUrl: i.imageURL},
		Metadata: i.metadata,
	}, nil
}

// JobOptions allow you to configure certain aspects of how the job is processed.
type JobOptions struct {
	// If set, the job will be evaluated with the given threshold. A threshold of 0.0 will be
	// considered unset and the server default will be used (0.50).
	Threshold float64
	// If set, this will prioritze speed over accuracy. In most cases, the accuracy difference is
	// minimal. For image content, expedited mode generally improves latency by 20-30%.
	Expedited bool
}

// EvaluateRequest is the request type for the Evaluate method.
type EvaluateRequest struct {
	// The content to evaluate.
	Content []Contenter
	// The ID of the policy to evaluate all of the supplied content against.
	PolicyID string
	// Options that will impact how the job is evaluated.
	Options JobOptions
}

func (e *EvaluateRequest) toProto() (*gatewayv1.EvaluateRequest, error) {
	contentData := make([]*sharedv1.ContentData, len(e.Content))
	for i, content := range e.Content {
		cd, err := content.toProtoContentData()
		if err != nil {
			return nil, fmt.Errorf("failed to prepare content: %w", err)
		}
		contentData[i] = cd
	}

	var threshold *float64
	if e.Options.Threshold != 0 {
		threshold = &e.Options.Threshold
	}

	var expedited *bool
	if e.Options.Expedited {
		expedited = &e.Options.Expedited
	}

	return &gatewayv1.EvaluateRequest{
		ContentData: contentData,
		PolicyId:    e.PolicyID,
		Threshold:   threshold,
		Expedited:   expedited,
	}, nil
}

// CreateJobRequest is the request type for the CreateJob method.
type CreateJobRequest struct {
	// The ID of the policy to evaluate the content against.
	PolicyID string
	// The content to evaluate.
	Content []Contenter
	// The options for the job.
	Options JobOptions
	// WebhookURL is the URL to call when the job is complete. Must be HTTPS and will be called with POST.
	WebhookURL string
}

func (c *CreateJobRequest) toProto() (*gatewayv1.CreateJobRequest, error) {
	contentData := make([]*sharedv1.ContentData, len(c.Content))
	for i, content := range c.Content {
		cd, err := content.toProtoContentData()
		if err != nil {
			return nil, fmt.Errorf("failed to prepare content: %w", err)
		}
		contentData[i] = cd
	}

	var threshold *float64
	if c.Options.Threshold != 0 {
		threshold = &c.Options.Threshold
	}

	var expedited *bool
	if c.Options.Expedited {
		expedited = &c.Options.Expedited
	}

	var webhook *gatewayv1.CreateJobRequest_Webhook
	if c.WebhookURL != "" {
		webhook = &gatewayv1.CreateJobRequest_Webhook{
			Url: c.WebhookURL,
		}
	}

	return &gatewayv1.CreateJobRequest{
		PolicyId:    c.PolicyID,
		ContentData: contentData,
		Threshold:   threshold,
		Expedited:   expedited,
		Webhook:     webhook,
	}, nil
}

// An Outcome is how we represent the result of an evaluation for labels and the overall policy that contains them.
// While outcomes are generally TRUE or FALSE, they can also have a FAILED status. This is rare, but when it happens
// it means that a particular label(s) failed to evaluate.
type Outcome string

const (
	// OutcomeUnspecified is the zero value. Should not be used.
	OutcomeUnspecified Outcome = "UNSPECIFIED"
	// OutcomeTrue means the label or policy evaluation was true.
	OutcomeTrue Outcome = "TRUE"
	// OutcomeFalse means the label or policy evaluation was false.
	OutcomeFalse Outcome = "FALSE"
	// OutcomeFailed means the label or policy evaluation failed.
	OutcomeFailed Outcome = "FAILED"
)

// String returns the string representation of the outcome.
func (o Outcome) String() string {
	return string(o)
}

func (o Outcome) fromProto(proto sharedv1.Outcome) Outcome {
	switch proto {
	case sharedv1.Outcome_OUTCOME_UNSPECIFIED:
		return OutcomeUnspecified
	case sharedv1.Outcome_OUTCOME_TRUE:
		return OutcomeTrue
	case sharedv1.Outcome_OUTCOME_FALSE:
		return OutcomeFalse
	case sharedv1.Outcome_OUTCOME_FAILED:
		return OutcomeFailed
	}
	return OutcomeUnspecified
}

// LabelReport provides details about the evaluation of a single piece of content against a single label's definition.
type LabelReport struct {
	// LabelName is the name of the label that was evaluated as defined in the policy. This field
	// will always match the key it is associated with in the `Matches` map of the `Report`.
	LabelName string
	// Score is the score of the label. Can be between 0.0 and 1.0. In rare cases, the score may be
	// -1.0, which means that evaluation failed.
	Score float64
	// Outcome is the outcome of the label. If the score is greater than the threshold, the outcome
	// will be TRUE.
	Outcome Outcome
	// Actions is the string that was associated with the `LABEL` in the policy.
	//
	// Example:
	// LABEL "Foo": "Action1, Action2" { ... }
	//
	// In this case, the Actions field will contain "Action1, Action2".
	Actions string
}

func (l *LabelReport) fromProto(proto *sharedv1.PolicyEvaluationReport_SectionEvaluationReport) *LabelReport {
	l.LabelName = proto.GetName()
	l.Score = proto.GetReviewResult().GetScore()
	l.Outcome = Outcome("").fromProto(proto.GetReviewResult().GetOutcome())
	l.Actions = proto.GetMessage()
	return l
}

// A Report provides the full details of evaluating a single piece of content against a policy.
type Report struct {
	// The ID of the policy that was evaluated.
	PolicyID string
	// The ID of the policy version that was evaluated.
	PolicyVersionID string
	// The FNV-1a hash of the content.
	ContentHash string
	// ContentMetadata will be filled with any metadata that was attached to the content at the time it was created.
	// This can be used to associate an ID your system understands with the report, allowing you to match results
	// to content inputs without the need to calculate the hash.
	ContentMetadata map[string]string
	// The overall outcome of the evaluation. If any label evaluated to true, the result will be true.
	Result Outcome
	// The threshold that was used to decide whether an outcome was true or false.
	Threshold float64

	// Matches is the list of labels that matched the content. Only labels with scores that exceed the threshold
	// will be included in this map. Each key is the label name, and the value is the score of the label.
	Matches map[string]float64
	// Actions is the list of actions associated with the labels that exceeded the threshold. If the same action
	// appears on more than one label, it is only included in this list once.
	Actions []string

	// The raw label reports. All labels are included, regardless of whether they exceeded the threshold. Each
	// key is the label name, and the value is the report, which includes the score the label received, the outcome
	// when the threshold was applied and the actions that were associated with the label.
	LabelReports map[string]LabelReport
}

func (r *Report) fromProto(proto *sharedv1.PolicyEvaluationReport) *Report {
	r.PolicyID = proto.PolicyId
	r.PolicyVersionID = proto.PolicyVersionId
	r.ContentHash = proto.ContentHash
	r.ContentMetadata = proto.ContentMetadata
	r.Result = Outcome("").fromProto(proto.ReviewResult.Outcome)
	r.Threshold = proto.GetThreshold()

	matches := make(map[string]float64)
	labelReports := make(map[string]LabelReport)
	actionsSet := make(map[string]struct{})
	for _, ser := range proto.SectionEvaluationReports {
		labelReports[ser.GetName()] = *(new(LabelReport).fromProto(ser))

		if ser.GetReviewResult().GetOutcome() == sharedv1.Outcome_OUTCOME_TRUE {
			matches[ser.GetName()] = ser.GetReviewResult().GetScore()
			actionsSet[ser.GetMessage()] = struct{}{}
		}
	}

	r.Matches = matches
	r.LabelReports = labelReports

	// Now we just need to build the actions list.
	r.Actions = make([]string, 0, len(actionsSet))
	for action := range actionsSet {
		r.Actions = append(r.Actions, action)
	}
	return r
}

// A Job is the result of a request to CreateJob. Each job will contain as many reports as there were pieces of content
// included in the job request.
type Job struct {
	ID          string
	CustomerID  string
	Status      JobStatus
	Results     []Report
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt time.Time
}

func (j *Job) fromProto(proto *sharedv1.Job) *Job {
	j.ID = proto.GetJobUuid()
	j.CustomerID = proto.CustomerId
	j.Status = JobStatus("").fromProto(proto.GetStatus())
	j.CreatedAt = proto.GetCreated().AsTime()
	j.UpdatedAt = proto.GetUpdated().AsTime()
	j.CompletedAt = proto.GetCompleted().AsTime()

	j.Results = make([]Report, len(proto.GetResults()))
	for i, result := range proto.GetResults() {
		j.Results[i] = *(new(Report).fromProto(result.GetReport()))
	}
	return j
}

// A TimeRange is a range of time. Used to filter jobs by creation, update, or completion time. Note
// that if you set `Start` or `End` but not both, the resulting filter will be treated like a simple
// greater than or less than filter rather than requiring that the time is within the range.
type TimeRange struct {
	// Start is the start of the time range.
	Start time.Time
	// End is the end of the time range.
	End time.Time
	// Inclusive is whether the range is inclusive. If true, the range will include the start and
	// end times. Ranges are exclusive by default.
	Inclusive bool
}

func (t *TimeRange) toProto() *sharedv1.TimeRange {
	return &sharedv1.TimeRange{
		Start:     timestamppb.New(t.Start),
		End:       timestamppb.New(t.End),
		Inclusive: t.Inclusive,
	}
}

// ListJobsQuery is the query type for the ListJobs method.
type ListJobsQuery struct {
	// Filter on the time the job was created.
	CreatedAt TimeRange
	// Filter on the time the job was last updated.
	UpdatedAt TimeRange
	// Filter on the time the job was completed.
	CompletedAt TimeRange
	// Filter on the status of the job.
	Status JobStatus
	// If set, only jobs that used this PolicyID will be returned.
	PolicyID string
}

func (l *ListJobsQuery) toProto() *gatewayv1.ListJobsRequest_Query {
	var policyID *string
	if l.PolicyID != "" {
		policyID = &l.PolicyID
	}

	return &gatewayv1.ListJobsRequest_Query{
		CreatedTimeRange:   l.CreatedAt.toProto(),
		UpdatedTimeRange:   l.UpdatedAt.toProto(),
		CompletedTimeRange: l.CompletedAt.toProto(),
		Status:             l.Status.toProto(),
		PolicyId:           policyID,
	}
}

// ListJobsRequest is the request type for the ListJobs method.
type ListJobsRequest struct {
	// The query to use to filter the jobs.
	Query ListJobsQuery
	// The maximum number of jobs to return on a single page. If set to `0` this field will be
	// considered unset and the default will be used.(Default: 1000)
	PageSize int
	// The token to use to get the next page of results. This token will be returned by the server
	// in the `NextPageToken` field of the response. It should be passed back to the server in the
	// `PageToken` field of the next request to get the next page of results.
	PageToken string
}

// NewListJobsRequest creates a new ListJobsRequest with the given query and page size.
func NewListJobsRequest(query ListJobsQuery, pageSize int) *ListJobsRequest {
	return &ListJobsRequest{
		Query:    query,
		PageSize: pageSize,
	}
}

func (l *ListJobsRequest) toProto() (*gatewayv1.ListJobsRequest, error) {
	query := l.Query.toProto()

	return &gatewayv1.ListJobsRequest{
		Query:     query,
		PageSize:  int32(l.PageSize),
		PageToken: l.PageToken,
	}, nil
}

// NextPage creates a new ListJobsRequest with the same query and page size, but with the page token
// set to the provided token. This can be used to get the next page of results from the current response.
//
// Example:
//
//	builder := NewListJobsRequestBuilder()
//	req := builder.Query(ListJobsQuery{Status: JobStatusRunning}).Build()
//	resp, err := c.ListJobs(ctx, req)
//	if err != nil {
//		return err
//	}
//
//	nextPageReq := req.NextPage(resp)
//	resp, err = c.ListJobs(ctx, nextPageReq)
func (l *ListJobsRequest) NextPage(resp *ListJobsResponse) *ListJobsRequest {
	return &ListJobsRequest{
		Query:     l.Query,
		PageSize:  l.PageSize,
		PageToken: resp.NextPageToken,
	}
}

// ListJobsResponse is the response type for the ListJobs method. It contains the list of jobs that match the query,
// and a token to use to get the next page of results.
type ListJobsResponse struct {
	Jobs          []Job
	NextPageToken string
}

func (l *ListJobsResponse) fromProto(proto *gatewayv1.ListJobsResponse) *ListJobsResponse {
	l.Jobs = make([]Job, len(proto.GetJobs()))
	for i, job := range proto.GetJobs() {
		l.Jobs[i] = *(new(Job).fromProto(job))
	}
	l.NextPageToken = proto.GetNextPageToken()
	return l
}

// NextPage is a helper method that can be used to get the next page of results from the current
// response. You'll need to provide it with the Clavata client instance and previous request.
func (l *ListJobsResponse) NextPage(
	ctx context.Context, c *Client, req *ListJobsRequest,
) (*ListJobsResponse, error) {
	req.PageToken = l.NextPageToken
	return c.ListJobs(ctx, req)
}

// Builders for requests
//
// These are provided to make it easier to construct requests.
//
// Example:
//
//	builder := NewCreateJobRequestBuilder()
//	builder.PolicyID("policy-id").Content([]Contenter{NewTextContent("Hello, world!")}).Build()

// CreateJobRequestBuilder simplifies the construction of a CreateJobRequest.
// First, create the builder with `NewCreateJobRequestBuilder()`.
// Then, use the builder to set the policy ID, content, options, and webhook.
// Finally, call `Build()` to create the request.
//
// Example:
//
//	builder := NewCreateJobRequestBuilder()
//	req := builder.PolicyID("policy-id").AddContent(NewTextContent("Hello, world!")).Build()
//
// You can then use the request with the CreateJob method.
type CreateJobRequestBuilder struct {
	policyID string
	content  []Contenter
	options  JobOptions
	webhook  string
}

// NewCreateJobRequestBuilder creates a new CreateJobRequestBuilder.
func NewCreateJobRequestBuilder() *CreateJobRequestBuilder {
	return &CreateJobRequestBuilder{
		content: make([]Contenter, 0, 10),
	}
}

// PolicyID sets the policy ID for the request.
func (b *CreateJobRequestBuilder) PolicyID(policyID string) *CreateJobRequestBuilder {
	b.policyID = policyID
	return b
}

// AddContent adds content to the request.
func (b *CreateJobRequestBuilder) AddContent(content ...Contenter) *CreateJobRequestBuilder {
	if b.content == nil {
		b.content = make([]Contenter, 0, len(content))
	}

	b.content = append(b.content, content...)
	return b
}

// Options sets the options for the request.
func (b *CreateJobRequestBuilder) Options(options JobOptions) *CreateJobRequestBuilder {
	b.options = options
	return b
}

// Threshold sets the threshold for the request.
func (b *CreateJobRequestBuilder) Threshold(threshold float64) *CreateJobRequestBuilder {
	b.options.Threshold = threshold
	return b
}

// Expedited sets the expedited flag for the request.
func (b *CreateJobRequestBuilder) Expedited(expedited bool) *CreateJobRequestBuilder {
	b.options.Expedited = expedited
	return b
}

// Webhook sets the webhook for the request.
func (b *CreateJobRequestBuilder) Webhook(webhook string) *CreateJobRequestBuilder {
	b.webhook = webhook
	return b
}

// Build creates a new CreateJobRequest from the builder.
func (b *CreateJobRequestBuilder) Build() *CreateJobRequest {
	return &CreateJobRequest{
		PolicyID:   b.policyID,
		Content:    b.content,
		Options:    b.options,
		WebhookURL: b.webhook,
	}
}

// EvaluateRequestBuilder simplifies the construction of an EvaluateRequest.
// First, create the builder with `NewEvaluateRequestBuilder()`.
// Then, use the builder to set the policy ID, content, and options.
// Finally, call `Build()` to create the request.
//
// Example:
//
//	builder := NewEvaluateRequestBuilder()
//	req := builder.PolicyID("policy-id").AddContent(NewTextContent("Hello, world!")).Build()
//
// You can then use the request with the Evaluate method.
type EvaluateRequestBuilder struct {
	policyID string
	content  []Contenter
	options  JobOptions
}

// NewEvaluateRequestBuilder creates a new EvaluateRequestBuilder.
func NewEvaluateRequestBuilder() *EvaluateRequestBuilder {
	return &EvaluateRequestBuilder{
		content: make([]Contenter, 0, 10),
	}
}

// PolicyID sets the policy ID for the request.
func (b *EvaluateRequestBuilder) PolicyID(policyID string) *EvaluateRequestBuilder {
	b.policyID = policyID
	return b
}

// AddContent adds content to the request.
func (b *EvaluateRequestBuilder) AddContent(content ...Contenter) *EvaluateRequestBuilder {
	if b.content == nil {
		b.content = make([]Contenter, 0, len(content))
	}

	b.content = append(b.content, content...)
	return b
}

// Options sets the options for the request.
func (b *EvaluateRequestBuilder) Options(options JobOptions) *EvaluateRequestBuilder {
	b.options = options
	return b
}

// Threshold sets the threshold for the request.
func (b *EvaluateRequestBuilder) Threshold(threshold float64) *EvaluateRequestBuilder {
	b.options.Threshold = threshold
	return b
}

// Expedited sets the expedited flag for the request.
func (b *EvaluateRequestBuilder) Expedited(expedited bool) *EvaluateRequestBuilder {
	b.options.Expedited = expedited
	return b
}

// Build creates a new EvaluateRequest from the builder.
func (b *EvaluateRequestBuilder) Build() *EvaluateRequest {
	return &EvaluateRequest{
		PolicyID: b.policyID,
		Content:  b.content,
		Options:  b.options,
	}
}

// ListJobsRequestBuilder simplifies the construction of a ListJobsRequest. First, create the
// builder with `NewListJobsRequestBuilder()`. Then, use the builder to set the query, page size,
// and page token. Finally, call `Build()` to create the request.
//
// Example:
//
//	builder := NewListJobsRequestBuilder()
//	req := builder.Query(ListJobsQuery{Status: JobStatusRunning}).Build()
//
// This builder, in particular, can be helpful for dealing with pagination as it makes it easy to
// build requests with the same query and page size, but an updated page token.
// For example:
//
//	builder := NewListJobsRequestBuilder()
//	builder = builder.Query(ListJobsQuery{Status: JobStatusRunning}).PageSize(100)
//	resp, err := c.ListJobs(ctx, builder.Build())
//	if err != nil {
//		return err
//	}
//
//	resp, err = c.ListJobs(ctx, builder.NextPage(resp).Build())
//	if err != nil {
//		return err
//	}
//
// And so on.
type ListJobsRequestBuilder struct {
	query     ListJobsQuery
	pageSize  int
	pageToken string
}

// NewListJobsRequestBuilder creates a new ListJobsRequestBuilder.
func NewListJobsRequestBuilder() *ListJobsRequestBuilder {
	return &ListJobsRequestBuilder{}
}

// Query sets the query for the request.
func (b *ListJobsRequestBuilder) Query(query ListJobsQuery) *ListJobsRequestBuilder {
	b.query = query
	return b
}

// PageSize sets the page size for the request.
func (b *ListJobsRequestBuilder) PageSize(pageSize int) *ListJobsRequestBuilder {
	b.pageSize = pageSize
	return b
}

// PageToken sets the page token for the request.
func (b *ListJobsRequestBuilder) PageToken(pageToken string) *ListJobsRequestBuilder {
	b.pageToken = pageToken
	return b
}

// Build creates a new ListJobsRequest from the builder.
func (b *ListJobsRequestBuilder) Build() *ListJobsRequest {
	return &ListJobsRequest{
		Query:     b.query,
		PageSize:  b.pageSize,
		PageToken: b.pageToken,
	}
}
