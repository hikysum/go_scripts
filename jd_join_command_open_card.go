package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/imroc/req/v3"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"main/utils"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var cookies []string

var (
	activityId  string
	shopId      string
	activityUrl *url.URL
	inviterUuid string
	signUrl     string

	client *req.Client
)

const (
	UA = "okhttp/3.12.1;jdmall;android;version/11.0.2;build/97565;"
)

func init() {
	utils.InitLog()
	joinCommonId, b := os.LookupEnv("jd_joinCommonId")
	if !b {
		log.Errorln("️⚠️未发现有效活动变量,退出程序!")
		os.Exit(1)
	}
	inviterUuid = os.Getenv("jd_joinCommon_uuid")
	if !strings.Contains(joinCommonId, "&") {
		log.Errorln("️⚠️活动变量错误，退出程序")
		os.Exit(1)
	}
	activityId = strings.Split(joinCommonId, "&")[0]
	shopId = strings.Split(joinCommonId, "&")[1]
	activityUrl, _ = url.Parse(fmt.Sprintf("https://lzdz1-isv.isvjcloud.com/dingzhi/joinCommon/activity/5929859?activityId=%s&shareUuid=%s&adsource=null&shareuserid4minipg=null&lng=00.000000&lat=00.000000&sid=&un_area=&&shopid=%s", activityId, inviterUuid, shopId))
	fmt.Println(fmt.Sprintf("【🛳活动入口】https://lzdz1-isv.isvjcloud.com/dingzhi/joinCommon/activity/5929859?activityId=%s", activityId))
	signUrlEnv, ok := os.LookupEnv("jd_sign_url")
	if !ok {
		log.Errorln("⚠️未发现jd_sign_url变量,退出程序!")
		os.Exit(1)
	}
	log.Infoln("使用sign: " + signUrlEnv)
	signUrl = signUrlEnv

	client = req.NewClient().SetCommonHeaders(map[string]string{
		`Content-Type`: `application/x-www-form-urlencoded`,
		`User-Agent`:   UA,
		`Referer`:      activityUrl.String(),
	}).
		SetProxy(http.ProxyFromEnvironment)

}

func getSign(fd string, body interface{}) (result string, err error) {
	data, _ := json.Marshal(body)
	response, err := req.NewClient().R().SetFormData(map[string]string{
		"functionId": fd,
		"body":       string(data),
	}).Post(signUrl)
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		log.Errorln("访问sign失败，响应码：" + response.Status)
		return "", errors.New(response.Status)
	}
	resp := response.Bytes()
	if gjson.GetBytes(resp, "code").Int() != 0 {
		log.Errorln("访问sign失败, " + string(resp))
		return "", errors.New(string(resp))
	} else {
		log.Debugln(response.String())
		return gjson.GetBytes(resp, "data").String(), nil
	}
}

func getToken(cookie []*http.Cookie) (string, error) {
	body := map[string]string{"id": "", "url": fmt.Sprintf("https://%s", activityUrl.Host)}
	sign, err := getSign("isvObfuscator", body)
	if err != nil {
		return "", err
	}
	data, _ := json.Marshal(body)
	response, err := client.R().SetCookies(cookie...).SetFormData(map[string]string{
		"body": string(data),
	}).Post(fmt.Sprintf("https://api.m.jd.com?functionId=isvObfuscator%s&eid=%s", sign, utils.RandStr(16)))
	if response.StatusCode != http.StatusOK {
		log.Errorln("获取token失败，响应码：" + response.String())
		return "", errors.New(response.Status)
	}
	resp := response.Bytes()
	if gjson.GetBytes(resp, "code").Int() != 0 {
		log.Errorln("获取token失败, " + string(resp))
		return "", errors.New(string(resp))
	} else {
		log.Debugln(response.String())
		return gjson.GetBytes(resp, "token").String(), nil
	}

}

