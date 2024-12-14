package internal

import (
	"context"
	"fmt"
	"net/url"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/clavataai/monorail/libs/containers/slices"
	pb "github.com/clavataai/monorail/libs/protobufs/gateway/v1"
	sharedpb "github.com/clavataai/monorail/libs/protobufs/shared/v1"
)

type client struct {
	// endpointBase is the base URL for the API endpoint. It generally should include just the host and port
	endpointBase *url.URL

	// apiKey is the API key used to authenticate requests
	apiKey string

	// conn is the gRPC client connection
	conn *grpc.ClientConn

	// stub is the gateway service client
	stub pb.GatewayServiceClient
}

func NewClient(host string, port int, apiKey string) (Client, error) {
	endpointBase := &url.URL{
		Scheme: "grpc",
		Host:   fmt.Sprintf("%s:%d", host, port),
	}

	c, err := grpc.NewClient(endpointBase.String(), grpc.WithTransportCredentials(
		insecure.NewCredentials(),
	))
	if err != nil {
		return nil, err
	}

	gsc := pb.NewGatewayServiceClient(c)

	return &client{
		endpointBase: endpointBase,
		apiKey:       apiKey,
		conn:         c,
		stub:         gsc,
	}, nil
}

func (c *client) Close() error {
	return c.conn.Close()
}

type contextKey string

const (
	contextKeyAuthorization contextKey = "Authorization"
)

func (c *client) Evaluate(ctx context.Context, inputs *EvaluateInput) (*EvaluateOutput, error) {
	// Add the API key to the context
	ctx = context.WithValue(ctx, contextKeyAuthorization, fmt.Sprintf("Bearer %s", c.apiKey))

	// The content and policy names are a N:M relationship. So for each content and each policy, we need to evaluate
	// the content against the policy. We'll generate one input to the gRPC call for each combination of content and
	// policy.
	contentInputs, err := slices.MapWithError(inputs.Content, func(c Content) (*sharedpb.ContentData, error) {
		// For each piece of content, we need to use the mode to figure out which gRPC oneOf field to populate
		var contentData *sharedpb.ContentData
		switch c.Mode {
		case ContentModeText:
			contentData = &sharedpb.ContentData{
				Content: &sharedpb.ContentData_Text{
					Text: string(c.Payload),
				},
			}
		// TODO(@boichee): Base64 is going to need its own field in the protobuf. For now this is here as a reminder and for completeness.
		case ContentModeImage, ContentModeImageBase64:
			contentData = &sharedpb.ContentData{
				Content: &sharedpb.ContentData_Image{
					Image: c.Payload,
				},
			}
		case ContentModeImageUrl:
			contentData = &sharedpb.ContentData{
				Content: &sharedpb.ContentData_ImageUrl{
					ImageUrl: string(c.Payload),
				},
			}
		case ContentModeUnspecified:
			// Return an error if the mode is unspecified
			return nil, ErrUnspecifiedContentMode
		default:
			// This should never happen, but if it does, we should panic
			panic(fmt.Sprintf("unknown content mode: %d", c.Mode))
		}

		return contentData, nil
	})
	if err != nil {
		return nil, err
	}

	req := &pb.EvaluateRequest{
		ContentData: contentInputs,
		PolicyId:    inputs.PolicyId,
	}

	stream, err := c.stub.Evaluate(ctx, req)
	if err != nil {
		return nil, err
	}

	outputs := make([]*EvaluateOutput, 0)
	_ = outputs
	for {
		out, err := stream.Recv()
		if err != nil {
			return nil, err
		}

		// Do something with the output
		// TODO(@boichee): Stopping here. Looking at the protobufs I'm realizing we may have an issue with how they are
		// designed. We may need to modify the protobufs to be able to get this right.
		_ = out
	}
}
