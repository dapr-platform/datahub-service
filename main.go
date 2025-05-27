package main

import (
	"datahub-service/api"
	_ "datahub-service/docs"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "datahub-service/service"

	daprd "github.com/dapr/go-sdk/service/http"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

var (
	PORT         = 80
	BASE_CONTEXT = ""
)

func init() {
	if val := os.Getenv("LISTEN_PORT"); val != "" {
		PORT, _ = strconv.Atoi(val)
	}

	if val := os.Getenv("BASE_CONTEXT"); val != "" {
		BASE_CONTEXT = val
	}
}

// @title 数据底座服务 API
// @version 1.0
// @description 智慧园区数据底座后台服务，提供数据采集、处理、存储、治理和共享功能
// @BasePath /swagger/datahub-service
func main() {
	mux := chi.NewRouter()

	// 如果有BASE_CONTEXT，则在该路径下挂载所有路由
	if BASE_CONTEXT != "" {
		mux.Route(BASE_CONTEXT, func(r chi.Router) {
			// 创建子路由器并初始化路由
			subMux := r.(*chi.Mux)
			api.InitRoute(subMux)
			r.Handle("/metrics", promhttp.Handler())
			r.Handle("/swagger*", httpSwagger.WrapHandler)
		})
	} else {
		api.InitRoute(mux)
		mux.Handle("/metrics", promhttp.Handler())
		mux.Handle("/swagger*", httpSwagger.WrapHandler)
	}

	s := daprd.NewServiceWithMux(":"+strconv.Itoa(PORT), mux)
	if err := s.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("error: %v", err)
	}
}