func getActive() {

	response, err := client.R().Get(activityUrl.String())
	if err != nil || !response.IsSuccessState() {
		log.Errorln(response.Status, "  ⚠️ip疑似黑了,休息一会再来撸~")
		os.Exit(1)
	}
	//client.SetCommonCookies(response.Cookies()...)
}

// {"masterSwitch":0,"activitySwitch":-1,"activityTypeSwitch":-1}
func getSystemConfigForNew() {
	response, err := client.R().Post("https://lzdz1-isv.isvjd.com/wxCommonInfo/getSystemConfigForNew")
	if err != nil || !response.IsSuccessState() {
		log.Errorln(response.Status, "  ⚠️ip疑似黑了,休息一会再来撸~")
		os.Exit(1)
	}
	//client.SetCommonCookies(response.Cookies()...)
}

func accessLogWithAD(venderId, secretPin string) {
	_, err := client.R().SetFormData(map[string]string{
		"venderId":   venderId,
		"code":       "99",
		"pin":        secretPin,
		"activityId": activityId,
		"pageUrl":    activityUrl.String(),
		"subType":    "app",
		"adSource":   "null",
	}).Post("https://lzdz1-isv.isvjcloud.com/common/accessLogWithAD")
	if err != nil {
		log.Errorln("accessLogWithAD调用错误，" + err.Error())
	}
	//client.SetCommonCookies(resp.Cookies()...)
}

// "{"activityId":null,"jdActivityId":11463591,"venderId":1000003443,"shopId":0,"activityType":58}
func getSimpleActInfoVo() (string, error) {
	response, err := client.R().
		SetFormData(map[string]string{"activityId": activityId}).
		Post("https://lzdz1-isv.isvjcloud.com/dz/common/getSimpleActInfoVo")
	if err != nil {
		log.Errorln("getSimpleActInfoVod调用失败," + err.Error())
		return "", err
	}
	client.SetCommonCookies(response.Cookies()...)
	return gjson.GetBytes(response.Bytes(), "data").String(), nil
}

// {"nickname":"j**e","secretPin":"CTZmf45w8Jzw","cid":null}
func getMyPing(venderId, token string) (nickname string, secretPin string, err error) {
	response, err := client.R().
		SetFormData(map[string]string{
			"userId":   venderId,
			"token":    token,
			"fromType": "APP",
		}).
		Post("https://lzdz1-isv.isvjcloud.com/customer/getMyPing")
	if err != nil {
		log.Errorln("getMyPing调用失败," + err.Error())
		return "", "", err
	}
	if !gjson.GetBytes(response.Bytes(), "result").Bool() {
		log.Errorln("getMyPing调用失败," + response.String())
		return "", "", errors.New(response.String())
	} else {
		log.Debugln(response.String())
		return gjson.GetBytes(response.Bytes(), "data.nickname").String(), gjson.GetBytes(response.Bytes(), "data.secretPin").String(), nil
	}
}

func getUserInfo(secretPin string) (nickname string, yunMidImageUrl string, pin string, err error) {
	response, err := client.R().SetFormData(map[string]string{
		"pin": secretPin,
	}).Post("https://lzdz1-isv.isvjcloud.com/wxActionCommon/getUserInfo")
	if err != nil {
		log.Errorln("getUserInfo调用失败," + err.Error())
		return "", "", "", err
	}
	if !gjson.GetBytes(response.Bytes(), "result").Bool() {
		log.Errorln("getUserInfo调用失败," + gjson.GetBytes(response.Bytes(), "errorMessage").String())
		return "", "", "", errors.New(response.String())
	} else {
		log.Debugln(response.String())
		return gjson.GetBytes(response.Bytes(), "data.nickname").String(),
			gjson.GetBytes(response.Bytes(), "data.yunMidImageUrl").String(),
			gjson.GetBytes(response.Bytes(), "data.pin").String(),
			nil
	}
}

