package api

import (
	"fmt"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/gnasnik/titan-explorer/config"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/errors"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/core/oplog"
	"github.com/gnasnik/titan-explorer/pkg/iptool"
	"github.com/mssola/user_agent"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"
	"net/http"
	"strings"
	"time"
)

const (
	loginStatusFailure = iota
	loginStatusSuccess
)

type login struct {
	Username   string `form:"username" json:"username" binding:"required"`
	Password   string `form:"password" json:"password" binding:"required"`
	VerifyCode string `form:"verify_code" json:"verify_code"`
}

type loginResponse struct {
	Token  string `json:"token"`
	Expire string `json:"expire"`
}

var identityKey = "id"

func jwtGinMiddleware(secretKey string) (*jwt.GinJWTMiddleware, error) {
	return jwt.New(&jwt.GinJWTMiddleware{
		Realm:             "User",
		Key:               []byte(secretKey),
		Timeout:           time.Hour,
		MaxRefresh:        24 * time.Hour,
		IdentityKey:       identityKey,
		SendAuthorization: true,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*model.User); ok {
				return jwt.MapClaims{
					identityKey: v.Username,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			return &model.User{
				Username: claims[identityKey].(string),
			}
		},
		LoginResponse: func(c *gin.Context, code int, token string, expire time.Time) {
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"data": loginResponse{
					Token:  token,
					Expire: expire.Format(time.RFC3339),
				},
			})
		},
		LogoutResponse: func(c *gin.Context, code int) {
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
			})
		},
		Authenticator: func(c *gin.Context) (interface{}, error) {
			loginParams := login{
				Username:   c.Query("username"),
				VerifyCode: c.Query("verify_code"),
				Password:   c.Query("password"),
			}

			signature := c.Query("sign")
			walletAddress := c.Query("address")
			if loginParams.Username == "" {
				return "", jwt.ErrMissingLoginValues
			}
			if loginParams.VerifyCode == "" && loginParams.Password == "" && signature == "" {
				return "", jwt.ErrMissingLoginValues
			}

			userAgent := c.Request.Header.Get("User-Agent")
			ua := user_agent.New(userAgent)
			os := ua.OS()
			browser, _ := ua.Browser()
			clientIP := iptool.GetClientIP(c.Request)

			location, err := GetLocation(c.Request.Context(), clientIP)
			if err != nil {
				log.Errorf("get ip location from iptable cloud: %v", err)
			}

			var loginLocation string
			if location != nil {
				loginLocation = fmt.Sprintf("%s-%s-%s-%s", location.Continent, location.Country, location.Province, location.City)
			}

			defer func() {
				if err != nil {
					log.Errorf("user login: %v", err)
					oplog.AddLoginLog(&model.LoginLog{
						IpAddress:     clientIP,
						Browser:       browser,
						Os:            os,
						Status:        loginStatusFailure,
						Msg:           err.Error(),
						LoginLocation: loginLocation,
					})
					return
				}

				go SetPeakBandwidth(loginParams.Username)

				oplog.AddLoginLog(&model.LoginLog{
					LoginUsername: loginParams.Username,
					LoginLocation: loginLocation,
					IpAddress:     clientIP,
					Browser:       browser,
					Os:            os,
					Status:        loginStatusSuccess,
					Msg:           "success",
				})

			}()

			if signature != "" {
				return loginBySignature(c, loginParams.Username, walletAddress, signature)
			}

			if loginParams.VerifyCode != "" {
				return loginByVerifyCode(c, loginParams.Username, loginParams.VerifyCode)
			}

			if loginParams.Password != "" {
				return loginByPassword(c, loginParams.Username, loginParams.Password)
			}

			return nil, nil
		},
		Unauthorized: func(c *gin.Context, code int, message string) {

			if strings.Contains(message, "Token is expired") {
				message = "Session expired"
			}

			c.JSON(200, gin.H{
				"code":    code,
				"msg":     message,
				"success": false,
			})
		},
		// TokenLookup is a string in the form of "<source>:<name>" that is used
		// to extract token from the request.
		// Optional. Default value "header:Authorization".
		// Possible values:
		// - "header:<name>"
		// - "query:<name>"
		// - "cookie:<name>"
		// - "param:<name>"
		//TokenLookup: "header: Authorization, query: token, cookie: jwt",
		TokenLookup: "header: JwtAuthorization",
		// TokenLookup: "query:token",
		// TokenLookup: "cookie:token",

		// TokenHeadName is a string in the header. Default value is "Bearer"
		TokenHeadName: "Bearer",

		// TimeFunc provides the current time. You can override it to use another time value. This is useful for testing or if your server uses a different time zone than your tokens.
		TimeFunc: time.Now,

		RefreshResponse: func(c *gin.Context, code int, token string, t time.Time) {
			c.Next()
		},
	})
}

