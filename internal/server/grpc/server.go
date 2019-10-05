package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/lucperkins/strato/internal/services/kv"

	"github.com/lucperkins/strato"
	"github.com/lucperkins/strato/internal/backend"

	"github.com/lucperkins/strato/proto"

	"github.com/sirupsen/logrus"

	"google.golang.org/grpc"
)

type Server struct {
	address string
	srv     *grpc.Server
	backend backend.Service
	log     *logrus.Entry
}

var (
	_ proto.CacheServer   = (*Server)(nil)
	_ proto.CounterServer = (*Server)(nil)
	_ proto.FlagServer    = (*Server)(nil)
	_ proto.KVServer      = (*Server)(nil)
	_ proto.SetServer     = (*Server)(nil)
)

func NewGrpcServer(cfg *strato.ServerConfig) (*Server, error) {
	addr := fmt.Sprintf(":%d", cfg.Port)

	srv := grpc.NewServer()

	bk, err := backend.NewBackend(cfg)
	if err != nil {
		return nil, err
	}

	logger := logrus.New()

	if cfg.Debug {
		logger.SetLevel(logrus.DebugLevel)
	}

	log := logger.WithField("server", "grpc")

	return &Server{
		address: addr,
		srv:     srv,
		backend: bk,
		log:     log,
	}, nil
}

func (s *Server) CacheGet(_ context.Context, req *proto.CacheGetRequest) (*proto.CacheGetResponse, error) {
	val, err := s.backend.CacheGet(req.Key)
	if err != nil {
		if strato.IsNotFound(err) {
			err = strato.NotFound(req.Key).AsProtoStatus()
		}
	}

	res := &proto.CacheGetResponse{
		Value: val,
	}

	return res, nil
}

func (s *Server) CacheSet(_ context.Context, req *proto.CacheSetRequest) (*proto.Empty, error) {
	key, val, ttl := req.Key, req.Item.Value, req.Item.Ttl

	if err := s.backend.CacheSet(key, val, ttl); err != nil {
		return nil, err
	}

	return &proto.Empty{}, nil
}

func (s *Server) IncrementCounter(_ context.Context, req *proto.IncrementCounterRequest) (*proto.Empty, error) {
	if err := s.backend.CounterIncrement(req.Key, req.Amount); err != nil {
		return nil, err
	}

	return &proto.Empty{}, nil
}

func (s *Server) GetCounter(_ context.Context, req *proto.GetCounterRequest) (*proto.GetCounterResponse, error) {
	val, err := s.backend.CounterGet(req.Key)
	if err != nil {
		return nil, err
	}

	return &proto.GetCounterResponse{
		Value: val,
	}, nil
}

func (s *Server) FlagGet(_ context.Context, req *proto.FlagGetRequest) (*proto.FlagResponse, error) {
	val, err := s.backend.FlagGet(req.Key)
	if err != nil {
		return nil, err
	}

	return &proto.FlagResponse{
		Value: val,
	}, nil
}

func (s *Server) FlagSet(_ context.Context, req *proto.FlagSetRequest) (*proto.Empty, error) {
	if err := s.backend.FlagSet(req.Key, req.Value); err != nil {
		return nil, err
	}

	return &proto.Empty{}, nil
}

func (s *Server) KVGet(_ context.Context, location *proto.Location) (*proto.GetResponse, error) {
	key := location.Key

	val, err := s.backend.KVGet(key)
	if err != nil {
		if strato.IsNotFound(err) {
			err = strato.NotFound(key).AsProtoStatus()
		}

		return nil, err
	}

	res := &proto.GetResponse{
		Value: &proto.Value{
			Content: val.Content,
		},
	}

	return res, nil
}

func (s *Server) KVPut(_ context.Context, req *proto.PutRequest) (*proto.Empty, error) {
	key := req.Location.Key

	val := &kv.Value{
		Content: req.Value.Content,
	}

	if err := s.backend.KVPut(key, val); err != nil {
		return nil, err
	}

	return &proto.Empty{}, nil
}

func (s *Server) KVDelete(_ context.Context, location *proto.Location) (*proto.Empty, error) {
	key := location.Key

	if err := s.backend.KVDelete(key); err != nil {
		return nil, err
	}

	return &proto.Empty{}, nil
}

// Sets
func (s *Server) SetGet(_ context.Context, req *proto.GetSetRequest) (*proto.SetResponse, error) {
	items, err := s.backend.SetGet(req.Set)
	if err != nil {
		if strato.IsNotFound(err) {
			return emptySetRes, nil
		} else {
			return nil, err
		}
	}

	return &proto.SetResponse{
		Items: items,
	}, nil
}

func (s *Server) SetAdd(_ context.Context, req *proto.ModifySetRequest) (*proto.SetResponse, error) {
	items, err := s.backend.SetAdd(req.Set, req.Item)
	if err != nil {
		if strato.IsNotFound(err) {
			return emptySetRes, nil
		} else {
			return nil, err
		}
	}

	return &proto.SetResponse{
		Items: items,
	}, nil
}

func (s *Server) SetRemove(_ context.Context, req *proto.ModifySetRequest) (*proto.SetResponse, error) {
	items, err := s.backend.SetRemove(req.Set, req.Item)
	if err != nil {
		if strato.IsNotFound(err) {
			return emptySetRes, nil
		} else {
			return nil, err
		}
	}

	return &proto.SetResponse{
		Items: items,
	}, nil
}

var emptySetRes = &proto.SetResponse{
	Items: []string{},
}

func (s *Server) Start() error {
	proto.RegisterCacheServer(s.srv, s)

	s.log.Debug("registered gRPC cache service")

	proto.RegisterCounterServer(s.srv, s)

	s.log.Debug("registered gRPC counter service")

	proto.RegisterKVServer(s.srv, s)

	s.log.Debug("registered gRPC KV service")

	proto.RegisterSetServer(s.srv, s)

	s.log.Debug("registered gRPC set service")

	lis, err := net.Listen("tcp", s.address)

	if err != nil {
		return err
	}

	s.log.Infof("starting the Strato gRPC server on %s", s.address)

	return s.srv.Serve(lis)
}

func (s *Server) ShutDown() error {
	s.log.Debug("shutting down")

	if err := s.backend.Close(); err != nil {
		return err
	}

	s.srv.GracefulStop()

	return nil
}
