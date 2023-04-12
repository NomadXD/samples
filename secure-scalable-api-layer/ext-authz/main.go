package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	auth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/golang-jwt/jwt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
)

type AuthServer struct{}
 
var secretKey []byte = []byte("my_secret_key")

func (authServer *AuthServer) Check(ctx context.Context, request *auth.CheckRequest) (*auth.CheckResponse, error) {
	tracer := otel.Tracer("ext-authz")
    _, span := tracer.Start(ctx, "ext-authz-span")
    defer span.End()
	log.Printf("Auth server received auth request: %v", request.String())
	authHeader, ok := request.Attributes.Request.Http.Headers["authorization"]
	var tokenString string

	if ok {
		tokenString = strings.TrimPrefix(authHeader, "Bearer ")
	}
	span.SetAttributes(
        attribute.String("example-key", "example-value"),
		attribute.String("guid:x-request-id", request.Attributes.Request.Http.Headers["x-request-id"]),
    )
	fmt.Printf("Token: %s", tokenString)
	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Check the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Return the secret key used to sign the token
		return secretKey, nil
	})
	fmt.Printf("Error: %v", err)
	span.End()
	if err == nil {
		return &auth.CheckResponse{
			Status: &status.Status{
				Code: int32(code.Code_OK),
				Message: "AuthServer authentication successful",
			},
		}, nil
	}
	return &auth.CheckResponse{
		Status: &status.Status{
			Code: int32(code.Code_PERMISSION_DENIED),
			Message: "AuthServer authentication unsuccessful",
		},
	}, nil 
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	opts := []grpc.ServerOption{grpc.MaxConcurrentStreams(10)}
	s := grpc.NewServer(opts...)

	auth.RegisterAuthorizationServer(s, &AuthServer{})

	handler := func(w http.ResponseWriter, r *http.Request) {
		claims := jwt.MapClaims{
			"sub": "1234567890",
			"name": "NomadXD",
			"iat": time.Now().Unix(),
			"exp": time.Now().Add(time.Hour * 24).Unix(),
		}
	
		// Define the signing method
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
		// Sign the token with a secret key
		
		tokenString, _ := token.SignedString(secretKey)
		fmt.Fprint(w, tokenString)
	}

	http.HandleFunc("/", handler)

	log.Println("Starting gRPC Server at 50051")
	go s.Serve(lis)
	initTracer()
	http.ListenAndServe(":8080", nil)
}

func initTracer() {
    exp, err := otlptracegrpc.New(
		context.Background(),
        otlptracegrpc.WithInsecure(),
        otlptracegrpc.WithEndpoint("jaeger:4317"),
    )
	//defer exp.Stop()
    if err != nil {
        fmt.Printf("Error: %v", err)
    }

    tp := trace.NewTracerProvider(
        trace.WithSampler(trace.AlwaysSample()),
        trace.WithSyncer(exp),
    )
	otel.SetTracerProvider(tp)
}
