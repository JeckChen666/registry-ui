package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

const sessionCookieName = "registry_ui_session"

// loginAttempt tracks failed login attempts for brute-force protection.
type loginAttempt struct {
	count    int
	lastFail time.Time
}

var (
	loginAttempts = make(map[string]*loginAttempt)
	loginMu       sync.Mutex
)

func checkRateLimit(ip string) (blocked bool, remaining int) {
	maxAttempts := viper.GetInt("auth.max_attempts")
	if maxAttempts <= 0 {
		return false, 0
	}
	lockoutMin := viper.GetInt("auth.lockout_minutes")
	if lockoutMin <= 0 {
		lockoutMin = 15
	}

	loginMu.Lock()
	defer loginMu.Unlock()

	attempt, exists := loginAttempts[ip]
	if !exists {
		return false, maxAttempts
	}

	// Reset if lockout period has passed.
	if time.Since(attempt.lastFail) > time.Duration(lockoutMin)*time.Minute {
		delete(loginAttempts, ip)
		return false, maxAttempts
	}

	if attempt.count >= maxAttempts {
		return true, 0
	}
	return false, maxAttempts - attempt.count
}

func recordFailure(ip string) {
	loginMu.Lock()
	defer loginMu.Unlock()

	attempt, exists := loginAttempts[ip]
	if !exists {
		loginAttempts[ip] = &loginAttempt{count: 1, lastFail: time.Now()}
		return
	}
	attempt.count++
	attempt.lastFail = time.Now()
}

func clearAttempts(ip string) {
	loginMu.Lock()
	defer loginMu.Unlock()
	delete(loginAttempts, ip)
}

func computeSignature(username, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(username))
	return hex.EncodeToString(mac.Sum(nil))
}

func getSessionUsername(c echo.Context) string {
	cookie, err := c.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	parts := strings.SplitN(cookie.Value, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	username, signature := parts[0], parts[1]
	expected := computeSignature(username, viper.GetString("auth.secret"))
	if hmac.Equal([]byte(signature), []byte(expected)) {
		return username
	}
	return ""
}

func authMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !viper.GetBool("auth.enabled") {
				return next(c)
			}

			// Skip auth for certain paths.
			path := c.Request().URL.Path
			basePath := strings.TrimRight(viper.GetString("uri_base_path"), "/")

			skipPaths := []string{
				"/login",
				"/logout",
				"/favicon.ico",
				"/event-receiver",
				"/static/",
			}
			for _, sp := range skipPaths {
				fullPath := basePath + sp
				if path == fullPath || strings.HasPrefix(path, fullPath) {
					return next(c)
				}
			}

			username := getSessionUsername(c)
			if username != "" {
				// Set the username in the header so setUserPermissions works.
				c.Request().Header.Set(usernameHTTPHeader, username)
				return next(c)
			}

			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("%s/login", basePath))
		}
	}
}

func loginHandler(c echo.Context) error {
	basePath := strings.TrimRight(viper.GetString("uri_base_path"), "/")
	data := jet.VarMap{}
	data.Set("user", "")
	data.Set("eventsAllowed", false)
	data.Set("deleteAllowed", false)

	ip := c.RealIP()

	if c.Request().Method == http.MethodPost {
		blocked, _ := checkRateLimit(ip)
		if blocked {
			lockoutMin := viper.GetInt("auth.lockout_minutes")
			if lockoutMin <= 0 {
				lockoutMin = 15
			}
			data.Set("error", fmt.Sprintf("登录失败次数过多，请在 %d 分钟后重试。", lockoutMin))
			return c.Render(http.StatusOK, "login.html", data)
		}

		username := c.FormValue("username")
		password := c.FormValue("password")

		if username == viper.GetString("auth.username") && password == viper.GetString("auth.password") {
			clearAttempts(ip)
			signature := computeSignature(username, viper.GetString("auth.secret"))
			cookieValue := fmt.Sprintf("%s:%s", username, signature)
			cookie := &http.Cookie{
				Name:     sessionCookieName,
				Value:    cookieValue,
				Path:     "/",
				HttpOnly: true,
			}
			c.SetCookie(cookie)
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("%s/", basePath))
		}

		recordFailure(ip)
		_, remaining := checkRateLimit(ip)
		data.Set("error", fmt.Sprintf("用户名或密码错误，剩余 %d 次尝试机会。", remaining))
	}

	return c.Render(http.StatusOK, "login.html", data)
}

func logoutHandler(c echo.Context) error {
	basePath := strings.TrimRight(viper.GetString("uri_base_path"), "/")
	cookie := &http.Cookie{
		Name:   sessionCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	c.SetCookie(cookie)
	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("%s/login", basePath))
}
