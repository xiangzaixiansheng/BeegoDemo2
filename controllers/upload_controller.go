package controllers

import (
	"BeegoDemo2/util"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/astaxie/beego"
)

type UploadController struct {
	beego.Controller
}

type Sizer interface {
	Size() int64
}

const (
	LOCAL_FILE_DIR    = "static/upload/"
	MIN_FILE_SIZE     = 1       // bytes
	MAX_FILE_SIZE     = 5000000 // bytes
	IMAGE_TYPES       = "(jpg|gif|p?jpeg|(x-)?png)"
	ACCEPT_FILE_TYPES = IMAGE_TYPES
	EXPIRATION_TIME   = 300 // seconds
	THUMBNAIL_PARAM   = "=s80"
)

var (
	imageTypes      = regexp.MustCompile(IMAGE_TYPES)
	acceptFileTypes = regexp.MustCompile(ACCEPT_FILE_TYPES)
)

type FileInfo struct {
	Url          string `json:"url,omitempty"`
	ThumbnailUrl string `json:"thumbnailUrl,omitempty"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Size         int64  `json:"size"`
	Error        string `json:"error,omitempty"`
	DeleteUrl    string `json:"deleteUrl,omitempty"`
	DeleteType   string `json:"deleteType,omitempty"`
}

// 检测文件类型是否合法
func (fi *FileInfo) ValidateType() (valid bool) {
	if acceptFileTypes.MatchString(fi.Type) {
		return true
	}

	fi.Error = "Filetype not allowed"
	return false
}

// 检查文件大小是否合法
func (fi *FileInfo) ValidateSize() (valid bool) {
	if fi.Size < MIN_FILE_SIZE {
		fi.Error = "File is too small"
	} else if fi.Size > MAX_FILE_SIZE {
		fi.Error = "File is too large"
	} else {
		return true
	}
	return false
}

// 检测是否合法
func (fi *FileInfo) check(err error) {
	if err != nil {
		panic(err)
	}
}

func (fi *FileInfo) escape(s string) string {
	return strings.Replace(url.QueryEscape(s), "+", "%20", -1)
}

func (fi *FileInfo) getFormValue(p *multipart.Part) string {
	var b bytes.Buffer
	io.CopyN(&b, p, int64((1 << 20))) // Copy max: 1 MiB
	return b.String()
}

// 截取字符串
func substr(s string, pos, length int) string {
	runes := []rune(s)
	l := pos + length
	if l > len(runes) {
		l = len(runes)
	}
	return string(runes[pos:l])
}

// 获取父级 目录
func getParentDirectory(dir string) string {
	return substr(dir, 0, strings.LastIndex(dir, "/"))
}

// 获取当前目录
func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		util.LogError(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}

// 获取文件后缀
func (fi *FileInfo) fileExt() {
	ext := path.Ext(fi.Name)
	fi.Type = ext
}

type UploadResult struct {
	Url      string `json:"url"`
	Uploaded bool   `json:"uploaded"`
	Msg      string `json:"msg"`
}

func (this *UploadController) Post() {
	util.LogDebug("上传文件进来了")

	//GetFile函数的参数和SaveToFile函数的第一个参数需要和标签中name属性相同。
	file, head, err := this.GetFile("file")
	if err != nil {
		this.Ctx.WriteString("获取文件失败")
		return
	}

	defer file.Close()

	fi := &FileInfo{
		Name: head.Filename,
	}
	// 获取文件类型
	fi.fileExt()

	if !fi.ValidateType() {
		this.Data["json"] = map[string]interface{}{"code": 0, "message": "文件类型错误"}
		this.ServeJSON()
		return
	}

	if sizeInterface, ok := file.(Sizer); ok {
		fi.Size = sizeInterface.Size()
		if !fi.ValidateSize() {
			this.Data["json"] = map[string]interface{}{"code": 0, "message": fi.Error}
			this.ServeJSON()
			return
		}
	} else {
		this.Data["json"] = map[string]interface{}{"code": 0, "message": "文件大小获取失败"}
		this.ServeJSON()
		return
	}
	now := time.Now()
	ctrlName, _ := this.GetControllerAndAction()
	dirPath := LOCAL_FILE_DIR + strings.ToLower(ctrlName[0:len(ctrlName)-10]) + "/" + now.Format("2006-01") + "/" + now.Format("02")
	fileExt := strings.TrimLeft(fi.Type, ".")
	fileSaveName := fmt.Sprintf("%s_%d.%s", "test", now.Unix(), fileExt)
	filePath := fmt.Sprintf("%s/%s", dirPath, fileSaveName)

	if !util.IsDir(dirPath) {
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			this.Data["json"] = map[string]interface{}{"code": 0, "message": "文件夹“" + dirPath + "”创建失败"}
			this.ServeJSON()
			return
		}
	}
	// 保存位置在 static/upload， 没有文件夹要先创建
	//GetFile函数的参数和SaveToFile函数的第一个参数需要和标签中name属性相同。
	this.SaveToFile("file", filePath)
	this.Data["json"] = map[string]interface{}{"code": 1, "message": "上传成功", "url": "/" + filePath}
	this.ServeJSON()
	return
}
