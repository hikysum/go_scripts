package utils

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/imroc/req/v3"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"hash"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	UA = "jdapp;android;11.4.4;;;appBuild/98651;ef/1;ep/%7B%22hdid%22%3A%22JM9F1ywUPwflvMIpYPok0tt5k9kW4ArJEU3lfLhxBqw%3D%22%2C%22ts%22%3A1675759978840%2C%22ridx%22%3A-1%2C%22cipher%22%3A%7B%22od%22%3A%22%22%2C%22ad%22%3A%22DwDvYWZvCQVtZNvsYzG4Dm%3D%3D%22%2C%22ud%22%3A%22DwDvYWZvCQVtZNvsYzG4Dm%3D%3D%22%2C%22ov%22%3A%22CtC%3D%22%2C%22sv%22%3A%22Ds4mBtO%3D%22%7D%2C%22ciphertype%22%3A5%2C%22version%22%3A%221.2.0%22%2C%22appname%22%3A%22com.jingdong.app.mall%22%7D;jdSupportDarkMode/0;Mozilla/5.0 (Linux; Android 6.0.1; vivo Y66L Build/MMB29M; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/89.0.4389.72 MQQBrowser/6.2 TBS/046140 Mobile Safari/537.36"
)

var (
	cryptoMap = map[string]any{
		"MD5":    md5.New,
		"SHA256": sha256.New,
		"SHA512": sha512.New,
		"HmacSHA512": func(token string) hash.Hash {
			return hmac.New(sha512.New, []byte(token))
		},
		"HmacSHA256": func(token string) hash.Hash {
			return hmac.New(sha256.New, []byte(token))
		},
		"HmacMD5": func(token string) hash.Hash {
			return hmac.New(md5.New, []byte(token))
		},
	}
)

type H5stBody struct {
	AppId         string `json:"appid"`
	Body          any    `json:"body"`
	Client        string `json:"client"`
	ClientVersion string `json:"clientVersion"`
	FunctionId    string `json:"functionId"`
}

func H5st(body H5stBody, appId string, fp string) string {
	client := req.NewClient().SetProxy(http.ProxyFromEnvironment)
	now := time.Now()
	response, err := client.R().SetHeaders(map[string]string{
		`Host`:         `cactus.jd.com`,
		`accept`:       `application/json`,
		`content-type`: `application/json`,
		`user-agent`:   UA,
	}).SetBodyJsonMarshal(map[string]string{
		"version":      "3.0",
		"appId":        appId,
		"fp":           fp,
		"timestamp":    strconv.FormatInt(now.UnixNano(), 10),
		"platform":     "web",
		"expandParams": "",
	}).Post("https://cactus.jd.com/request_algo?g_ty=ajax")
	if err != nil {
		log.Errorln("request_algo失败，" + err.Error())
		return ""
	}
	result := gjson.GetBytes(response.Bytes(), "data.result").String()
	tk := gjson.Get(result, "tk").String()
	algo := gjson.Get(result, "algo").String()
	rd, method := parseALgo(algo)
	str := fmt.Sprintf("%s%s%d%s%s", tk, fp, now.UnixNano(), appId, rd)
	var y []byte
	switch method.(type) {
	case func() hash.Hash:
		h := method.(func() hash.Hash)()
		h.Write([]byte(str))
		y = h.Sum(nil)
	case func(token string) hash.Hash:
		h := method.(func(token string) hash.Hash)(tk)
		h.Write([]byte(str))
		y = h.Sum(nil)
	}
	bodyData, _ := json.Marshal(body.Body)
	sha256M := sha256.New()
	sha256M.Write(bodyData)
	newBody := hex.EncodeToString(sha256M.Sum(nil))
	h1 := hmac.New(sha256.New, y)
	h1.Write([]byte(fmt.Sprintf("appid:%v&body:%v&client:%v&clientVersion:%v&functionId:%v", body.AppId, newBody, body.Client, body.ClientVersion, body.FunctionId)))
	s5 := hex.EncodeToString(h1.Sum(nil))
	result = fmt.Sprintf("%v;%v;%v;%v;%v;%v;%v", strings.ReplaceAll(now.Format("20060102150405.000"), ".", ""), fp, appId, tk, s5, "3.0", now.UnixMilli())
	log.Debugln(result)
	return result
}

func parseALgo(algo string) (rd string, cryptoMethod any) {
	rdRegex := regexp.MustCompile(`rd='(.*?)'`)
	cryptoMethodNameRegex := regexp.MustCompile(`algo\.(.*?)\(`)
	rdMatch := rdRegex.FindStringSubmatch(algo)
	cryptoMethodNameMatch := cryptoMethodNameRegex.FindStringSubmatch(algo)
	log.Infoln(rdMatch)
	log.Infoln(cryptoMethodNameMatch)

	return rdMatch[1], cryptoMap[cryptoMethodNameMatch[1]]
}
