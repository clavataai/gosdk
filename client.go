package clavata

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	gatewayv1 "github.com/clavataai/monorail/libs/protobufs/gateway/v1"
)

const (
	defaultEndpoint = "gateway.app.clavata.ai:443"
	defaultTimeout  = 30 * time.Second
)

// RetryConfig configures retry behavior for rate-limited requests
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts (0 means no retry)
	MaxRetries uint64
	// InitialInterval is the initial backoff interval
	InitialInterval time.Duration
	// MaxInterval is the maximum backoff interval between retries.
	MaxInterval time.Duration
	// Multiplier is the backoff multiplier (e.g., 2.0 for exponential backoff)
	Multiplier float64
	// RandomizationFactor adds jitter to prevent thundering herd
	RandomizationFactor float64
}

// DefaultRetryConfig returns our recommended retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      5,
		InitialInterval: 500 * time.Millisecond,
		MaxInterval:     30 * time.Second,
		// A multiplier of 2.0 gives us
		Multiplier: 2.0,
		// A high randomization factor is recommended to prevent thundering herd.
		RandomizationFactor: 0.65,
	}
}

// option is a function that configures the client
type option func(*cfg)

// WithAPIKey sets the API key for the client. If you do not have an API key, please reach
// out to support@clavata.ai to request one.
func WithAPIKey(apiKey string) option {
	return func(c *cfg) {
		c.apiKey = apiKey
	}
}

// WithEndpoint sets the endpoint for the client. Unless you have been told to set
// a different endpoint, there's no need to set this flag.
func WithEndpoint(endpoint string) option {
	return func(c *cfg) {
		c.endpoint = endpoint
	}
}

// WithInsecure sets the insecure flag for the client. This is only used by Clavata
// for testing purposes and should not be used in production.
func WithInsecure(insecure bool) option {
	return func(c *cfg) {
		c.insecure = insecure
	}
}

// WithTimeout sets the default timeout for requests. If not set, the default
// timeout is 30 seconds.
func WithTimeout(timeout time.Duration) option {
	return func(c *cfg) {
		c.timeout = timeout
	}
}

// WithRetryConfig sets custom retry configuration for the client
func WithRetryConfig(retryConfig RetryConfig) option {
	return func(c *cfg) {
		c.retryConfig = retryConfig
	}
}

// WithDisableRetry disables automatic retry on rate limits
func WithDisableRetry() option {
	return func(c *cfg) {
		c.retryConfig.MaxRetries = 0
	}
}

// cfg holds configuration for the Clavata client
type cfg struct {
	// apiKey is your Clavata API key
	apiKey string
	// endpoint is the Clavata API endpoint (default: "gateway.app.clavata.ai:8443")
	endpoint string
	// insecure disables TLS verification (for development only)
	insecure bool
	// timeout is the default timeout for requests
	timeout time.Duration
	// retryConfig configures retry behavior for rate-limited requests
	retryConfig RetryConfig
}

// Client is the main Clavata SDK client
type Client struct {
	config  *cfg
	conn    *grpc.ClientConn
	gateway gatewayv1.GatewayServiceClient
}

func commonAuthInterceptor(ctx context.Context, apiKey string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+apiKey)
}

// isRetriableError checks if the error is retriable (rate limit)
func isRetriableError(err error) bool {
	if err == nil {
		return false
	}

	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	// Only retry on RESOURCE_EXHAUSTED (rate limit) or UNAVAILABLE (which may suggest a resolvable issue)
	return st.Code() == codes.ResourceExhausted || st.Code() == codes.Unavailable
}

// createBackoff creates a configured exponential backoff
func createBackoff(config RetryConfig) backoff.BackOff {
	if config.MaxRetries == 0 {
		return &backoff.StopBackOff{}
	}

	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = config.InitialInterval
	expBackoff.MaxInterval = config.MaxInterval
	expBackoff.Multiplier = config.Multiplier
	expBackoff.RandomizationFactor = config.RandomizationFactor
	expBackoff.MaxElapsedTime = 0 // We control retries with WithMaxRetries

	return backoff.WithMaxRetries(expBackoff, config.MaxRetries)
}

func unaryInterceptorFactory(apiKey string, retryConfig RetryConfig) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Add auth to context
		ctx = commonAuthInterceptor(ctx, apiKey)

		// Create backoff for this request
		b := createBackoff(retryConfig)

		// Retry loop with exponential backoff
		return backoff.Retry(func() error {
			err := invoker(ctx, method, req, reply, cc, opts...)
			if err != nil && isRetriableError(err) {
				// Return the error to trigger backoff
				return err
			}
			// For non-retriable errors or success, stop retrying
			if err != nil {
				return backoff.Permanent(err)
			}
			return nil
		}, backoff.WithContext(b, ctx))
	}
}

func streamInterceptorFactory(apiKey string, retryConfig RetryConfig) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		// Add auth to context
		ctx = commonAuthInterceptor(ctx, apiKey)

		// For streaming, we can only retry the initial connection
		// Create backoff for this request
		b := createBackoff(retryConfig)

		var stream grpc.ClientStream
		err := backoff.Retry(func() error {
			var err error
			stream, err = streamer(ctx, desc, cc, method, opts...)
			if err != nil && isRetriableError(err) {
				// Return the error to trigger backoff
				return err
			}
			// For non-retriable errors or success, stop retrying
			if err != nil {
				return backoff.Permanent(err)
			}

			return nil
		}, backoff.WithContext(b, ctx))

		return stream, err
	}
}

