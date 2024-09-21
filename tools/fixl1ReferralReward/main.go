package main

import (
	"context"
	"fmt"
	"github.com/gnasnik/titan-explorer/config"
	"github.com/gnasnik/titan-explorer/core/dao"
	"github.com/gnasnik/titan-explorer/core/generated/model"
	logging "github.com/ipfs/go-log/v2"
	errs "github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/tealeg/xlsx/v3"
)

var log = logging.Logger("cleaning")

func main() {
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("reading config file: %v\n", err)
	}

	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("unmarshaling config file: %v\n", err)
	}

	if err := dao.Init(&cfg); err != nil {
		log.Fatalf("initital: %v\n", err)
	}

	ctx := context.Background()

	//  生成excel表
	err := l1UserRewardDetails(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return

	//err := updateL1UserRewards(ctx)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//return

	// 查出 l1 的所有用户
	userIds, err := GetL1RefUserId(ctx)
	if err != nil {
		log.Fatal(err)
	}
	//userIds := parentUserIds

	kolConfig, _, err := dao.GetKolLevelConfig(ctx, dao.QueryOption{})
	if err != nil {
		log.Fatal(err)
	}

	kolConfigMap := make(map[int]*model.KOLLevelConfig)
	for _, k := range kolConfig {
		kolConfigMap[k.Level] = k
	}

	referralReward := make(map[string]float64)
	var updateDetails []*model.UserRewardDetail

	// 按当前的KOL等级， 更新users表的 referral_reward
	for _, userID := range userIds {
		kolInfo, err := dao.GetKOLByUserId(ctx, userID)
		if err != nil {
			fmt.Println(err)
		}

		if kolInfo == nil {
			kolInfo = &model.KOL{UserId: userID, Level: 0}
		}

		currentLevel := kolConfigMap[kolInfo.Level]

		urd, err := dao.GetUserRewardDetailsByUserID(ctx, userID)
		if err != nil {
			fmt.Println(err)
		}

		for _, ur := range urd {
			rewards, err := GetUserDeviceReward(ctx, ur.FromUserId)
			if err != nil {
				log.Fatal(err)
			}

			rw := rewards.L2Reward * currentLevel.CommissionPercent / 100
			if ur.Relationship == 2 {
				rw = rewards.L2Reward * currentLevel.ParentCommissionPercent / 100
			}

			updateDetails = append(updateDetails, &model.UserRewardDetail{
				UserId:       userID,
				FromUserId:   ur.FromUserId,
				Reward:       rw,
				Relationship: ur.Relationship,
			})

			referralReward[userID] += rw
		}

	}

	//for k, v := range referralReward {
	//	fmt.Println("==>", k, v)
	//}

	// 更新 users 表, 邀请人的 referral_reward
	err = updateReferrerReward(ctx, referralReward)
	if err != nil {
		log.Fatal(err)
	}

	// 更新邀请奖励详情
	err = updateUserRewardDetails(ctx, updateDetails)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("finished")
}

