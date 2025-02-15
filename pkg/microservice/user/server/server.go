/*
Copyright 2021 The KodeRover Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"context"
	"net"
	"net/http"
	"time"

	ext_authz_v3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/grpc"

	"github.com/koderover/zadig/pkg/microservice/user/core"
	userGrpcServer "github.com/koderover/zadig/pkg/microservice/user/server/grpc"
	"github.com/koderover/zadig/pkg/microservice/user/server/rest"
	"github.com/koderover/zadig/pkg/tool/log"
)

func Serve(ctx context.Context) error {
	core.Start(ctx)
	defer core.Stop(ctx)

	log.Info("Start user system service")

	engine := rest.NewEngine()
	server := &http.Server{Addr: ":80", Handler: engine}

	grpcServer := grpc.NewServer()

	stopChan := make(chan struct{})
	go func() {
		defer close(stopChan)

		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Errorf("Failed to stop server, error: %s", err)
		}

		grpcServer.GracefulStop()
	}()

	listen, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to start grpc server listener, error: %s", err)
	}

	ext_authz_v3.RegisterAuthorizationServer(grpcServer, &userGrpcServer.AuthServer{})

	go func() {
		log.Infof("grpc server stating on port: %d.....", 8080)
		if err := grpcServer.Serve(listen); err != nil {
			log.Fatalf("failed to start grpc server, error: %s", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Errorf("Failed to start http server, error: %s", err)
		return err
	}

	<-stopChan

	return nil
}