func loginByPassword(c *gin.Context, username, password string) (interface{}, error) {
	user, err := dao.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		log.Errorf("get user by username: %v", err)
		return nil, errors.NewErrorCode(errors.UserNotFound, c)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PassHash), []byte(password)); err != nil {
		return nil, errors.NewErrorCode(errors.InvalidPassword, c)
	}

	return &model.User{Uuid: user.Uuid, Username: user.Username, Role: user.Role}, nil
}

func loginBySignature(c *gin.Context, username, address, msg string) (interface{}, error) {
	nonce, err := getNonceFromCache(c.Request.Context(), username, NonceStringTypeSignature)
	if err != nil {
		return nil, errors.NewErrorCode(errors.InvalidParams, c)
	}
	if nonce == "" {
		return nil, errors.NewErrorCode(errors.VerifyCodeExpired, c)
	}
	if address == "" {
		address = username
	}
	recoverAddress, err := VerifyMessage(nonce, msg)
	if strings.ToUpper(recoverAddress) != strings.ToUpper(address) {
		return nil, errors.NewErrorCode(errors.PassWordNotAllowed, c)
	}
	return &model.User{Username: username, Role: 0}, nil
}

func loginByVerifyCode(c *gin.Context, username, inputCode string) (interface{}, error) {
	code, err := getNonceFromCache(c.Request.Context(), username, NonceStringTypeLogin)
	if err != nil {
		log.Errorf("get user by verify code: %v", err)
		return nil, errors.NewErrorCode(errors.InvalidParams, c)
	}
	if code == "" {
		return nil, errors.NewErrorCode(errors.VerifyCodeExpired, c)
	}
	user, err := dao.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		log.Errorf("get user by username: %v", err)
		return nil, errors.NewErrorCode(errors.UserNotFound, c)
	}
	if code != inputCode {
		return nil, errors.NewErrorCode(errors.InvalidVerifyCode, c)
	}

	return &model.User{Uuid: user.Uuid, Username: user.Username, Role: user.Role}, nil
}

func AuthRequired(authMiddleware *jwt.GinJWTMiddleware) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		claims, e := authMiddleware.GetClaimsFromJWT(ctx)
		if e == nil {
			if int64(claims["exp"].(float64)-authMiddleware.Timeout.Seconds()/2) < authMiddleware.TimeFunc().Unix() {
				tokenString, _, e := authMiddleware.RefreshToken(ctx)
				if e == nil {
					go SetPeakBandwidth(ctx.Query("user_id"))
					ctx.Header("new-token", tokenString)
				}
			}
		}
		ctx.Next()
	}
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, jwtauthorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}
		c.Next()
	}
}

func GetLocation(ctx context.Context, ipAddr string) (*model.Location, error) {
	var location model.Location

	err := dao.GetLocationInfoByIp(ctx, ipAddr, &location, model.LanguageEN)
	if err != nil {
		log.Errorf("get location by ip: %v", err)
	}

	if location == (model.Location{}) {
		return &location, nil
	}

	loc, err := iptool.IPDataCloudGetLocation(ctx, config.Cfg.IpDataCloud.Url, ipAddr, config.Cfg.IpDataCloud.Key, model.LanguageEN)
	if err != nil {
		log.Errorf("get ip location from iptable cloud: %v", err)
	}

	return loc, nil
}
