package main

import (
	"gorm.io/gorm"
)

func UserDataByID(userID int64) (BlubUsers, error) {
	blubUserData := BlubUsers{UserId: userID}
	result := DB.Omit("referral_id", "is_sign_up", "sign_up_rewards").First(&blubUserData)
	if result.Error != nil {
		return BlubUsers{}, result.Error
	}

	return blubUserData, nil
}

func UserExistsByUserID(userID int64) bool {
	var user BlubUsers
	result := DB.Where("user_id = ?", userID).Limit(1).Find(&user)

	if result.RowsAffected > 0 {
		return true
	} else {
		return false
	}
}

func TaskDataByID(taskID int64, userID int64) (BlubTasks, error) {
	var task BlubTasks
	err := DB.Table("blub_tasks").
		Select("blub_tasks.*, CASE WHEN blub_claimed_tasks.task_id IS NOT NULL THEN 'claimed' ELSE 'available' END AS status").
		Joins("LEFT JOIN blub_claimed_tasks ON blub_claimed_tasks.task_id = blub_tasks.task_id AND blub_claimed_tasks.user_id = ?", userID).
		Where("blub_tasks.task_id = ?", taskID).
		Take(&task).Error

	if err != nil {
		return BlubTasks{}, err
	}

	return task, nil
}

func UserReferralCount(userID int64) (int64, error) {
	var count int64

	err := DB.Model(&BlubUsersReferrals{}).
		Where("referral_id = ?", userID).
		Count(&count).Error

	if err != nil {
		return 0, err
	}

	return count, nil
}

func InsertClaimedTask(userID, taskID int64) error {
	claimedTask := BlubClaimedTasks{UserId: userID, TaskId: taskID}
	result := DB.Create(claimedTask)
	return result.Error
}

func UpdateUserBalance(userID int64, amount float64, field string) error {
	err := DB.Model(&BlubUsers{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"balance": gorm.Expr("balance + ?", amount),
			field:     gorm.Expr(field+" + ?", amount),
		}).Error

	return err
}

func AllLevels() (Stats, error) {
	var stats Stats

	var users []Rank
	err := DB.Raw(`
		SELECT 
			first_name, 
			balance,
			ROW_NUMBER() OVER (ORDER BY balance DESC) AS placement 
		FROM blub_users 
		ORDER BY balance DESC LIMIT 50
	`).Scan(&users).Error
	if err != nil {
		return Stats{}, err
	}

	var count int64
	err = DB.Model(&BlubUsers{}).Count(&count).Error
	if err != nil {
		return Stats{}, err
	}

	stats.All = count
	stats.UserRankings = users

	return stats, nil
}

func UserRank(userID int64) (Rank, error) {
	var userRank Rank
	err := DB.Raw(`
	SELECT u.first_name, u.balance,
			(SELECT COUNT(*) FROM blub_users WHERE Balance > (SELECT Balance FROM blub_users WHERE user_id = ?)) + 1 as placement
		FROM blub_users u
		WHERE u.user_id = ?; 
	`, userID, userID).Scan(&userRank).Error

	return userRank, err
}

// func WithdrawExistByUserId(userID int64) bool {
// 	var count int64

// 	DB.Model(&BlubUsersWithdrawProcessed{}).
// 		Where("user_id = ?", userID).
// 		Or(DB.Model(&BlubUsersWithdrawPending{}).
// 			Where("user_id = ?", userID)).
// 		Count(&count)

// 	if count == 0 {
// 		return false
// 	} else {
// 		return true
// 	}
// }
