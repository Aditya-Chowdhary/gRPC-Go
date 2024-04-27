package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	authTokenKey   = "auth_token"
	authTokenValue = "authd"
)

func validateAuthToken(ctx context.Context) (context.Context, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if t, ok := md[authTokenKey]; ok {
		switch {
		case len(t) != 1:
			return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("%s should contain only 1 value", authTokenKey))
		case t[0] != authTokenValue: // Simulate checking if auth token is valid
			return nil, status.Errorf(codes.Unauthenticated, fmt.Sprintf("incorrect %s", authTokenKey))
		}
	} else {
		return nil, status.Errorf(codes.Unauthenticated, fmt.Sprintf("failed to get %s", authTokenKey))
	}

	return ctx, nil
}

const grpcService = 5
const grpcMethod = 7

func logCalls(l *log.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, level logging.Level, msg string, fields ...any) {
		// f := make(map[string]any, len(fields)/2)
		// i := logging.Fields(fields).Iterator()
		// for i.Next() {
		// 	k, v := i.At()
		// 	f[k] = v
		// }
		switch level {
		case logging.LevelDebug:
			msg = fmt.Sprintf("DEBUG :%v", msg)
		case logging.LevelInfo:
			msg = fmt.Sprintf("INFO :%v", msg)
		case logging.LevelWarn:
			msg = fmt.Sprintf("WARN :%v", msg)
		case logging.LevelError:
			msg = fmt.Sprintf("ERROR :%v", msg)
		default:
			panic(fmt.Sprintf("unknown level %v", level))
		}
		l.Println(msg, fields[grpcService], fields[grpcMethod])
	})
}
