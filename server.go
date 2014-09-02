package main

import (
	"strconv"
	"strings"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
)

func GetServers(username AuthUser, r render.Render) {
	var servers []Server
	db.Find(&servers)

	data := map[string]interface{}{"username": username, "servers": servers}
	r.HTML(200, "servers", data)
}
func DeleteServer(params martini.Params, r render.Render) {
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		r.JSON(200, ActionMessage{
			Success: false,
			Message: "id参数错误",
		})
		return
	}

	err = db.Delete(&Server{Id: id}).Error
	if err != nil {
		r.JSON(200, ActionMessage{
			Success: false,
			Message: "删除出错." + err.Error(),
		})
		return
	}
	r.JSON(200, ActionMessage{
		Success: true,
		Message: "成功",
	})
}
func EditServer(params martini.Params, server Server, r render.Render) {
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		r.JSON(200, ActionMessage{
			Success: false,
			Message: "id参数错误",
		})
		return
	}

	server.Tags = strings.TrimLeft(server.Tags, ",")
	server.Tags = strings.TrimRight(server.Tags, ",")
	var temp Server
	db.First(&temp, id)
	temp.Tags = server.Tags
	err = db.Save(&temp).Error
	if err != nil {
		r.JSON(200, ActionMessage{
			Success: false,
			Message: "更新出错." + err.Error(),
		})
		return
	}
	r.JSON(200, ActionMessage{
		Success: true,
		Message: "成功",
		Data:    temp,
	})
}
func ToggleServer(params martini.Params, r render.Render) {
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		r.JSON(200, ActionMessage{
			Success: false,
			Message: "id参数错误",
		})
		return
	}

	var temp Server
	db.First(&temp, id)
	temp.Disable = !temp.Disable
	err = db.Save(&temp).Error
	if err != nil {
		r.JSON(200, ActionMessage{
			Success: false,
			Message: "保存出错." + err.Error(),
		})
		return
	}
	r.JSON(200, ActionMessage{
		Success: true,
		Message: "成功",
		Data:    temp,
	})
}
