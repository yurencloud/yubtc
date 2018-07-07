package main

import (
	"github.com/gorilla/mux"
	"github.com/yurencloud/yugo/config"
	"net/http"
	_ "github.com/yurencloud/yugo/log"
	log "github.com/sirupsen/logrus"
)

func Run() {

	router := mux.NewRouter()

	InitRouter(router)

	configMap := config.GetConfigMap()

	appName := configMap["app.name"]

	log.Info("app: " + appName + ", started at port " + configMap["rpc.port"])

	http.ListenAndServe(":"+configMap["rpc.port"], router)
}