type activityContent struct {
	EndTime      int64       `json:"endTime"`
	HasEnd       bool        `json:"hasEnd"`
	ActivityName string      `json:"activityName"`
	JdActivityId int         `json:"jdActivityId"`
	VenderId     int         `json:"venderId"`
	ShopId       int         `json:"shopId"`
	TaskType     string      `json:"taskType"`
	RankType     string      `json:"rankType"`
	ShowRank     int         `json:"showRank"`
	ScoreName    string      `json:"scoreName"`
	Version      string      `json:"version"`
	DrawScore    interface{} `json:"drawScore"`
	ShareConfig  struct {
		ShareCategory  string `json:"shareCategory"`
		ShareTitle     string `json:"shareTitle"`
		ShareContent   string `json:"shareContent"`
		ShareImage     string `json:"shareImage"`
		MiniShareImage string `json:"miniShareImage"`
	} `json:"shareConfig"`
	ActorInfo struct {
		Uuid             string `json:"uuid"`
		Score            int    `json:"score"`
		TotalScore       int    `json:"totalScore"`
		AssistCount      int    `json:"assistCount"`
		TotalAssistCount int    `json:"totalAssistCount"`
		IsShare          bool   `json:"isShare"`
		OpenCardCount    int    `json:"openCardCount"`
		OldMemberCount   int    `json:"oldMemberCount"`
	} `json:"actorInfo"`
}

func shareRecord(userPin, actorUuid string) {
	resp, err := client.R().SetFormData(map[string]string{
		"num":        "30",
		"uuid":       actorUuid,
		"pin":        userPin,
		"activityId": activityId,
	}).Post("https://lzdz1-isv.isvjcloud.com/dingzhi/joinCommon/shareRecord")
	if err != nil {
		log.Errorln("shareRecord调用错误，" + err.Error())
	}
	log.Debugln("shareRecord ==> ", resp.String())
}

func taskRecord(userPin, actorUuid string) {
	resp, err := client.R().SetFormData(map[string]string{
		"taskType":   "",
		"uuid":       actorUuid,
		"pin":        userPin,
		"activityId": activityId,
	}).Post("https://lzdz1-isv.isvjcloud.com/dingzhi/joinCommon/taskRecord")
	if err != nil {
		log.Errorln("shareRecord调用错误，" + err.Error())
	}
	log.Debugln("taskRecord ==> ", resp.String())
}

func doTask(actorUuid, pin string, taskType int) {
	resp, err := client.R().SetFormData(map[string]string{
		"taskType":   strconv.Itoa(taskType),
		"uuid":       actorUuid,
		"pin":        pin,
		"activityId": activityId,
		"taskValue":  "",
	}).Post("https://lzdz1-isv.isvjcloud.com/dingzhi/joinCommon/doTask")
	if err != nil {
		log.Warningln("doTask失败，" + err.Error())
		return
	}
	log.Infoln("doTask --> ", resp.String())
	if !gjson.GetBytes(resp.Bytes(), "result").Bool() {
		log.Errorln("doTask调用失败," + gjson.GetBytes(resp.Bytes(), "errorMessage").String())
		return
	} else {
		if gjson.GetBytes(resp.Bytes(), "data.score").Int() == 0 {
			log.Infoln("\t获得 💨💨💨")
		} else {
			log.Infoln(fmt.Sprintf("获得%v积分", gjson.GetBytes(resp.Bytes(), "data.score").Int()))
		}
	}
}

type assist struct {
	AssistState  int `json:"assistState"`
	OpenCardInfo struct {
		OpenAll         bool  `json:"openAll"`
		Beans           int   `json:"beans"`
		Score           int   `json:"score"`
		SendStatus      bool  `json:"sendStatus"`
		HasNewOpen      bool  `json:"hasNewOpen"`
		OpenVenderId    []int `json:"openVenderId"`
		HasOpenCardTask bool  `json:"hasOpenCardTask"`
	} `json:"openCardInfo"`
}

