package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	authTokenKey   = "auth_token"
	authTokenValue = "authd"
)

func validateAuthToken(ctx context.Context) error {
	md, _ := metadata.FromIncomingContext(ctx)
	if t, ok := md[authTokenKey]; ok {
		switch {
		case len(t) != 1:
			return status.Errorf(codes.InvalidArgument, fmt.Sprintf("%s should contain only 1 value", authTokenKey))
		case t[0] != authTokenValue: // Simulate checking if auth token is valid
			return status.Errorf(codes.Unauthenticated, fmt.Sprintf("incorrect %s", authTokenKey))
		}
	} else {
		return status.Errorf(codes.Unauthenticated, fmt.Sprintf("failed to get %s", authTokenKey))
	}

	return nil
}

func unaryAuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if err := validateAuthToken(ctx); err != nil {
		return nil, err
	}

	return handler(ctx, req)
}

func streamAuthInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := validateAuthToken(ss.Context()); err != nil {
		return err
	}

	return handler(srv, ss)
}