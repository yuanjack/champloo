package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
)

func HttpGet(url string) (string, int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return err.Error(), -1, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		fmt.Printf("请求%s出错，错误消息：%v", url, err.Error())
		return err.Error(), -1, err
	}

	return string(body), resp.StatusCode, nil
}

func PutJson(url string, data interface{}) (string, int, error) {
	bytesData, _ := json.Marshal(data)
	buf := bytes.NewReader(bytesData)
	client := &http.Client{}
	req, err := http.NewRequest("PUT", url, buf)
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err.Error(), -1, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		fmt.Printf("请求%s出错，提交内容：%v，错误消息：%v", url, bytesData, err.Error())
		return err.Error(), -1, err
	}

	return string(body), resp.StatusCode, nil
}

func PostJson(url string, data interface{}) (string, int, error) {
	bytesData, _ := json.Marshal(data)
	buf := bytes.NewReader(bytesData)
	fmt.Println(string(bytesData))
	resp, err := http.Post(url, "application/json", buf)
	if err != nil {
		return err.Error(), -1, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		fmt.Printf("请求%s出错，提交内容：%v，错误消息：%v", url, bytesData, err.Error())
		return err.Error(), -1, err
	}

	return string(body), resp.StatusCode, nil
}

func PostJsonWithFile(url string, data interface{}, fileParams map[string][]byte) (string, int, error) {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	//关键的一步操作，创建一个multipart，用于写入文件数据
	for filename, fileBytes := range fileParams {
		fileWriter, err := bodyWriter.CreateFormFile(filename, filename)
		if err != nil {
			return err.Error(), -1, err
		}

		//写入文件数据到multipart
		fileBuf := bytes.NewReader(fileBytes)
		_, err = io.Copy(fileWriter, fileBuf)
		if err != nil {
			return err.Error(), -1, err
		}
	}

	//写入json数据
	fileWriter, err := bodyWriter.CreateFormField("json")
	if err != nil {
		return err.Error(), -1, err
	}
	bytesData, _ := json.Marshal(data)
	jsonBuf := bytes.NewReader(bytesData)
	_, err = io.Copy(fileWriter, jsonBuf)
	if err != nil {
		return err.Error(), -1, err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	resp, err := http.Post(url, contentType, bodyBuf)
	if err != nil {
		return err.Error(), -1, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		fmt.Printf("请求%s出错，提交内容：%v，错误消息：%v", url, bytesData, err.Error())
		return err.Error(), -1, err
	}
	return string(body), resp.StatusCode, nil

}

// 下载文件
func DownloadFileToBytes(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		fmt.Errorf(err.Error())
		return []byte{}, err
	}
	bytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return []byte{}, errors.New(fmt.Sprintf("请求出错.响应码：%d 响应内容：%s", res.StatusCode, string(bytes)))
	}
	if err != nil {
		fmt.Errorf(err.Error())
		return []byte{}, err
	}

	return bytes, err
}