type TaskInfo struct {
	Field1 struct {
		TaskType    int    `json:"taskType"`
		SortType    string `json:"sortType"`
		TaskScore   int    `json:"taskScore"`
		TaskBeans   int    `json:"taskBeans"`
		SettingInfo []struct {
			Type    int    `json:"type"`
			Value   string `json:"value"`
			Value2  string `json:"value2"`
			Name    string `json:"name"`
			Content string `json:"content"`
			ToUrl   string `json:"toUrl"`
			ImgUrl  string `json:"imgUrl"`
		} `json:"settingInfo"`
	} `json:"1"`
	Field2 struct {
		TaskType    int    `json:"taskType"`
		SortType    string `json:"sortType"`
		TaskScore   int    `json:"taskScore"`
		TaskBeans   int    `json:"taskBeans"`
		SettingInfo []struct {
			Type    int         `json:"type"`
			Value   string      `json:"value"`
			Value2  string      `json:"value2"`
			Name    string      `json:"name"`
			Content string      `json:"content"`
			ToUrl   interface{} `json:"toUrl"`
			ImgUrl  string      `json:"imgUrl"`
		} `json:"settingInfo"`
	} `json:"2"`
	Field3 struct {
		TaskType    int    `json:"taskType"`
		SortType    string `json:"sortType"`
		TaskScore   int    `json:"taskScore"`
		TaskBeans   int    `json:"taskBeans"`
		SettingInfo []struct {
			Type    int    `json:"type"`
			Value   string `json:"value"`
			Value2  string `json:"value2"`
			Name    string `json:"name"`
			Content string `json:"content"`
			ToUrl   string `json:"toUrl"`
			ImgUrl  string `json:"imgUrl"`
		} `json:"settingInfo"`
	} `json:"20"`
	Field4 struct {
		TaskType    int           `json:"taskType"`
		SortType    string        `json:"sortType"`
		TaskScore   int           `json:"taskScore"`
		TaskBeans   int           `json:"taskBeans"`
		SettingInfo []interface{} `json:"settingInfo"`
	} `json:"4"`
	Field5 struct {
		TaskType    int    `json:"taskType"`
		SortType    string `json:"sortType"`
		TaskScore   int    `json:"taskScore"`
		TaskBeans   int    `json:"taskBeans"`
		SettingInfo []struct {
			Type    int         `json:"type"`
			Value   string      `json:"value"`
			Value2  string      `json:"value2"`
			Name    string      `json:"name"`
			Content string      `json:"content"`
			ToUrl   interface{} `json:"toUrl"`
			ImgUrl  string      `json:"imgUrl"`
		} `json:"settingInfo"`
	} `json:"23"`
	Field6 struct {
		TaskType    int    `json:"taskType"`
		SortType    string `json:"sortType"`
		TaskScore   int    `json:"taskScore"`
		TaskBeans   int    `json:"taskBeans"`
		SettingInfo []struct {
			Type    int         `json:"type"`
			Value   string      `json:"value"`
			Value2  string      `json:"value2"`
			Name    string      `json:"name"`
			Content interface{} `json:"content"`
			ToUrl   string      `json:"toUrl"`
			ImgUrl  string      `json:"imgUrl"`
		} `json:"settingInfo"`
	} `json:"24"`
	Field7 struct {
		TaskType    int           `json:"taskType"`
		SortType    interface{}   `json:"sortType"`
		TaskScore   int           `json:"taskScore"`
		TaskBeans   int           `json:"taskBeans"`
		SettingInfo []interface{} `json:"settingInfo"`
	} `json:"9"`
	Field8 struct {
		TaskType    int    `json:"taskType"`
		SortType    string `json:"sortType"`
		TaskScore   int    `json:"taskScore"`
		TaskBeans   int    `json:"taskBeans"`
		SettingInfo []struct {
			Type    int    `json:"type"`
			Value   string `json:"value"`
			Value2  string `json:"value2"`
			Name    string `json:"name"`
			Content string `json:"content"`
			ToUrl   string `json:"toUrl"`
			ImgUrl  string `json:"imgUrl"`
		} `json:"settingInfo"`
	} `json:"10"`
	Field9 struct {
		TaskType    int           `json:"taskType"`
		SortType    string        `json:"sortType"`
		TaskScore   int           `json:"taskScore"`
		TaskBeans   int           `json:"taskBeans"`
		SettingInfo []interface{} `json:"settingInfo"`
	} `json:"13"`
}

