package utils

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

func GetCookie() []string {
	var cookies []string
	env, b := os.LookupEnv("JD_COOKIE")
	if b {
		if strings.Contains(env, "&") {
			cookies = append(cookies, strings.Split(env, "&")...)
		} else if strings.Contains(env, "\n") {
			cookies = append(cookies, strings.Split(env, "\n")...)
		} else {
			cookies = append(cookies, env)
		}
	} else {
		if PathExists("JD_COOKIE.txt") {
			data, _ := os.ReadFile("JD_COOKIE.txt")
			env = string(data)
			if strings.Contains(env, "&") {
				cookies = append(cookies, strings.Split(env, "&")...)
			} else if strings.Contains(env, "\n") {
				cookies = append(cookies, strings.Split(env, "\n")...)
			} else {
				cookies = append(cookies, env)
			}
		} else {
			fmt.Println("未获取到正确✅格式的京东账号Cookie")
		}
	}
	fmt.Println(fmt.Sprintf("====================共%d个京东账号Cookie=========\n", len(cookies)))
	fmt.Println(fmt.Sprintf("==================脚本执行- 北京时间(UTC+8)：%s=====================\n", time.Now().Format("2006-01-02 15:04:05")))

	return cookies
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func ParseJDCookie(cookie string) (pin string, key string) {
	result := regexp.MustCompile(`pt_key=(.*?);pt_pin=(.*?);`).FindAllStringSubmatch(cookie, -1)
	pin = result[0][2]
	key = result[0][1]
	return
}

func ParseCookieToArray(cookie string) []*http.Cookie {
	var cookies []*http.Cookie
	indexs := strings.Split(strings.TrimSuffix(cookie, ";"), ";")
	for _, index := range indexs {
		data := strings.Split(index, "=")
		cookies = append(cookies, &http.Cookie{
			Name:   data[0],
			Value:  data[1],
			Path:   "/",
			Domain: "jd.com",
		})
	}
	return cookies
}

type MyCookieJar struct {
	data map[string]map[string]*http.Cookie
}

func (m *MyCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	if m.data == nil {
		m.data = make(map[string]map[string]*http.Cookie, 5)
	}
	addCookies := make(map[string]*http.Cookie, len(cookies))
	for _, cookie := range cookies {
		addCookies[cookie.Name] = cookie
	}
	cks, ok := m.data[u.String()]
	if !ok {
		m.data[u.String()] = addCookies
	} else {
		for s, cookie := range addCookies {
			cks[s] = cookie
		}
		m.data[u.String()] = cks
	}

}

func (m *MyCookieJar) Cookies(u *url.URL) []*http.Cookie {
	var cookies []*http.Cookie
	for _, cookie := range m.data[u.String()] {
		cookies = append(cookies, cookie)
	}
	return cookies
}
