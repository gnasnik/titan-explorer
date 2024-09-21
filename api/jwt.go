package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/errors"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/core/geo"
	"github.com/gnasnik/titan-explorer/core/oplog"
	"github.com/gnasnik/titan-explorer/core/storage"
	"github.com/gnasnik/titan-explorer/pkg/iptool"
	"github.com/gnasnik/titan-explorer/pkg/opcheck"
	"github.com/mssola/user_agent"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"
)

const (
	loginStatusFailure = iota
	loginStatusSuccess
)

type login struct {
	Username   string `form:"username" json:"username"`
	Password   string `form:"password" json:"password"`
	VerifyCode string `form:"verify_code" json:"verify_code"`
	Sign       string `form:"sign" json:"sign"`
	Address    string `form:"address" json:"address"`
	PublicKey  string `form:"publicKey" json:"publicKey"`
}

type loginResponse struct {
	Token  string `json:"token"`
	Expire string `json:"expire"`
}

var (
	identityKey = "id"
	roleKey     = "role"
)

func jwtGinMiddleware(secretKey string) (*jwt.GinJWTMiddleware, error) {
	return jwt.New(&jwt.GinJWTMiddleware{
		Realm:             "User",
		Key:               []byte(secretKey),
		Timeout:           24 * time.Hour,
		MaxRefresh:        7 * 24 * time.Hour,
		IdentityKey:       identityKey,
		SendAuthorization: true,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*model.User); ok {
				return jwt.MapClaims{
					identityKey: v.Username,
					roleKey:     v.Role,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)

			var role int32
			_, ok := claims[roleKey]
			if ok {
				role = int32(claims[roleKey].(float64))
			}

			return &model.User{
				Username: claims[identityKey].(string),
				Role:     role,
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
			var loginParams login
			if err := c.BindJSON(&loginParams); err != nil {
				return "", fmt.Errorf("invalid input params")
			}

			if loginParams.Username == "" {
				return "", jwt.ErrMissingLoginValues
			}
			if loginParams.VerifyCode == "" && loginParams.Password == "" && loginParams.Sign == "" {
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

			if loginParams.Sign != "" {
				return loginBySignature(c, loginParams.Username, loginParams.Address, loginParams.Sign, loginParams.PublicKey)
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
				msg := "Session expired, please log in again"

				if c.GetHeader("Lang") == "cn" {
					msg = "会话已过期, 请重新登陆"
				}

				message = msg
			}

			c.JSON(http.StatusOK, gin.H{
				"code":    401,
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
	if err == sql.ErrNoRows {
		return nil, errors.NewErrorCode(errors.UserNotFound, c)
	}

	if err != nil {
		log.Errorf("get user by username: %v", err)
		return nil, errors.NewErrorCode(errors.InternalServer, c)
	}

	if user.TotalStorageSize == 0 {
		err = dao.UpdateUserTotalSize(c.Request.Context(), user.Username, 100*1024*1024)
		if err != nil {
			log.Errorf(err.Error())
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PassHash), []byte(password)); err != nil {
		return nil, errors.NewErrorCode(errors.InvalidPassword, c)
	}

	return &model.User{Uuid: user.Uuid, Username: user.Username, Role: user.Role}, nil
}

func loginBySignature(c *gin.Context, username, address, msg, pk string) (interface{}, error) {
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
	if strings.TrimSpace(pk) == "" {
		recoverAddress, _ := VerifyMessage(nonce, msg)
		if !strings.EqualFold(recoverAddress, address) {
			return nil, errors.NewErrorCode(errors.PassWordNotAllowed, c)
		}
	} else {
		match, _ := opcheck.VerifyComosSign(address, nonce, msg, pk)
		if !match {
			return nil, errors.NewErrorCode(errors.PassWordNotAllowed, c)
		}
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
	if err == sql.ErrNoRows {
		return nil, errors.NewErrorCode(errors.UserNotFound, c)
	}

	if err != nil {
		log.Errorf("get user by username: %v", err)
		return nil, errors.NewErrorCode(errors.InternalServer, c)
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
			switch v := claims["exp"].(type) {
			case nil:
				authMiddleware.Unauthorized(ctx, http.StatusUnauthorized, authMiddleware.HTTPStatusMessageFunc(jwt.ErrMissingExpField, ctx))
				return
			case float64:
				if int64(v) < authMiddleware.TimeFunc().Unix() {
					authMiddleware.Unauthorized(ctx, http.StatusUnauthorized, authMiddleware.HTTPStatusMessageFunc(jwt.ErrExpiredToken, ctx))
					return
				}
			case json.Number:
				n, err := v.Int64()
				if err != nil {
					authMiddleware.Unauthorized(ctx, http.StatusUnauthorized, authMiddleware.HTTPStatusMessageFunc(jwt.ErrWrongFormatOfExp, ctx))
					return
				}
				if n < authMiddleware.TimeFunc().Unix() {
					authMiddleware.Unauthorized(ctx, http.StatusUnauthorized, authMiddleware.HTTPStatusMessageFunc(jwt.ErrExpiredToken, ctx))
					return
				}
			default:
				authMiddleware.Unauthorized(ctx, http.StatusUnauthorized, authMiddleware.HTTPStatusMessageFunc(jwt.ErrWrongFormatOfExp, ctx))
				return
			}
			ctx.Set("JWT_PAYLOAD", claims)
			identity := authMiddleware.IdentityHandler(ctx)

			if identity != nil {
				ctx.Set(authMiddleware.IdentityKey, identity)
			}

			if !authMiddleware.Authorizator(identity, ctx) {
				authMiddleware.Unauthorized(ctx, http.StatusUnauthorized, authMiddleware.HTTPStatusMessageFunc(jwt.ErrForbidden, ctx))
				return
			}
			if int64(claims["exp"].(float64)-authMiddleware.Timeout.Seconds()/2) < authMiddleware.TimeFunc().Unix() {
				tokenString, _, e := authMiddleware.RefreshToken(ctx)
				if e == nil {
					go SetPeakBandwidth(ctx.Query("user_id"))
					ctx.Header("new-token", tokenString)
				}
			}
		} else {
			apiKey := ctx.GetHeader("apiKey")
			uid, err := storage.AesDecryptCBCByKey(apiKey)
			if err != nil {
				authMiddleware.Unauthorized(ctx, http.StatusUnauthorized, authMiddleware.HTTPStatusMessageFunc(jwt.ErrForbidden, ctx))
				return
			}
			ctx.Set("JWT_PAYLOAD", jwt.MapClaims{
				identityKey: uid,
			})
		}
		ctx.Next()
	}
}

func AdminOnly(data interface{}, c *gin.Context) bool {
	user, ok := data.(*model.User)
	if ok && model.UserRole(user.Role) >= model.UserRoleAdmin {
		return true
	}
	return false
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, jwtauthorization, lang")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}
		c.Next()
	}
}

func GetLocation(ctx context.Context, ipAddr string) (*model.Location, error) {
	//var location model.Location

	//err := dao.GetLocationInfoByIp(ctx, ipAddr, &location, model.LanguageEN)
	//if err != nil {
	//	log.Errorf("get location by ip: %v", err)
	//}
	loc, err := geo.GetIpLocation(ctx, ipAddr, model.LanguageEN)
	if err != nil || loc == nil {
		log.Errorf("get ip location %v", err)
		// applyLocationFromLocalGEODB(deviceInfo)
	}

	return loc, nil
}
