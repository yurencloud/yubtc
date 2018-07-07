package main

import (
	"github.com/gorilla/mux"
	"github.com/yurencloud/yubtc/controller"
)

func InitRouter(router *mux.Router) {
	router.HandleFunc("/", controller.Index)
	router.HandleFunc("/login", controller.Login)
}
