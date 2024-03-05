package api

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/errors"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	"github.com/gnasnik/titan-explorer/pkg/random"
	"github.com/gnasnik/titan-explorer/pkg/rsa"
	"github.com/go-redis/redis/v9"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type NonceStringType string

const (
	NonceStringTypeRegister  NonceStringType = "1"
	NonceStringTypeLogin     NonceStringType = "2"
	NonceStringTypeReset     NonceStringType = "3"
	NonceStringTypeSignature NonceStringType = "4"
)

var defaultNonceExpiration = 5 * time.Minute

func GetUserInfoHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)
	user, err := dao.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.UserNotFound, c))
		return
	}
	counter, err := dao.CountUserDeviceInfo(c.Request.Context(), username)
	if err != nil {
		//c.JSON(http.StatusOK, respErrorCode(errors.UserNotFound, c))
		//return
		log.Errorf("CountUserDeviceInfo %s: %v", username, err)
	}

	if counter != nil {
		user.Reward = int64(counter.CumulativeProfit)
	}

	c.JSON(http.StatusOK, respJSON(user))
}

func UserRegister(c *gin.Context) {
	userInfo := &model.User{
		Username:     c.Query("username"),
		UserEmail:    c.Query("username"),
		Referrer:     c.Query("referrer"),
		ReferralCode: random.GenerateRandomString(6),
		CreatedAt:    time.Now(),
	}

	verifyCode := c.Query("verify_code")
	passwd := c.Query("password")
	if userInfo.Username == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}
	_, err := dao.GetUserByUsername(c.Request.Context(), userInfo.Username)
	if err == nil {
		c.JSON(http.StatusOK, respErrorCode(errors.UserEmailExists, c))
		return
	}
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	var referrer *model.User
	if userInfo.Referrer != "" {
		referrer, err = dao.GetUserByRefCode(c.Request.Context(), userInfo.Referrer)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.InvalidReferralCode, c))
			return
		}
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.PassWordNotAllowed, c))
		return
	}
	userInfo.PassHash = string(passHash)

	nonce, err := getNonceFromCache(c.Request.Context(), userInfo.Username, NonceStringTypeRegister)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	if nonce == "" || verifyCode == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidVerifyCode, c))
		return
	}

	if nonce != verifyCode && os.Getenv("TEST_ENV_VERIFY_CODE") != verifyCode {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidVerifyCode, c))
		return
	}

	err = dao.CreateUser(c.Request.Context(), userInfo)
	if err != nil {
		log.Errorf("create user : %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	if referrer != nil {
		rewardStatement := &model.RewardStatement{
			Username:  referrer.Username,
			FromUser:  userInfo.Username,
			Amount:    0,
			Event:     model.RewardEventInviteFrens,
			Status:    1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := dao.UpdateUserReward(c.Request.Context(), rewardStatement)
		if err != nil {
			log.Errorf("Update user reward: %v", err)
		}
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
}

func PasswordRest(c *gin.Context) {
	username := c.Query("username")
	verifyCode := c.Query("verify_code")
	passwd := c.Query("password")

	_, err := dao.GetUserByUsername(c.Request.Context(), username)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, respErrorCode(errors.NameNotExists, c))
		return
	}
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.PassWordNotAllowed, c))
		return
	}

	nonce, err := getNonceFromCache(c.Request.Context(), username, NonceStringTypeReset)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.Unknown, c))
		return
	}

	if nonce == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.VerifyCodeExpired, c))
		return
	}

	if verifyCode == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidVerifyCode, c))
		return
	}

	if nonce != verifyCode && os.Getenv("TEST_ENV_VERIFY_CODE") != verifyCode {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidVerifyCode, c))
		return
	}

	err = dao.ResetPassword(c.Request.Context(), string(passHash), username)
	if err != nil {
		log.Errorf("update user : %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
}

func GetNonceStringHandler(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	nonce, err := generateNonceString(c.Request.Context(), getRedisNonceSignatureKey(username))
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	_, err = dao.GetUserByUsername(c.Request.Context(), username)
	if err == sql.ErrNoRows {
		//c.JSON(http.StatusOK, respErrorCode(errors.UserNotFound, c))
		//return
		user := &model.User{
			Username:     username,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			ReferralCode: random.GenerateRandomString(6),
		}
		err = dao.CreateUser(c.Request.Context(), user)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
	}

	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"code": nonce,
	}))
}

