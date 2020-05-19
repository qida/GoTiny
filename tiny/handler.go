package tiny

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/qida/go/logs"
	"github.com/qida/go/nets"
)

const TinyApiHost = "https://api.tinify.com"

type TinyHandler struct {
	ApiKey      string //api key 必须值
	httpclient  *http.Client
	authortoken string
	outImgDir   string
}

func (tinyhandler *TinyHandler) SetData(apikey string, outImgDir string) {
	tinyhandler.ApiKey = apikey
	tinyhandler.httpclient = http.DefaultClient
	tinyhandler.outImgDir = outImgDir
	tinyhandler.getAuthorCode(tinyhandler.ApiKey)
}

func (tinyhandler *TinyHandler) getAuthorCode(apikey string) string {
	strCode := "api:" + apikey
	tinyhandler.authortoken = "Basic " + base64.StdEncoding.EncodeToString([]byte(strCode))
	return tinyhandler.authortoken
}

/**
上传图片
*/
func (tinyhandler *TinyHandler) UploadFile(imgfilepath string) (error, string) {
	apiurl := TinyApiHost + "/shrink"
	imgBytes, err := ioutil.ReadFile(imgfilepath)
	if err != nil {
		return err, ""
	}
	req, err := http.NewRequest("POST", apiurl, bytes.NewReader(imgBytes))
	if err != nil {
		return err, ""
	}
	req.Header.Set("Content-Type", "multipart/form-data")
	req.Header.Set("Authorization", tinyhandler.authortoken)
	res, err := tinyhandler.httpclient.Do(req)
	if err != nil {
		return err, ""
	}
	imgUrl := res.Header.Get("Location")
	// log.Println("img url", imgUrl)
	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return err, imgUrl
	}
	log.Printf("处理中 图片：[%s]\r\n", imgfilepath)
	// logs.Send2Dingf(logs.Rb监控, "图片压缩：%s", string(resBytes))
	return nil, imgUrl
}

/**
下载图片
*/
func (tinyhandler *TinyHandler) DownloadImg(imgUrl string, outFilePath string) error {
	req, err := http.NewRequest("GET", imgUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", tinyhandler.authortoken)
	response, err := tinyhandler.httpclient.Do(req)
	if err != nil {
		return err
	}
	resByres, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(outFilePath, resByres, 0666)
	return err
}

/**
压缩单张图片
*/
func (tinyHandler *TinyHandler) CompressImageFile(imgFile string, outFile string) {
	if outFile == "" {
		fileName := filepath.Base(imgFile)
		outFile = filepath.Join(tinyHandler.outImgDir, fileName)
	}
	err, imgUrl := tinyHandler.UploadFile(imgFile)
	if err != nil {
		logs.Send2Dingf(logs.Rb错误, "图片压缩 上传图片错误：[%s] %s", imgFile, err.Error())
	}
	if imgUrl == "" {
		logs.Send2Dingf(logs.Rb错误, "图片压缩 图片错误_图片url为空 [%s]", imgFile)
		return
	}
	err = tinyHandler.DownloadImg(imgUrl, outFile)
	if err != nil {
		logs.Send2Dingf(logs.Rb错误, "图片压缩 下载图片失败：[%s] %s", imgFile, err.Error())
		return
	}
	log.Printf("压缩完 图片：[%s]\r\n", imgFile)
	log.Println(strings.Repeat("*", NumRepeat))
	logs.Send2Dingf(logs.Rb监控, "图片压缩 下载成功：[%s]", imgFile)
}

var NumRepeat = 30

/**
压缩文件夹内的所有图片
*/
func (tinyHandler *TinyHandler) CompressAllImages(imgsDir string, outDir string) error {
	fileinfoList, err := ioutil.ReadDir(imgsDir)
	if err != nil {
		return err
	}
	log.Printf("只支持 jpg png 格式\r\n\r\n")
	log.Printf("如长时间没有反应可按回车键激活任务\r\n\r\n")
	log.Println("qida v1.0")
	log.Println(strings.Repeat("=", NumRepeat))
	var num int
	for _, itemFile := range fileinfoList {
		if itemFile.IsDir() || itemFile.Size() <= 0 {
			continue
		}
		if !strings.Contains(itemFile.Name(), "png") && !strings.Contains(itemFile.Name(), "jpg") {
			continue
		}
		num++
		log.Printf("准备中 图片：[%s]\r\n", itemFile.Name())
		tinyHandler.CompressImageFile(filepath.Join(imgsDir, itemFile.Name()), outDir)
	}
	log.Printf("已完成 图片：%d 个\r\n", num)
	log.Println(strings.Repeat("=", NumRepeat))
	ip, err := nets.ExternalIP()
	if err == nil {
		logs.Send2Dingf(logs.Rb监控, "图片压缩 Num: %d IP:%s", num, ip.String())
	}
	return nil
}