func getStruct[T any](name, postUrl string, params map[string]string) (*T, error) {
	response, err := client.R().SetFormData(params).Post(postUrl)

	if err != nil {
		log.Errorln(name + "调用失败," + err.Error())
		return nil, err
	}
	if !gjson.GetBytes(response.Bytes(), "result").Bool() {
		log.Errorln(name + "调用失败," + gjson.GetBytes(response.Bytes(), "errorMessage").String())
		return nil, errors.New(response.String())
	} else {
		log.Debugln(response.String())
		as := new(T)
		data := gjson.GetBytes(response.Bytes(), "data").String()
		err := json.Unmarshal([]byte(data), as)
		if err != nil {
			log.Errorln(name + "结构体解析失败，请开启LOG_DEBUG查看详情")
			return nil, err
		}
		return as, nil
	}
}

func getShopOpenCardInfo(verderId string) string {
	body, _ := json.Marshal(map[string]string{
		"venderId": verderId,
		"channel":  "401",
	})
	response, err := client.R().SetHeaders(map[string]string{
		`Host`:            `api.m.jd.com`,
		`Accept`:          `*/*`,
		`Connection`:      `keep-alive`,
		`User-Agent`:      UA,
		`Accept-Language`: `zh-cn`,
		`Referer`:         `https://shopmember.m.jd.com/`,
		`Accept-Encoding`: `gzip, deflate`,
	}).SetQueryParams(map[string]string{
		"appid":         "jd_shop_member",
		"functionId":    "getShopOpenCardInfo",
		"body":          string(body),
		"client":        "H5",
		"clientVersion": "9.2.0",
		"uuid":          "88888",
	}).Post("https://api.m.jd.com/client.action")
	if err != nil || response == nil {
		return verderId
	}
	if gjson.GetBytes(response.Bytes(), "success").Bool() {
		return gjson.GetBytes(response.Bytes(), "result.shopMemberCardInfo.venderCardName").String()
	} else {
		return verderId
	}
}

func bindWithVender(verderId string) string {
	body := utils.H5stBody{
		AppId: "jd_shop_member",
		Body: map[string]string{
			`venderId`:             verderId,
			`shopId`:               verderId,
			`bindByVerifyCodeFlag`: `1`,
		},
		Client:        "H5",
		ClientVersion: "9.2.0",
		FunctionId:    "bindWithVender",
	}
	h5st := utils.H5st(body, "8adfb", utils.RandInt(16))
	bodyData, _ := json.Marshal(body.Body)
	res, err := client.R().SetHeaders(map[string]string{
		`Host`:            `api.m.jd.com`,
		`Accept`:          `*/*`,
		`Connection`:      `keep-alive`,
		`User-Agent`:      UA,
		`Accept-Language`: `zh-cn`,
		`Referer`:         `https://shopmember.m.jd.com/`,
		`Accept-Encoding`: `gzip, deflate`,
	}).SetQueryParams(map[string]string{
		"appid":         body.AppId,
		"body":          string(bodyData),
		"client":        "H5",
		"clientVersion": "9.2.0",
		"functionId":    "bindWithVender",
		"h5st":          h5st,
	}).Post("https://api.m.jd.com/client.action")
	if err != nil {
		log.Errorln("访问失败，", res.Status)
	}
	if gjson.GetBytes(res.Bytes(), "success").Bool() {
		return gjson.GetBytes(res.Bytes(), "message").String()
	} else {
		log.Errorln("访问失败", gjson.GetBytes(res.Bytes(), "message").String())
	}
	return ""
}

