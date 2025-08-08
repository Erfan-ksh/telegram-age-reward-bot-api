package main

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"time"
	"runtime"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	initdata "github.com/telegram-mini-apps/init-data-golang"
)

func FolderExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func CalcUserMaxCountByLevel(level int64) int64 {
	return 1000 + (level-1)*500
}

func CalcLevelUpPrice(level int64) int64 {
	return int64(2000 * math.Pow(2, float64(level-1)))
}

func RandomMult() int64 {
	numbers := []int{2, 4, 6}
	randomIndex := rand.Intn(len(numbers))
	return int64(numbers[randomIndex])
}

func CheckUserMembershipInTelegramChat(userID, channelID int64, channelUserName string) (bool, error) {
	chatMember, err := BOT.GetChatMember(tgbotapi.GetChatMemberConfig{ChatConfigWithUser: tgbotapi.ChatConfigWithUser{ChatID: channelID, SuperGroupUsername: channelUserName, UserID: userID}})
	if err != nil {
		return false, err
	}

	statuses := []string{"restricted", "left", "kicked"}

	var isMember bool
	if Contains(statuses, chatMember.Status) {
		isMember = false
	} else {
		isMember = true
	}

	return isMember, nil
}

func Contains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}

func CalcRankLevel(totalTaps int64) int64 {
	if totalTaps < 1600 {
		return 0
	}
	level := (totalTaps - 1600) / 400
	return level
}

func AreSlicesEqual(a, b []int64) bool {
	// If lengths are not equal, the slices cannot be the same
	if len(a) != len(b) {
		return false
	}

	// Sort both slices
	sort.Slice(a, func(i, j int) bool { return a[i] < a[j] })
	sort.Slice(b, func(i, j int) bool { return b[i] < b[j] })

	// Compare sorted slices element by element
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func SubSlice(s1 []int64, s2 []int64) bool {
	if len(s1) > len(s2) {
		return false
	}
	for _, e := range s1 {
		if !Contains(s2, e) {
			return false
		}
	}
	return true
}

func TruncateFloat(val float64, precision int) float64 {
	factor := math.Pow(10, float64(precision))
	return math.Trunc(val*factor) / factor
}

// Function to generate a random value within a specified range and probabilities
func GenerateRandomValue(min, max int, rng *rand.Rand) int {
	// Define ranges and probabilities
	ranges := []struct {
		min, max int
		prob     float64
	}{
		{min, min + 100000, 0.40},          // 5% chance for very low numbers
		{min + 100000, min + 500000, 0.40}, // 45% chance for low numbers
		{min + 500000, max - 100000, 0.15}, // 45% chance for moderate numbers
		{max - 100000, max, 0.05},          // 5% chance for very high numbers
	}

	// Define a cumulative probability threshold
	randVal := rng.Float64() // Random value between 0 and 1
	cumulativeProb := 0.0

	// Determine which range to use based on the random value
	for _, r := range ranges {
		cumulativeProb += r.prob
		if randVal < cumulativeProb {
			// Generate a random value within the selected range
			return generateValueInRange(r.min, r.max, rng)
		}
	}

	// Fallback if no range matched (should not happen)
	return min
}

// Helper function to generate a random value within a specific range
func generateValueInRange(min, max int, rng *rand.Rand) int {
	step := 10000 // Define step size for rounding values

	// Ensure the range is valid
	if max <= min || step <= 0 {
		return min
	}

	rangeSize := (max - min) / step
	if rangeSize <= 0 {
		return min // Return min if range size is too small
	}

	// Generate a random value within a valid range
	numSteps := rangeSize + 1
	if numSteps <= 0 {
		numSteps = 1 // Ensure that there is at least one step
	}

	return min + rng.Intn(numSteps)*step
}

func DownloadImage(url string, filepath string) error {
	// Make the HTTP GET request
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// Check if the request was successful
	if response.StatusCode != http.StatusOK {
		ErrorLogger.Printf("cacheTopFifty, Error: %v\n", err)
		return fmt.Errorf("error")
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy the response body to the file
	_, err = io.Copy(out, response.Body)
	return err
}

func OnError(r *http.Request, err string) {
	_, _, line, _ := runtime.Caller(1)
	userData, ok := r.Context().Value("userdata").(initdata.InitData)
	if !ok {
		ErrorLogger.Println("could not get userData inside OnError")
	}
	ErrorLogger.Println("error on line: ", line,userData.User.ID, r.Method, r.URL.String(), err)
}

func UserInitDataFromContext(r *http.Request) (initdata.InitData, bool) {
	userData, ok := r.Context().Value("userdata").(initdata.InitData)
	if !ok {
		return initdata.InitData{}, ok
	}

	return userData, true
}

func Round(val float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return float64(int(val*pow+0.5)) / pow
}

func GetAccountAge(key int64) int64 {
	var keys []int64
	for k := range ages {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	if key < keys[0] {
		return ages[keys[0]]
	}

	if key > keys[len(keys)-1] {
		return ages[keys[len(keys)-1]]
	}

	for i := len(keys) - 1; i >= 0; i-- {
		if key >= keys[i] {
			return ages[keys[i]]
		}
	}

	return 0
}

func CalcYearsUntilNow(uni int64) int {
	unixMilli := uni // Example timestamp (milliseconds)

	// Convert the Unix timestamp (milliseconds) to time.Time
	date := time.Unix(0, unixMilli*int64(time.Millisecond))

	// Get the current time
	now := time.Now()

	// Calculate the difference in years
	years := now.Year() - date.Year()

	// Adjust the result if the current date is before the anniversary date this year
	if now.YearDay() < date.YearDay() {
		years-- // Instead of setting years to 1, we decrement by 1
	}

	return years
}

var ages = map[int64]int64{
	2768409:    1383264000000,
	7679610:    1388448000000,
	11538514:   1391212000000,
	15835244:   1392940000000,
	23646077:   1393459000000,
	38015510:   1393632000000,
	44634663:   1399334000000,
	46145305:   1400198000000,
	54845238:   1411257000000,
	63263518:   1414454000000,
	101260938:  1425600000000,
	101323197:  1426204000000,
	103258382:  1432771000000,
	103151531:  1433376000000,
	109393468:  1439078000000,
	111220210:  1429574000000,
	116812045:  1437696000000,
	122600695:  1437782000000,
	124872445:  1439856000000,
	125828524:  1444003000000,
	130029930:  1441324000000,
	133909606:  1444176000000,
	143445125:  1448928000000,
	145221100:  1486702950000,
	152079341:  1453420000000,
	157242073:  1446768000000,
	171295414:  1457481000000,
	181783990:  1460246000000,
	1974255900: 1634000000000,
	222021233:  1465344000000,
	225034354:  1466208000000,
	278941742:  1473465000000,
	285253072:  1476835000000,
	294851037:  1479600000000,
	297621225:  1481846000000,
	328594461:  1482969000000,
	337808429:  1487707000000,
	341546272:  1487782000000,
	352940995:  1487894000000,
	369669043:  1490918000000,
	400169472:  1501459000000,
	805158066:  1563208000000,
}
