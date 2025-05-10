package server

import (
	"context"
	"io"
	"subpub_demo/proto"
	"subpub_demo/subpub"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PubSubService struct {
	proto.UnimplementedPubSubServer
	bus    subpub.SubPub
	logger *logrus.Logger
}

func NewPubSubService(bus subpub.SubPub, logger *logrus.Logger) *PubSubService {
	return &PubSubService{
		bus:    bus,
		logger: logger,
	}
}

func (s *PubSubService) Subscribe(req *proto.SubscribeRequest, stream proto.PubSub_SubscribeServer) error {
	ctx := stream.Context()
	key := req.GetKey()

	s.logger.WithField("key", key).Info("New subscription request")

	eventChan := make(chan string)
	handler := func(msg interface{}) {
		if data, ok := msg.(string); ok {
			select {
			case eventChan <- data:
			case <-ctx.Done():
				return
			}
		}
	}

	sub, err := s.bus.Subscribe(key, handler)
	if err != nil {
		s.logger.WithError(err).Error("Failed to subscribe")
		return status.Error(codes.Internal, "failed to subscribe")
	}
	defer sub.Unsubscribe()

	for {
		select {
		case data := <-eventChan:
			err := stream.Send(&proto.Event{Data: data})
			if err != nil {
				if err == io.EOF {
					s.logger.Info("Subscription stream closed by client")
					return nil
				}
				s.logger.WithError(err).Error("Failed to send event")
				return status.Error(codes.Internal, "failed to send event")
			}
		case <-ctx.Done():
			s.logger.Info("Subscription context done")
			return nil
		}
	}
}

func (s *PubSubService) Publish(ctx context.Context, req *proto.PublishRequest) (*emptypb.Empty, error) {
	key := req.GetKey()
	data := req.GetData()

	s.logger.WithFields(logrus.Fields{
		"key":  key,
		"data": data,
	}).Info("Publishing event")

	err := s.bus.Publish(key, data)
	if err != nil {
		s.logger.WithError(err).Error("Failed to publish")
		return nil, status.Error(codes.Internal, "failed to publish")
	}

	return &emptypb.Empty{}, nil
}