func updateL1UserRewards(ctx context.Context) error {
	users, err := GetL1UserId(ctx)
	if err != nil {
		return err
	}

	var updateReward []*model.UserReward
	for _, userID := range users {
		parentUserReward, err := GetUserDeviceReward(ctx, userID)
		if err != nil {
			log.Fatal(err)
		}

		updateReward = append(updateReward, &model.UserReward{
			UserId:   userID,
			L2Reward: parentUserReward.L2Reward,
			L1Reward: parentUserReward.L1Reward,
		})
	}

	//for _, user := range updateReward {
	//	fmt.Println("=>", user.UserId, user.L1Reward+user.L2Reward)
	//}

	err = updateUserRewards(ctx, updateReward)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func GetL1RefUserId(ctx context.Context) ([]string, error) {

	var out []string
	err := dao.DB.SelectContext(ctx, &out, "select DISTINCT(referrer_user_id) from users where username in (select DISTINCT(user_id) from device_info where node_type = 2) and referrer_user_id <> ''")
	if err != nil {
		return nil, err
	}

	return out, nil
}

func GetL1UserId(ctx context.Context) ([]string, error) {

	var out []string
	err := dao.DB.SelectContext(ctx, &out, "select DISTINCT(user_id) from device_info where node_type = 2 and user_id <> ''")
	if err != nil {
		return nil, err
	}

	return out, nil
}

func GetUserDeviceReward(ctx context.Context, userId string) (*model.UserReward, error) {

	query := `
		select ifnull(user_id, '') as user_id,
	  ifnull(sum(if(node_type = 2, cumulative_profit, 0)),0) as l1_reward,
      ifnull(sum(if(node_type = 1, cumulative_profit, 0)),0) as l2_reward,
      count(device_id) as device_count
		from device_info  where user_id = ?`

	var out model.UserReward
	err := dao.DB.GetContext(ctx, &out, query, userId)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

const batchSize = 1000

func updateUserRewards(ctx context.Context, userRewards []*model.UserReward) error {
	var todos []*model.User

	for i, u := range userRewards {
		todos = append(todos, &model.User{
			Username: u.UserId,
			Reward:   u.L2Reward + u.L1Reward,
		})

		// Perform bulk insert when todos reaches a multiple of batchSize or at the end of updateUserRewards
		if len(todos)%batchSize == 0 || i == len(userRewards)-1 {
			if err := BulkUpdateUserReward(ctx, todos); err != nil {
				return errs.Wrap(err, "BulkUpdateUserReward")
			}
			todos = todos[:0] // Reset todos slice without reallocating memory
		}
	}

	return nil
}

func BulkUpdateUserReward(ctx context.Context, users []*model.User) error {
	query := `INSERT INTO users (username, reward,  updated_at) 
	VALUES (:username, :reward, :updated_at) 
	ON DUPLICATE KEY UPDATE reward = VALUES(reward), updated_at = now()`
	_, err := dao.DB.NamedExecContext(ctx, query, users)
	return err
}

func updateReferrerReward(ctx context.Context, referralReward map[string]float64) error {
	var todos []*model.User
	var i int

	for userId, rw := range referralReward {
		todos = append(todos, &model.User{Username: userId, ReferralReward: rw})

		i++
		if len(todos)%batchSize == 0 || i == len(referralReward) {
			if err := BulkUpdateUserReferralReward(ctx, todos); err != nil {
				return errs.Wrap(err, "BulkUpdateUserReferralReward")
			}
			todos = todos[:0]
		}

	}

	return nil
}

func BulkUpdateUserReferralReward(ctx context.Context, users []*model.User) error {
	query := `INSERT INTO users (username, referral_reward, updated_at) 
	VALUES (:username, :referral_reward, :updated_at) ON DUPLICATE KEY UPDATE referral_reward = VALUES(referral_reward) , updated_at = now()`
	_, err := dao.DB.NamedExecContext(ctx, query, users)
	return err
}

func BulkUpdateUserRewardDetails(ctx context.Context, logs []*model.UserRewardDetail) error {
	query := `INSERT INTO user_reward_detail (user_id, from_user_id, reward, relationship, created_at, updated_at) 
	VALUES (:user_id, :from_user_id, :reward, :relationship, now(), now()) ON DUPLICATE KEY UPDATE reward = VALUES(reward), updated_at = now()`
	_, err := dao.DB.NamedExecContext(ctx, query, logs)
	return err
}

func updateUserRewardDetails(ctx context.Context, updateDetails []*model.UserRewardDetail) error {
	var todos []*model.UserRewardDetail

	for i, u := range updateDetails {
		todos = append(todos, u)

		// Perform bulk insert when todos reaches a multiple of batchSize or at the end of updateUserRewards
		if len(todos)%batchSize == 0 || i == len(updateDetails)-1 {
			if err := BulkUpdateUserRewardDetails(ctx, todos); err != nil {
				return errs.Wrap(err, "BulkUpdateUserRewardDetails")
			}
			todos = todos[:0] // Reset todos slice without reallocating memory
		}
	}

	return nil
}

func l1UserRewardDetails(ctx context.Context) error {
	query1 := `
		select username, referral_reward from users where username in (
		select referrer_user_id from users where username in (
		select  user_id from device_info where node_type = 2 and cumulative_profit  > 0 and user_id <> ''
		) and referrer_user_id <> ''
		) order by username
	`

	var users []*model.User
	if err := dao.DB.SelectContext(ctx, &users, query1); err != nil {
		return err
	}

	//query1 := `select username, referral_reward from users where username in (?)  order by username`
	//queryIn, args, _ := sqlx.In(query1, parentUserIds)
	//
	//var users []*model.User
	//if err := dao.DB.SelectContext(ctx, &users, queryIn, args...); err != nil {
	//	return err
	//}

	file := xlsx.NewFile()

	sh, err := file.AddSheet("Sheet1")
	if err != nil {
		log.Fatal(err)
	}

	headerRow := sh.AddRow()
	headerRow.AddCell().SetValue("user_id")
	headerRow.AddCell().SetValue("referral_reward")
	headerRow.AddCell().SetValue("kol_level")
	headerRow.AddCell().SetValue("children_l1_reward")
	headerRow.AddCell().SetValue("children_l2_reward")
	headerRow.AddCell().SetValue("grandchild_l1_reward")
	headerRow.AddCell().SetValue("grandchild_l2_reward")

	query2 := `select ifnull(user_id, '') as user_id,
      ifnull(sum(if(node_type = 2, cumulative_profit,0)),0) as l1_reward,
     ifnull(sum(if(node_type = 1, cumulative_profit,0)),0) as l2_reward
		from device_info  where user_id in (select from_user_id from user_reward_detail where user_id = ? and relationship = ?)`

	for _, user := range users {
		row := sh.AddRow()
		row.AddCell().SetValue(user.Username)
		row.AddCell().SetValue(user.ReferralReward)

		kolInfo, err := dao.GetKOLByUserId(ctx, user.Username)
		if err != nil {
			fmt.Println(err)
		}

		if kolInfo != nil {
			row.AddCell().SetValue(kolInfo.Level)
		}

		var userReward model.UserReward

		err = dao.DB.GetContext(ctx, &userReward, query2, user.Username, 1)
		if err != nil {
			fmt.Println(err)
		}

		row.AddCell().SetValue(userReward.L1Reward)
		row.AddCell().SetValue(userReward.L2Reward)

		var userReward2 model.UserReward
		err = dao.DB.GetContext(ctx, &userReward2, query2, user.Username, 2)
		if err != nil {
			fmt.Println(err)
		}

		row.AddCell().SetValue(userReward2.L1Reward)
		row.AddCell().SetValue(userReward2.L2Reward)
	}

	err = file.Save("./l1_user_referral_reward.xlsx")
	if err != nil {
		fmt.Println(err)
	}

	return nil
}
