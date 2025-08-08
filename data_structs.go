package main

type BlubUsers struct {
	UserId                   int64          `json:"user_id" gorm:"primaryKey"`
	FirstName                string         `json:"first_name"`
	Balance                  float64        `json:"balance"`
	ProfitFromInvites        float64        `json:"profit_from_invites"`
	ProfitFromRewards        float64        `json:"profit_from_rewards"`
	ProfitFromTasks          float64        `json:"profit_from_tasks"`
	LastProfitClaimTimestamp int64          `json:"-"`
	ReferralId               *int64         `json:"-"`
	IsSignUp                 bool           `json:"is_sign_up,omitempty" gorm:"-"`
	SignUpRewards            map[string]any `json:"sign_up_rewards,omitempty" gorm:"-"`
}

type BlubTasks struct {
	TaskId          int64   `json:"task_id" gorm:"primaryKey"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	TaskType        string  `json:"task_type"`
	Reward          int64   `json:"reward"`
	Link            *string `json:"link,omitempty"`
	ChannelUsername *string `json:"channel_username,omitempty"`
	ChannelId       *int64  `json:"channel_id,omitempty"`
	ShouldInvite    *int64  `json:"should_invite,omitempty"`
	Status          string  `json:"status"`
	ImageName       string  `json:"image_name,omitempty"`
	IsPartnerTask   bool    `json:"is_partner_task"`
}

type BlubClaimedTasks struct {
	TaskId int64
	UserId int64
}

type BlubUsersReferrals struct {
	ReferralId int64 `json:"referral_id" gorm:"primaryKey"`
	UserId     int64 `json:"user_id"`
	JoinTime   int64 `json:"join_time"`
}

type Stats struct {
	All          int64  `json:"all"`
	UserRank     Rank   `json:"user_rank"`
	UserRankings []Rank `json:"user_rankings"`
}

type Rank struct {
	FirstName string  `json:"first_name"`
	Placement int64   `json:"placement"`
	Balance   float64 `json:"balance"`
}

type BlubUsersWithdraws struct {
	Id int64 `gorm:"primaryKey"`
	UserId int64 `json:"user_id"`
	Amount int64 `json:"amount"`
	Wallet string `json:"wallet"`
	Timestamp int64 `json:"timestamp"`
}

type RequestBody struct {
	Wallet string `json:"wallet"`
}