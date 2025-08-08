package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"strconv"

	"github.com/goccy/go-json"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	// tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	initdata "github.com/telegram-mini-apps/init-data-golang"
	"golang.org/x/time/rate"
)

var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonURL("Launch AXO 💕", "https://t.me/<>/<>"),
		tgbotapi.NewInlineKeyboardButtonURL("Community 💎", "https://t.me/<>"),
	),
)

func InitUserRoutes() {
	initRateLimitCleanUp()

	userRouter := Router.PathPrefix("/user").Subrouter()

	userRouter.Use(AccessControlMiddleware)
	userRouter.Use(limitConcurrentRequests)
	userRouter.Use(authMiddleWare)
	userRouter.Use(rateLimitMiddleWare)
	userRouter.Use(logging)

	// /user/data
	userRouter.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		userData, ok := UserInitDataFromContext(r)
		if !ok {
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		blubUserData, err := UserDataByID(userData.User.ID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Redirect(w, r, "/user/signup", http.StatusFound)
				return
			}
			OnError(r, err.Error())
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		err = json.NewEncoder(w).Encode(blubUserData)
		if err != nil {
			OnError(r, err.Error())
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
	})

	userRouter.HandleFunc("/signup", func(w http.ResponseWriter, r *http.Request) {
		userData, ok := UserInitDataFromContext(r)
		if !ok {
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		userExists := UserExistsByUserID(userData.User.ID)
		if userExists {
			http.Redirect(w, r, "/user/data", http.StatusFound)
			return
		}

		rand.NewSource(time.Now().UnixNano())

		totalAxo := 0
		premiumReward := 0
		isPremium := userData.User.IsPremium
		if isPremium {
			premiumReward = 2 + rand.Intn(2)
			totalAxo += premiumReward
		}

		ageInUnixMilli := GetAccountAge(userData.User.ID)

		yearsReward := 0
		years := CalcYearsUntilNow(ageInUnixMilli)
		for i := 0; i < years; i++ {
			amount := 2 + rand.Intn(2)
			yearsReward += amount
			totalAxo += amount
		}

		newUser := BlubUsers{UserId: userData.User.ID, Balance: float64(totalAxo), FirstName: userData.User.FirstName}

		result := DB.Omit("referral_id", "is_sign_up", "sign_up_rewards").Create(&newUser)
		if result.Error != nil {
			OnError(r, result.Error.Error())
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		if userData.StartParam != "" {
			refID, err := strconv.ParseInt(userData.StartParam, 10, 64)
			if err != nil {
				OnError(r, err.Error())
				http.Error(w, "somethings went wrong!", http.StatusBadRequest)
				return
			}

			if refID != userData.User.ID {
				newRef := BlubUsersReferrals{UserId: userData.User.ID, ReferralId: refID, JoinTime: time.Now().UnixMilli()}
				result = DB.Create(&newRef)
				if result.Error != nil {
					OnError(r, result.Error.Error())
					http.Error(w, "somethings went wrong!", http.StatusBadRequest)
					return
				}

				err = UpdateUserBalance(refID, 1, "profit_from_invites")
				if err != nil {
					OnError(r, err.Error())
					http.Error(w, "something went wrong!", http.StatusInternalServerError)
					return
				}

			}
		}

		blubUserData, err := UserDataByID(userData.User.ID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Redirect(w, r, "/user/signup", http.StatusFound)
				return
			}
			OnError(r, err.Error())
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		blubUserData.IsSignUp = true

		signUpRewards := make(map[string]any)
		signUpRewards["premium_reward"] = premiumReward
		signUpRewards["age"] = years
		signUpRewards["age_reward"] = yearsReward

		blubUserData.SignUpRewards = signUpRewards

		// send message
		msg := tgbotapi.NewPhoto(userData.User.ID, tgbotapi.FilePath("bot_start.jpeg"))
		msg.Caption = `Dear` + userData.User.FirstName + `

Welcome to Community Bot 💕

Community is a Web3 Community built on TON Network with the aim to make meme building easier on TON Blockchain 

Launch the app and Start earning today 🎁`
		msg.ReplyMarkup = numericKeyboard
		_, err = BOT.Send(msg)
		if err != nil {
			ErrorLogger.Println(err)
		}

		err = json.NewEncoder(w).Encode(blubUserData)
		if err != nil {
			OnError(r, err.Error())
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
	})

	userRouter.HandleFunc("/profit/claim", func(w http.ResponseWriter, r *http.Request) {
		userData, ok := UserInitDataFromContext(r)
		if !ok {
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		var remaining int64
		var roundedValue float64
		err := DB.Transaction(func(tx *gorm.DB) error {
			var user BlubUsers

			if err := tx.Table("blub_users").
				Select("blub_users.*, blub_users_referrals.referral_id").
				Joins("LEFT JOIN blub_users_referrals ON blub_users_referrals.user_id = blub_users.user_id").
				Where("blub_users.user_id = ?", userData.User.ID).
				Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&user).Error; err != nil {
				return err
			}

			now := time.Now().UnixMilli()
			diff := now - user.LastProfitClaimTimestamp

			if diff < 1800000 {
				remaining = 1800000 - diff
				return fmt.Errorf("bad")
			}

			user.LastProfitClaimTimestamp = now

			rand.NewSource(time.Now().UnixNano())

			min := 0.2
			max := 0.5
			randomValue := min + rand.Float64()*(max-min)
			roundedValue = Round(randomValue, 2)

			user.Balance += roundedValue
			user.ProfitFromRewards += roundedValue

			amount := roundedValue * 0.20

			if user.ReferralId != nil {
				err := UpdateUserBalance(*user.ReferralId, amount, "profit_from_invites")
				if err != nil {
					return err
				}
			}

			return tx.Omit("referral_id", "is_sign_up", "sign_up_rewards").Save(&user).Error
		})

		if err != nil {
			if err.Error() == "bad" {
				http.Error(w, fmt.Sprintf("%d", remaining), http.StatusBadRequest)
				return
			}

			OnError(r, err.Error())
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		w.Write([]byte(fmt.Sprintf("%.2f", roundedValue)))
	})

	userRouter.HandleFunc("/withdraw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		userData, ok := UserInitDataFromContext(r)
		if !ok {
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		// doesWithdrawExist := WithdrawExistByUserId(userData.User.ID)

		// if doesWithdrawExist {
		// 	http.Error(w, "You have already made a withdrawal.", http.StatusBadRequest)
		// 	return
		// }

		var requestBody RequestBody
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&requestBody)
		if err != nil {
			OnError(r, err.Error())
			http.Error(w, "Unable to parse JSON", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var amount float64
		err = DB.Transaction(func(tx *gorm.DB) error {
			var user BlubUsers

			if err := tx.Where("user_id = ?", userData.User.ID).Clauses(clause.Locking{Strength: "UPDATE"}).First(&user).Error; err != nil {
				return err
			}

			if user.Balance < 150 {
				return fmt.Errorf("bad")
			}

			amount = math.Floor(user.Balance)

			user.Balance -= amount

			invoice := BlubUsersWithdraws{UserId: userData.User.ID, Amount: int64(amount), Timestamp: time.Now().Local().Unix(), Wallet: requestBody.Wallet}

			result := tx.Create(&invoice)
			if result.Error != nil {
				return result.Error
			}

			return tx.Omit("referral_id").Save(&user).Error
		})

		if err != nil {
			if err.Error() == "bad" {
				http.Error(w, "To make a withdrawal, please ensure you have a minimum balance of 150$ AXO.", http.StatusBadRequest)
				return
			}
			OnError(r, err.Error())
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		w.Write([]byte(fmt.Sprintf("Your withdrawal of %d$ coin is currently pending. We will process and deposit it shortly.", int64(amount))))
	})

	userRouter.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		userData, ok := UserInitDataFromContext(r)
		if !ok {
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		var tasks []BlubTasks
		err := DB.Table("blub_tasks").
			Select("blub_tasks.*, CASE WHEN blub_claimed_tasks.task_id IS NOT NULL THEN 'claimed' ELSE 'available' END AS status").
			Joins("LEFT JOIN blub_claimed_tasks ON blub_claimed_tasks.task_id = blub_tasks.task_id AND blub_claimed_tasks.user_id = ?", userData.User.ID).
			Scan(&tasks).Error

		if err != nil {
			OnError(r, err.Error())
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		err = json.NewEncoder(w).Encode(tasks)
		if err != nil {
			OnError(r, err.Error())
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
	})

	var topFifty Stats
	go cacheUserRankings(&topFifty)
	userRouter.HandleFunc("/friends", func(w http.ResponseWriter, r *http.Request) {
		userData, ok := UserInitDataFromContext(r)
		if !ok {
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		userRank, err := UserRank(userData.User.ID)
		if err != nil {
			OnError(r, err.Error())
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		newTopFifty := topFifty

		newTopFifty.UserRank = userRank

		err = json.NewEncoder(w).Encode(newTopFifty)
		if err != nil {
			OnError(r, err.Error())
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
	})

	userRouter.HandleFunc("/tasks/claim", func(w http.ResponseWriter, r *http.Request) {
		userData, ok := UserInitDataFromContext(r)
		if !ok {
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		taskID := r.URL.Query().Get("task-id")
		if taskID == "" {
			http.Error(w, "task-id can't be empty.", http.StatusBadRequest)
			return
		}

		numTaskID, err := strconv.ParseInt(taskID, 10, 64)
		if err != nil {
			OnError(r, err.Error())
			http.Error(w, "somethings went wrong!", http.StatusBadRequest)
			return
		}

		task, err := TaskDataByID(numTaskID, userData.User.ID)
		if err != nil {
			OnError(r, err.Error())
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "no task with this id", http.StatusBadRequest)
				return
			}
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}

		if task.Status == "claimed" {
			http.Error(w, "Reward already claimed", http.StatusBadRequest)
			return
		}

		userID := userData.User.ID

		switch task.TaskType {
		case "invite":
			{
				userRefCount, err := UserReferralCount(userID)
				if err != nil {
					ErrorLogger.Printf("/user/tasks/claim userID: %d, taskID: %d. Error: %v", userID, numTaskID, err)
					http.Error(w, "something went wrong", http.StatusInternalServerError)
					return
				}

				if userRefCount >= *task.ShouldInvite {
					err = InsertClaimedTask(userID, task.TaskId)
					if err != nil {
						ErrorLogger.Printf("/user/tasks/claim userID: %d, taskID: %d. could not insert claimed task. Error: %v", userID, numTaskID, err)
						http.Error(w, "something went wrong!", http.StatusInternalServerError)
						return
					}

					err = UpdateUserBalance(userID, float64(task.Reward), "profit_from_tasks")
					if err != nil {
						OnError(r, err.Error())
						http.Error(w, "something went wrong!", http.StatusInternalServerError)
						return
					}

					text := fmt.Sprintf("added %d coins to user", task.Reward)
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(text))
					return
				} else {
					w.WriteHeader(http.StatusBadRequest)
					text := fmt.Sprintf("you need %d more refs", *task.ShouldInvite-userRefCount)
					w.Write([]byte(text))
					return
				}
			}
		case "join":
			{

				isMember, err := CheckUserMembershipInTelegramChat(userID, *task.ChannelId, *task.ChannelUsername)
				if err != nil {
					OnError(r, err.Error())
					http.Error(w, "something went wrong", http.StatusInternalServerError)
					return
				}

				if !isMember {
					http.Error(w, "First, join the channel, and then you can receive your reward!", http.StatusBadRequest)
					return
				}

				err = InsertClaimedTask(userID, task.TaskId)
				if err != nil {
					OnError(r, err.Error())
					http.Error(w, "something went wrong!", http.StatusInternalServerError)
					return
				}

				err = UpdateUserBalance(userID, float64(task.Reward), "profit_from_tasks")
				if err != nil {
					OnError(r, err.Error())
					http.Error(w, "something went wrong!", http.StatusInternalServerError)
					return
				}

				text := fmt.Sprintf("Successfully added %d coins to the user's account!", task.Reward)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(text))
				return
			}
		case "link":
			{
				http.Error(w, "this kind of task can not be claimed", http.StatusBadRequest)
				return

			}
		case "claim":
			{
				err = InsertClaimedTask(userID, task.TaskId)
				if err != nil {
					OnError(r, err.Error())
					http.Error(w, "something went wrong!", http.StatusInternalServerError)
					return
				}

				err = UpdateUserBalance(userID, float64(task.Reward), "profit_from_tasks")
				if err != nil {
					OnError(r, err.Error())
					http.Error(w, "something went wrong!", http.StatusInternalServerError)
					return
				}

				text := fmt.Sprintf("Successfully added %d coins to the user's account!", task.Reward)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(text))
				return
			}
		}
	})
}

func authMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if len(authHeader) < 4 {
			http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
			return
		}

		tokenurl := authHeader[4:]
		token, err := url.QueryUnescape(tokenurl)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			ErrorLogger.Println("Error decoding string:", err)
			return
		}

		expIn := 999 * time.Hour

		err = initdata.Validate(tokenurl, BOT_TOKEN, expIn)
		if err != nil {
			http.Error(w, "unauth", http.StatusUnauthorized)
			return
		}

		data, err := initdata.Parse(token)
		if err != nil {
			OnError(r, err.Error())
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), "userdata", data)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func AccessControlMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		host := ""

		if origin == "http://localhost:5173" {
			host = "http://localhost:5173"
		}

		w.Header().Set("Access-Control-Allow-Origin", host)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Create a custom visitor struct which holds the rate limiter for each
// visitor and the last time that the visitor was seen.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Change the map to hold values of the type visitor.
var visitors = make(map[int64]*visitor)
var mu sync.Mutex

// Run a background goroutine to remove old entries from the visitors map.
func initRateLimitCleanUp() {
	go cleanupVisitors()
}

func getVisitor(userID int64) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[userID]
	if !exists {
		limiter := rate.NewLimiter(3, 7)
		// Include the current time when creating a new visitor.
		visitors[userID] = &visitor{limiter, time.Now()}
		return limiter
	}

	// Update the last seen time for the visitor.
	v.lastSeen = time.Now()
	return v.limiter
}

// Every minute check the map for visitors that haven't been seen for
// more than 3 minutes and delete the entries.
func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		mu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(visitors, ip)
			}
		}
		mu.Unlock()
	}
}

func rateLimitMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, ok := UserInitDataFromContext(r)
		if !ok {
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		limiter := getVisitor(userData.User.ID)
		if !limiter.Allow() {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, ok := r.Context().Value("userdata").(initdata.InitData)
		if !ok {
			ErrorLogger.Println("could not get userData inside logging")
		}
		start := time.Now()
		next.ServeHTTP(w, r)
		if time.Since(start) > 50*time.Millisecond {
			InfoLogger.Println(userData.User.ID, r.Method, r.URL.String(), time.Since(start))
		}
	})
}

func cacheUserRankings(topFifty *Stats) {
	topFifty1, err := AllLevels()
	if err != nil {
		ErrorLogger.Printf("cacheUserRankings Error: %v.\n", err)
	}
	*topFifty = topFifty1

	// rerun every hour
	for range time.Tick(time.Minute * 30) {
		topFifty1, err := AllLevels()
		if err != nil {
			ErrorLogger.Printf("cacheUserRankings Error: %v\n", err)
		}
		*topFifty = topFifty1
	}
}

const maxConcurrentRequests = 3000

var sem = make(chan struct{}, maxConcurrentRequests)

func limitConcurrentRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			next.ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, "Server is overloaded. Please try again later.")
		}
	})
}