func generateNonceString(ctx context.Context, key string) (string, error) {
	rand := random.GenerateRandomNumber(6)
	verifyCode := "TitanNetWork(" + rand + ")"
	bytes, err := json.Marshal(verifyCode)
	if err != nil {
		return "", err
	}

	_, err = dao.RedisCache.Set(ctx, key, bytes, defaultNonceExpiration).Result()
	if err != nil {
		log.Errorf("%v:", err)
		return "", err
	}

	return verifyCode, nil
}

func GetNumericVerifyCodeHandler(c *gin.Context) {
	userInfo := &model.User{}
	userInfo.Username = c.Query("username")
	verifyType := c.Query("type")
	lang := c.GetHeader("Lang")
	userInfo.UserEmail = userInfo.Username

	var key string
	switch NonceStringType(verifyType) {
	case NonceStringTypeRegister:
		key = getRedisNonceRegisterKey(userInfo.Username)
	case NonceStringTypeLogin:
		key = getRedisNonceLoginKey(userInfo.Username)
	case NonceStringTypeReset:
		key = getRedisNonceResetKey(userInfo.Username)
	case NonceStringTypeSignature:
		key = getRedisNonceSignatureKey(userInfo.Username)
	default:
		c.JSON(http.StatusOK, respErrorCode(errors.UnsupportedVerifyCodeType, c))
		return
	}

	nonce, err := getNonceFromCache(c.Request.Context(), userInfo.Username, NonceStringType(verifyType))
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	if nonce != "" {
		c.JSON(http.StatusOK, respErrorCode(errors.GetVCFrequently, c))
		return
	}

	verifyCode := random.GenerateRandomNumber(6)

	if err = sendEmail(userInfo.Username, verifyCode, lang); err != nil {
		log.Errorf("send email: %v", err)
		if strings.Contains(err.Error(), "timed out") {
			c.JSON(http.StatusOK, respErrorCode(errors.TimeoutCode, c))
			return
		}
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	if err = cacheVerifyCode(c.Request.Context(), key, verifyCode); err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
}

func DeviceBindingHandler(c *gin.Context) {
	var params model.Signature

	if err := c.BindJSON(&params); err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	if params.Signature == "" || params.NodeId == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	sign, err := dao.GetSignatureByHash(c.Request.Context(), params.Hash)
	if err == dao.ErrNoRow {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidSignature, c))
		return
	}

	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	deviceInfo, err := dao.GetDeviceInfo(c.Request.Context(), params.NodeId)
	if err == dao.ErrNoRow {

		device, err := getNodeInfoFromScheduler(c.Request.Context(), params.NodeId)
		if err != nil {
			log.Errorf("getNodeInfoFromScheduler %v", err)
			c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
			return
		}

		deviceInfo = device
		deviceInfo.DeviceRank = 999
		err = dao.BulkAddDeviceInfo(c.Request.Context(), []*model.DeviceInfo{deviceInfo})
		if err != nil {
			log.Errorf("BulkAddDeviceInfo %v", err)
		}
	}

	if deviceInfo == nil {
		c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
		return
	}

	if deviceInfo.UserID != "" {
		c.JSON(http.StatusOK, respErrorCode(errors.DeviceBound, c))
		return
	}

	if params.AreaId == "" {
		params.AreaId = dao.GetAreaID(c.Request.Context(), sign.Username)
	}

	schedulerClient, err := getSchedulerClient(c.Request.Context(), params.AreaId)
	if err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
		return
	}

	pubKeyString, err := schedulerClient.GetNodePublicKey(c.Request.Context(), params.NodeId)
	if err != nil {
		log.Errorf("api get node public key: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	pubicKey, err := rsa.Pem2PublicKey([]byte(pubKeyString))
	if err != nil {
		log.Errorf("pem 2 publicKey: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	signature, err := hex.DecodeString(params.Signature)
	if err != nil {
		log.Errorf("hex decode: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidSignature, c))
		return
	}

	err = rsa.VerifySHA256Sign(pubicKey, signature, []byte(params.Hash))
	if err != nil {
		log.Errorf("pem 2 publicKey: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidSignature, c))
		return
	}

	if err = dao.UpdateUserDeviceInfo(c.Request.Context(), &model.DeviceInfo{
		UserID:     sign.Username,
		DeviceID:   params.NodeId,
		BindStatus: "binding",
	}); err != nil {
		log.Errorf("update device binding status: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	if sign.Signature == "" {
		err = dao.UpdateSignature(c.Request.Context(), params.Signature, params.NodeId, params.AreaId, params.Hash)
		if err != nil {
			log.Errorf("update signature: %v", err)
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
	} else {
		params.Username = sign.Username
		err = dao.AddSignature(c.Request.Context(), &params)
		if err != nil {
			log.Errorf("add signature: %v", err)
			c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
			return
		}
	}

	c.JSON(http.StatusOK, respJSON(nil))

}

// DeviceBindingHandlerOld Deprecate using DeviceBindingHandler instead of
func DeviceBindingHandlerOld(c *gin.Context) {
	deviceInfo := &model.DeviceInfo{}
	deviceInfo.DeviceID = c.Query("device_id")
	deviceInfo.UserID = c.Query("user_id")
	deviceInfo.BindStatus = c.Query("band_status")

	old, err := dao.GetDeviceInfoByID(c.Request.Context(), deviceInfo.DeviceID)
	if err != nil {
		log.Errorf("get user device: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	if old != nil && old.UserID != "" && old.BindStatus == deviceInfo.BindStatus {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}
	if deviceInfo.UserID != "" {
		areaId := dao.GetAreaID(c.Request.Context(), deviceInfo.UserID)
		schedulerClient, err := getSchedulerClient(c.Request.Context(), areaId)
		if err != nil {
			c.JSON(http.StatusOK, respErrorCode(errors.NoSchedulerFound, c))
			return
		}
		if deviceInfo.BindStatus == "binding" {
			deviceInfo.ActiveStatus = 1
			err = schedulerClient.UndoNodeDeactivation(c.Request.Context(), deviceInfo.DeviceID)
			if err != nil {
				log.Errorf("api UndoNodeDeactivation: %v", err)
				c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
				return
			}
		}
		if deviceInfo.BindStatus == "unbinding" {
			deviceInfo.ActiveStatus = 2
			err = schedulerClient.DeactivateNode(c.Request.Context(), deviceInfo.DeviceID, 24)
			if err != nil {
				log.Errorf("api DeactivateNode: %v", err)
				c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
				return
			}
		}
	}

	if old != nil && old.BoundAt.IsZero() {
		deviceInfo.BoundAt = time.Now()
	}

	err = dao.UpdateUserDeviceInfo(c.Request.Context(), deviceInfo)
	if err != nil {
		log.Errorf("update user device: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
}

func DeviceUnBindingHandlerOld(c *gin.Context) {
	deviceInfo := &model.DeviceInfo{}
	deviceInfo.DeviceID = c.Query("device_id")
	UserID := c.Query("user_id")
	deviceInfo.BindStatus = "unbinding"
	deviceInfo.ActiveStatus = 2

	old, err := dao.GetDeviceInfoByID(c.Request.Context(), deviceInfo.DeviceID)
	if err != nil {
		log.Errorf("get user device: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	if old == nil {
		c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
		return
	}

	if old.UserID != UserID {
		c.JSON(http.StatusOK, respErrorCode(errors.UnbindingNotAllowed, c))
		return
	}

	err = dao.UpdateUserDeviceInfo(c.Request.Context(), deviceInfo)
	if err != nil {
		log.Errorf("update user device: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
}

func DeviceUpdateHandler(c *gin.Context) {
	deviceInfo := &model.DeviceInfo{}
	deviceInfo.DeviceID = c.Query("device_id")
	deviceInfo.UserID = c.Query("user_id")
	deviceInfo.DeviceName = c.Query("device_name")

	old, err := dao.GetDeviceInfoByID(c.Request.Context(), deviceInfo.DeviceID)
	if err != nil {
		log.Errorf("get user device: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}
	if old == nil {
		c.JSON(http.StatusOK, respErrorCode(errors.DeviceNotExists, c))
		return
	}

	err = dao.UpdateDeviceName(c.Request.Context(), deviceInfo)
	if err != nil {
		log.Errorf("update user device: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"msg": "success",
	}))
}

func cacheVerifyCode(ctx context.Context, key, verifyCode string) error {
	bytes, err := json.Marshal(verifyCode)
	if err != nil {
		return err
	}

	_, err = dao.RedisCache.Set(ctx, key, bytes, defaultNonceExpiration).Result()
	if err != nil {
		return err
	}

	return nil
}

func SetPeakBandwidth(userId string) {
	peakBandwidth, err := dao.GetPeakBandwidth(context.Background(), userId)
	if err != nil {
		log.Errorf("get peak bandwidth: %v", err)
		return
	}
	var expireTime time.Duration
	expireTime = time.Hour
	_ = SetUserInfo(context.Background(), userId, peakBandwidth, expireTime)
	return
}

func SetUserInfo(ctx context.Context, key string, peakBandwidth int64, expireTime time.Duration) error {
	bytes, err := json.Marshal(peakBandwidth)
	vc := GetUserInfo(ctx, key)
	if vc != 0 {
		_, err := dao.RedisCache.Expire(ctx, key, expireTime).Result()
		if err != nil {
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}
	_, err = dao.RedisCache.Set(ctx, key, bytes, expireTime).Result()
	if err != nil {
		return err
	}
	return nil
}

func getRedisNonceSignatureKey(username string) string {
	return fmt.Sprintf("TITAN::SIGN::%s", username)
}

func getRedisNonceRegisterKey(username string) string {
	return fmt.Sprintf("TITAN::REG::%s", username)
}

func getRedisNonceLoginKey(username string) string {
	return fmt.Sprintf("TITAN::LOGIN::%s", username)
}

func getRedisNonceResetKey(username string) string {
	return fmt.Sprintf("TITAN::RESET::%s", username)
}

func getNonceFromCache(ctx context.Context, username string, t NonceStringType) (string, error) {
	var key string

	switch t {
	case NonceStringTypeRegister:
		key = getRedisNonceRegisterKey(username)
	case NonceStringTypeLogin:
		key = getRedisNonceLoginKey(username)
	case NonceStringTypeReset:
		key = getRedisNonceResetKey(username)
	case NonceStringTypeSignature:
		key = getRedisNonceSignatureKey(username)
	default:
		return "", fmt.Errorf("unsupported nonce string type")
	}

	bytes, err := dao.RedisCache.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	var verifyCode string
	err = json.Unmarshal(bytes, &verifyCode)
	if err != nil {
		return "", err
	}

	return verifyCode, nil
}

func VerifyMessage(message string, signedMessage string) (string, error) {
	// Hash the unsigned message using EIP-191
	hashedMessage := []byte("\x19Ethereum Signed Message:\n" + strconv.Itoa(len(message)) + message)
	hash := crypto.Keccak256Hash(hashedMessage)
	// Get the bytes of the signed message
	decodedMessage := hexutil.MustDecode(signedMessage)
	// Handles cases where EIP-115 is not implemented (most wallets don't implement it)
	if decodedMessage[64] == 27 || decodedMessage[64] == 28 {
		decodedMessage[64] -= 27
	}
	// Recover a public key from the signed message
	sigPublicKeyECDSA, err := crypto.SigToPub(hash.Bytes(), decodedMessage)
	if sigPublicKeyECDSA == nil {
		log.Errorf("Could not get a public get from the message signature")
	}
	if err != nil {
		return "", err
	}

	return crypto.PubkeyToAddress(*sigPublicKeyECDSA).String(), nil
}

func GetUserInfo(ctx context.Context, key string) int64 {
	bytes, err := dao.RedisCache.Get(ctx, key).Bytes()
	if err != nil && err != redis.Nil {
		return 0
	}
	if err == redis.Nil {
		return 0
	}
	var peakBandwidth int64
	err = json.Unmarshal(bytes, &peakBandwidth)
	if err != nil {
		return 0
	}
	return peakBandwidth
}

func BindWalletHandler(c *gin.Context) {
	type bindParams struct {
		Username   string `json:"username"`
		VerifyCode string `json:"verify_code"`
		Sign       string `json:"sign"`
		Address    string `json:"address"`
	}

	var param bindParams
	if err := c.BindJSON(&param); err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	nonce, err := getNonceFromCache(c.Request.Context(), param.Username, NonceStringTypeSignature)
	if err != nil {
		log.Errorf("query nonce string: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	if nonce == "" {
		c.JSON(http.StatusOK, respErrorCode(errors.VerifyCodeExpired, c))
		return
	}

	recoverAddress, err := VerifyMessage(nonce, param.Sign)
	if strings.ToUpper(recoverAddress) != strings.ToUpper(param.Address) {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidSignature, c))
		return
	}

	user, err := dao.GetUserByUsername(c.Request.Context(), param.Username)
	if err != nil || user == nil {
		c.JSON(http.StatusOK, respErrorCode(errors.UserNotFound, c))
		return
	}

	if user.WalletAddress != "" {
		c.JSON(http.StatusOK, respErrorCode(errors.WalletBound, c))
		return
	}

	if err := dao.UpdateUserWalletAddress(context.Background(), param.Username, recoverAddress); err != nil {
		log.Errorf("update user wallet address: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(nil))
}

func UnBindWalletHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)

	ctx := context.Background()
	user, err := dao.GetUserByUsername(ctx, username)
	if err != nil {
		log.Errorf("get user by username: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	if user == nil {
		log.Errorf("user not found: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	if err := dao.UpdateUserWalletAddress(context.Background(), user.Username, ""); err != nil {
		log.Errorf("update user wallet address: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(nil))
}

func GetReferralListHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)

	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	order := c.Query("order")
	orderField := c.Query("order_field")
	option := dao.QueryOption{
		Page:       page,
		PageSize:   pageSize,
		Order:      order,
		OrderField: orderField,
	}

	event := []model.RewardEvent{model.RewardEventInviteFrens, model.RewardEventBindDevice}
	total, referList, err := dao.GetReferralList(c.Request.Context(), username, event, option)
	if err != nil {
		log.Errorf("get referral list: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	totalReward, err := dao.GetUserReferralReward(c.Request.Context(), username)
	if err != nil {
		log.Errorf("get user referral reward: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":         referList,
		"total":        total,
		"total_reward": totalReward,
	}))
}

func WithdrawHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)

	type withdrawRequest struct {
		Amount int64  `json:"amount"`
		To     string `json:"to"`
	}

	var params withdrawRequest
	if err := c.BindJSON(&params); err != nil {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	if params.Amount <= 0 {
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	user, err := dao.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		log.Errorf("query user: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InvalidParams, c))
		return
	}

	if user.Reward < params.Amount {
		c.JSON(http.StatusOK, respErrorCode(errors.InsufficientBalance, c))
		return
	}

	request := &model.Withdraw{
		Username:  username,
		ToAddress: params.To,
		Amount:    params.Amount,
		Status:    0,
		CreatedAt: time.Now(),
	}

	if err = dao.AddWithdrawRequest(c.Request.Context(), request); err != nil {
		log.Errorf("add withdraw request: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(nil))
}

func GetWithdrawListHandler(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	username := claims[identityKey].(string)

	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	page, _ := strconv.Atoi(c.Query("page"))
	order := c.Query("order")
	orderField := c.Query("order_field")
	option := dao.QueryOption{
		Page:       page,
		PageSize:   pageSize,
		Order:      order,
		OrderField: orderField,
	}

	total, withdrawList, err := dao.GetWithdrawRecordList(c.Request.Context(), username, option)
	if err != nil {
		log.Errorf("get withdraw list: %v", err)
		c.JSON(http.StatusOK, respErrorCode(errors.InternalServer, c))
		return
	}

	c.JSON(http.StatusOK, respJSON(JsonObject{
		"list":  withdrawList,
		"total": total,
	}))
}
