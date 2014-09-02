package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
)

func NewSystem(username AuthUser, r render.Render) {
	var servers []Server
	db.Select("tags").Find(&servers)

	tagsMap := map[string]bool{}
	tagsMap["全部"] = true
	for _, server := range servers {
		if server.Tags == "" {
			continue
		}

		arr := strings.Split(server.Tags, ",")
		for _, tag := range arr {
			if strings.TrimSpace(tag) == "" {
				continue
			}

			tagsMap[strings.TrimSpace(tag)] = true
		}
	}

	tags := []string{}

	for key, _ := range tagsMap {
		tags = append(tags, key)
	}

	data := map[string]interface{}{"username": username, "tags": tags, "conf": SystemConfig{Way: "checkout", BackupNum: 10}}
	r.HTML(200, "config", data)
}

func SaveSystem(req *http.Request, params martini.Params, r render.Render) {
	req.ParseForm()
	id, _ := strconv.Atoi(req.PostForm.Get("id"))
	name := req.PostForm.Get("name")
	enbaleDevStage := req.PostForm.Get("dev-stage") == "on"
	enbaleProdStage := req.PostForm.Get("prod-stage") == "on"
	way := req.PostForm.Get("way")
	path := req.PostForm.Get("path")
	shared := req.PostForm.Get("shared")
	num, _ := strconv.Atoi(req.PostForm.Get("num"))
	repo := req.PostForm.Get("repo")
	username := req.PostForm.Get("username")
	password := req.PostForm.Get("password")
	beforecmd := req.PostForm.Get("before-cmd")
	aftercmd := req.PostForm.Get("after-cmd")
	devBeforeCmd := req.PostForm.Get("dev-before-cmd")
	devAfterCmd := req.PostForm.Get("dev-after-cmd")
	prodBeforeCmd := req.PostForm.Get("prod-before-cmd")
	prodAfterCmd := req.PostForm.Get("prod-after-cmd")
	tagsSlice := req.Form["tags"]
	tags := ""
	if len(tagsSlice) > 0 {
		tags = strings.Join(tagsSlice, ",")
	}

	conf := SystemConfig{
		Id:              id,
		Name:            name,
		EnableDevStage:  enbaleDevStage,
		EnableProdStage: enbaleProdStage,
		Way:             way,
		Path:            path,
		Shared:          shared,
		BackupNum:       num,
		Repo:            repo,
		UserName:        username,
		Password:        password,
		Tags:            tags,
		BeforeCmd:       beforecmd,
		AfterCmd:        aftercmd,
		DevBeforeCmd:    devBeforeCmd,
		DevAfterCmd:     devAfterCmd,
		ProdBeforeCmd:   prodBeforeCmd,
		ProdAfterCmd:    prodAfterCmd,
	}
	err := db.Save(&conf).Error
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
		Data:    conf,
	})
}

func GetSystemById(username AuthUser, params martini.Params, r render.Render) {
	id, _ := strconv.Atoi(params["id"])

	var servers []Server
	db.Select("tags").Find(&servers)

	tagsMap := map[string]bool{}
	tagsMap["全部"] = true
	for _, server := range servers {
		if server.Tags == "" {
			continue
		}

		arr := strings.Split(server.Tags, ",")
		for _, tag := range arr {
			if strings.TrimSpace(tag) == "" {
				continue
			}

			tagsMap[strings.TrimSpace(tag)] = true
		}
	}

	tags := []string{}

	for key, _ := range tagsMap {
		tags = append(tags, key)
	}

	var conf SystemConfig
	db.First(&conf, id)
	data := map[string]interface{}{"username": username, "tags": tags, "conf": conf}
	r.HTML(200, "config", data)
}

func ToggleStarSystem(username AuthUser, params martini.Params, r render.Render) {
	id, _ := strconv.Atoi(params["id"])

	var user User
	db.First(&user, User{Name: string(username)})
	if user.Id <= 0 {
		r.JSON(200, ActionMessage{
			Success: false,
			Message: "收藏处理失败,找不到用户信息",
		})
		return
	}

	var star UserStar
	db.First(&star, UserStar{SystemId: id, UserId: user.Id})
	var err error
	if star.Id > 0 {
		err = db.Delete(&star).Error
	} else {
		star.SystemId = id
		star.UserId = user.Id
		err = db.Save(star).Error
	}

	if err != nil {
		r.JSON(200, ActionMessage{
			Success: false,
			Message: "收藏处理出错." + err.Error(),
		})
		return
	}
	r.JSON(200, ActionMessage{
		Success: true,
		Message: "成功",
		Data:    star,
	})

}
