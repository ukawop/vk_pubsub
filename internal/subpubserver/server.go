package subpubserver

import (
	"context"
	"subpub/pkg/subpub"
	"sync"

	"subpub/proto/subpub/gen/go/subpubv1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type serverAPI struct {
	ctx context.Context
	subpubv1.UnimplementedPubSubServer
	pubsub  subpub.SubPub
	mu      sync.Mutex
	streams map[string][]*streamContext
}

type streamContext struct {
	stream subpubv1.PubSub_SubscribeServer
	cancel context.CancelFunc
}

func Register(ctx context.Context, gRPC *grpc.Server, pubsub subpub.SubPub) {
	api := &serverAPI{
		pubsub:  pubsub,
		streams: make(map[string][]*streamContext),
		ctx:     ctx,
	}
	subpubv1.RegisterPubSubServer(gRPC, api)
}

func (s *serverAPI) Subscribe(req *subpubv1.SubscribeRequest, stream subpubv1.PubSub_SubscribeServer) error {
	if req.GetKey() == "" {
		return status.Error(codes.InvalidArgument, "key is required")
	}
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	s.mu.Lock()
	StreamCtx := streamContext{
		stream: stream,
		cancel: cancel,
	}
	s.streams[req.GetKey()] = append(s.streams[req.GetKey()], &StreamCtx)
	s.mu.Unlock()

	sub, err := s.pubsub.Subscribe(req.GetKey(), func(msg interface{}) {
		if event, ok := msg.(*subpubv1.Event); ok {
			if err := stream.Send(event); err != nil {
				cancel()
			}
		}
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe: %v", err)
	}

	select {
	case <-ctx.Done():
	case <-s.ctx.Done():
		cancel()
	}
	defer sub.Unsubscribe()
	defer func() {
		s.mu.Lock()

		defer s.mu.Unlock()
		streams := s.streams[req.GetKey()]
		for index, strm := range streams {
			if strm == &StreamCtx {
				s.streams[req.GetKey()] = append(streams[:index], streams[index+1:]...)
				break
			}
		}
	}()

	return nil
}

func (s *serverAPI) Publish(ctx context.Context, req *subpubv1.PublishRequest) (*emptypb.Empty, error) {
	if req.GetKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "key is required")
	}
	if req.GetData() == "" {
		return nil, status.Error(codes.InvalidArgument, "data is required")
	}

	event := &subpubv1.Event{Data: req.GetData()}
	if err := s.pubsub.Publish(req.GetKey(), event); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to publish: %v", err)
	}

	return &emptypb.Empty{}, nil
}
