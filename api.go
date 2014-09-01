package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-martini/martini"
)

func Heartbeat(r *http.Request, params martini.Params) string {
	fmt.Println(r.URL.RequestURI())
	host := r.URL.Query().Get("host")
	ip := r.URL.Query().Get("ip")
	port, err := strconv.Atoi(r.URL.Query().Get("port"))
	if err != nil {
		port = 26400
	}

	var server Server

	db.Where(&Server{Ip: ip}).First(&server)
	if server.Id <= 0 {
		server = Server{
			Host:             host,
			Ip:               ip,
			Port:             port,
			Tags:             "全部",
			LastHeatbeatTime: time.Now(),
		}
	}

	server.LastHeatbeatTime = time.Now()
	db.Save(&server)
	return "ok"
}