// New creates a new Clavata client
func New(options ...option) (*Client, error) {
	config := &cfg{
		endpoint:    defaultEndpoint,
		timeout:     defaultTimeout,
		insecure:    false,
		retryConfig: DefaultRetryConfig(),
	}

	for _, option := range options {
		option(config)
	}

	if config.apiKey == "" {
		return nil, ErrAPIKeyRequired
	}

	// Create gRPC connection
	var creds credentials.TransportCredentials
	if config.insecure {
		creds = insecure.NewCredentials()
	} else {
		creds = credentials.NewTLS(&tls.Config{})
	}

	conn, err := grpc.NewClient(
		config.endpoint,
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                2 * time.Minute,
			Timeout:             10 * time.Second,
			PermitWithoutStream: false,
		}),
		grpc.WithUnaryInterceptor(unaryInterceptorFactory(config.apiKey, config.retryConfig)),
		grpc.WithStreamInterceptor(streamInterceptorFactory(config.apiKey, config.retryConfig)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for Clavata API: %w", err)
	}

	client := &Client{
		config:  config,
		conn:    conn,
		gateway: gatewayv1.NewGatewayServiceClient(conn),
	}

	return client, nil
}

// Close closes the client connection. You can do this with defer to ensure that the connection
// is always cleaned up.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

var (
	cleanupHandlers []func()
	cleanupMutex    sync.Mutex
	cleanupOnce     sync.Once
)

// setupCleanupHandler sets up a signal handler for cleanup functions
func setupCleanupHandler() {
	cleanupOnce.Do(func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			cleanupMutex.Lock()
			defer cleanupMutex.Unlock()
			for _, handler := range cleanupHandlers {
				handler()
			}
			os.Exit(0)
		}()
	})
}

// addCleanupHandler adds a cleanup function to be called on exit
func addCleanupHandler(handler func()) {
	cleanupMutex.Lock()
	defer cleanupMutex.Unlock()
	cleanupHandlers = append(cleanupHandlers, handler)
	setupCleanupHandler()
}

// CloseOnExit registers the client for cleanup. This can be useful if you are using a long
// lived instance of the client and want to make sure it is always closed before exit.
func (c *Client) CloseOnExit() {
	addCleanupHandler(func() {
		c.Close()
	})
}

// CreateJob creates a new content evaluation job
func (c *Client) CreateJob(ctx context.Context, req *CreateJobRequest) (*Job, error) {
	protoReq, err := req.toProto()
	if err != nil {
		return nil, fmt.Errorf("failed to convert request to proto: %w", err)
	}

	resp, err := c.gateway.CreateJob(ctx, protoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	return new(Job).fromProto(resp.GetJob()), nil
}

// GetJob retrieves a job by its UUID
func (c *Client) GetJob(ctx context.Context, jobID string) (*Job, error) {
	protoReq := &gatewayv1.GetJobRequest{
		JobUuid: jobID,
	}

	resp, err := c.gateway.GetJob(ctx, protoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return new(Job).fromProto(resp.GetJob()), nil
}

// ListJobs lists jobs with optional filtering
func (c *Client) ListJobs(ctx context.Context, req *ListJobsRequest) (*ListJobsResponse, error) {
	protoReq, err := req.toProto()
	if err != nil {
		return nil, fmt.Errorf("failed to convert request to proto: %w", err)
	}

	resp, err := c.gateway.ListJobs(ctx, protoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	return new(ListJobsResponse).fromProto(resp), nil
}

// EvaluateIter creates a streaming evaluation job and returns a go iterator. You can loop over it
// with a for..range loop to receive the reports as they become available. The convenience of this
// function is that each iteration loop will return an error if one occurred. Otherwise, the report
// will be returned as the first value.
func (c *Client) EvaluateIter(ctx context.Context, req *EvaluateRequest) iter.Seq2[*Report, error] {
	ch, callErr := c.Evaluate(ctx, req)
	return func(yield func(*Report, error) bool) {
		if callErr != nil {
			yield(nil, callErr)
			return
		}

		for result := range ch {
			// If the caller doesn't want to continue, we stop the iterator.
			if !yield(result.Report, result.Error) {
				return
			}
		}
	}
}

// EvaluateResult is the result of an evaluation. If an error occurs, the Error field will be set.
// Otherwise, the Report field will be set to the report for the current piece of content.
type EvaluateResult struct {
	Report *Report
	Error  error
}

// Evaluate creates a streaming evaluation job and returns a channel. You can read from the channel
// to receive the reports as they become available. If an error occurs, the channel will be closed
// and the error will be sent on the channel.
func (c *Client) Evaluate(ctx context.Context, req *EvaluateRequest) (chan EvaluateResult, error) {
	protoReq, err := req.toProto()
	if err != nil {
		return nil, fmt.Errorf("failed to prepare the request: %w", err)
	}

	stream, err := c.gateway.Evaluate(ctx, protoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create the stream: %w", err)
	}

	ch := make(chan EvaluateResult)
	go func() {
		defer close(ch)
		for {
			protoResp, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				ch <- EvaluateResult{Error: err}
			}

			report := new(Report).fromProto(protoResp.GetPolicyEvaluationReport())
			ch <- EvaluateResult{Report: report}
		}
	}()
	return ch, nil
}

// EvaluateOne is sugar for times when you only need to evaluate a single content item and get its report.
func (c *Client) EvaluateOne(
	ctx context.Context,
	policyId string,
	content Contenter,
	options JobOptions,
) (*Report, error) {
	seq := c.EvaluateIter(ctx, &EvaluateRequest{
		Content:  []Contenter{content},
		PolicyID: policyId,
		Options:  options,
	})

	nxt, stop := iter.Pull2(seq)
	defer stop()

	resp, err, valid := nxt()
	if !valid {
		return nil, errors.New("unable to retrieve result")
	}

	return resp, err
}