func main() {
	cookies = utils.GetCookie()
	inviteNum := 0
	for i, cookie := range cookies {
		pin, _ := utils.ParseJDCookie(cookie)
		log.Infoln(fmt.Sprintf("开始执行第%d个账号：%s", i+1, pin))
		token, err := getToken(utils.ParseCookieToArray(cookie))
		if err != nil {
			if i == 0 {
				log.Fatalln("车头获取token失败，退出程序")
			} else {
				log.Errorln("获取token失败")
				continue
			}
		}
		client.SetCommonCookies(utils.ParseCookieToArray(cookie)...)
		log.Infoln("获取到token: " + token)
		time.Sleep(500000000)
		getActive()
		time.Sleep(500000000)
		getSystemConfigForNew()
		time.Sleep(300000000)
		venderId := shopId
		simpleActInfoVo, err := getSimpleActInfoVo()
		if err == nil {
			log.Infoln("获取到venderId: " + venderId)
			venderId = gjson.Get(simpleActInfoVo, "venderId").String()
		}
		time.Sleep(200000000)
		nickname, secretPin, err := getMyPing(venderId, token)
		if err != nil {
			if i == 0 {
				log.Fatalln("车头获取token失败，退出程序")
			} else {
				continue
			}
		}
		log.Infoln()
		time.Sleep(200000000)
		accessLogWithAD(venderId, secretPin)
		nickname, yunMidImageUrl, infoPin, err := getUserInfo(secretPin)
		if yunMidImageUrl == "" {
			yunMidImageUrl = "https://img10.360buyimg.com/imgzone/jfs/t1/21383/2/6633/3879/5c5138d8E0967ccf2/91da57c5e2166005.jpg"
		}
		content, err := getStruct[activityContent]("activityContent",
			"https://lzdz1-isv.isvjcloud.com/dingzhi/joinCommon/activityContent",
			map[string]string{
				"activityId": activityId,
				"pin":        infoPin,
				"pinImg":     yunMidImageUrl,
				"nick":       nickname,
				"cjyxPin":    "",
				"cjhyPin":    "",
				"shareUuid":  inviterUuid,
				"adSource":   "",
			})
		if err != nil {
			if i == 0 {
				log.Fatalln("车头获取邀请码失败，退出程序")
			} else {
				continue
			}
		}
		if content.HasEnd {
			log.Fatalln("活动已经结束了，下次早点额！")
		}
		log.Infoln(fmt.Sprintf("✅开启【%s】活动", content.ActivityName))
		if i == 0 {
			log.Infoln(fmt.Sprintf("🛳 已邀请%d, 有效助力%d", content.ActorInfo.TotalAssistCount, content.ActorInfo.AssistCount))
		}
		log.Infoln("邀请码：--》 ", content.ActorInfo.Uuid)
		log.Infoln("准备助力：--》 ", inviterUuid)
		time.Sleep(500000000)
		shareRecord(infoPin, content.ActorInfo.Uuid)
		time.Sleep(500000000)
		taskRecord(infoPin, content.ActorInfo.Uuid)
		doTask(content.ActorInfo.Uuid, infoPin, 20)
		time.Sleep(500000000)
		doTask(content.ActorInfo.Uuid, infoPin, 23)
		time.Sleep(500000000)
		assistContent, err := getStruct[assist](
			"assist",
			"https://lzdz1-isv.isvjcloud.com/dingzhi/joinCommon/assist",
			map[string]string{
				"activityId": activityId,
				"pin":        infoPin,
				"uuid":       content.ActorInfo.Uuid,
				"shareUuid":  inviterUuid,
			})
		if err != nil {
			if i == 0 {
				log.Fatalln("车头获取assistContent失败，退出程序")
			} else {
				continue
			}
		}
		assStat := false
		if assistContent.OpenCardInfo.OpenAll {
			log.Infoln("已完成全部开卡")
			switch assistContent.AssistState {
			case 0:
				log.Warningln("无法助力自己")
			case 1:
				log.Warningln("已完成全部开卡任务，未助力过任何人")
				assStat = true
			case 3:
				log.Warningln("已助力过其他好友")
			default:
				assStat = true
			}
		} else {
			log.Infoln("现在去开卡")
			taskInfo, err := getStruct[TaskInfo]("taskInfo", "https://lzdz1-isv.isvjcloud.com/dingzhi/joinCommon/taskInfo", map[string]string{
				"activityId": activityId,
				"pin":        infoPin,
			})
			if err != nil {
				if i == 0 {
					log.Fatalln("车头获取taskInfo失败，退出程序")
				} else {
					continue
				}
			}
			var taskList []map[string]string
			for _, s := range taskInfo.Field1.SettingInfo {
				isUnOpen := false
				for _, i3 := range assistContent.OpenCardInfo.OpenVenderId {
					if s.Value == strconv.Itoa(i3) {
						isUnOpen = true
					}
				}
				if !isUnOpen {
					taskList = append(taskList, map[string]string{"name": s.Name, "value": s.Value})
				}
			}
			log.Infoln(fmt.Sprintf("共获取到%d个未开卡店铺", len(taskList)))
			for _, m := range taskList {
				log.Infoln("去开卡 ", m["name"], "  ", m["value"])
				getShopOpenCardInfo(m["value"])
				bindResult := bindWithVender(m["value"])
				if bindResult == "" || strings.Contains(bindResult, "火爆") || strings.Contains(bindResult, "失败") || strings.Contains(bindResult, "解绑") {
					log.Errorln(fmt.Sprintf("⛈⛈ %s, %s", m["name"], bindResult))
					assStat = false
					break
				} else {
					log.Infoln(fmt.Sprintf("🎉🎉 %s，%s", m["name"], bindResult))
					assStat = true
					time.Sleep(1500000000)
				}
			}
			_, _ = getStruct[activityContent]("activityContent",
				"https://lzdz1-isv.isvjcloud.com/dingzhi/joinCommon/activityContent",
				map[string]string{
					"activityId": activityId,
					"pin":        infoPin,
					"pinImg":     yunMidImageUrl,
					"nick":       nickname,
					"cjyxPin":    "",
					"cjhyPin":    "",
					"shareUuid":  inviterUuid,
					"adSource":   "",
				})
			shareRecord(infoPin, content.ActorInfo.Uuid)
			time.Sleep(500000000)
			taskRecord(infoPin, content.ActorInfo.Uuid)
			time.Sleep(500000000)
			assistContent1, err := getStruct[assist](
				"assist",
				"https://lzdz1-isv.isvjcloud.com/dingzhi/joinCommon/assist",
				map[string]string{
					"activityId": activityId,
					"pin":        infoPin,
					"uuid":       content.ActorInfo.Uuid,
					"shareUuid":  inviterUuid,
				})
			if assStat && assistContent1.AssistState == 1 {
				log.Infoln("🎉🎉🎉助力成功！")
				if i != 0 {
					inviteNum++
					log.Infoln(fmt.Sprintf("本次车头已助力%d人", inviteNum))
				}
			} else if assStat && assistContent.AssistState == 1 {
				log.Infoln("🎉🎉🎉助力成功！")
				if i != 0 {
					inviteNum++
					log.Infoln(fmt.Sprintf("本次车头已助力%d人", inviteNum))
				}
			}
			if i == 0 {
				log.Infoln(fmt.Sprintf("后面账号全部助力：%s", content.ActorInfo.Uuid))
				inviterUuid = content.ActorInfo.Uuid
				parse, _ := url.Parse(fmt.Sprintf("https://lzdz1-isv.isvjcloud.com/dingzhi/joinCommon/activity/5929859?activityId=%s&shareUuid=%s&adsource=null&shareuserid4minipg=null&lng=00.000000&lat=00.000000&sid=&un_area=&&shopid=%v", activityId, inviterUuid, shopId))
				activityUrl = parse
			}
		}
	}
}
