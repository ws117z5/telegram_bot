//go:build darwin || windows

package telegram_game

import (
	"bufio"
	"cmp"
	"context"
	"fmt"
	"log"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	. "github.com/ws117z5/telegram_bot/functions"
)

const (
	VOTE_YES = iota
	VOTE_NO
	VOTE_NONE
)

const Admin = "adventurer_v"
const botToken = "7113196065:AAEenTOKBuC1FnTrw5K8koozuaNlKe2UKdY"

type State struct {
	active  bool
	started bool

	messageId int
	users     []string
	votes     map[string]byte
	voteCount []int

	user_statistics map[string][3]int

	endTime              time.Time
	startTime            time.Time
	cancelSubroutineFunc context.CancelFunc
}

func NewState() *State {
	s := new(State)
	s.voteCount = make([]int, 3)
	s.votes = make(map[string]byte)
	s.user_statistics = make(map[string][3]int)

	usersCount := len(s.users)

	lines, err := ReadLines("users")
	if err != nil {
		log.Fatalf("readLines: %s", err)
	}

	//update user stats
	for _, line := range lines {
		data := strings.Split(line, " ")
		if len(data) == 4 {
			name := data[0]
			s.users = append(s.users, name)
			s.user_statistics[name] = [3]int{0, 0, 0}

			//worst mechanics ever thanks golang
			var tmp = s.user_statistics[name]

			var j int
			for i := 1; i <= 3; i++ {
				_, err := fmt.Sscan(data[i], &j)
				if err == nil {
					tmp[i-1] = j
				}
			}

			s.user_statistics[name] = tmp
		}

	}

	_, s.endTime = MoscowTime(23)

	fmt.Println("End Time:", s.endTime)

	s.users = Map(lines, MapUsernames)

	for _, u := range s.users {
		s.votes[u] = VOTE_NONE
	}

	s.voteCount[VOTE_NONE] = usersCount
	s.voteCount[VOTE_NO] = 0
	s.voteCount[VOTE_YES] = 0

	s.started = true
	s.active = false

	return s
}

func (s State) WriteStats() error {

	file, err := os.Open("users")
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)

	for _, user := range s.users {
		stats := s.user_statistics[user]
		fmt.Fprintf(w, "%s %d %d %d", user, stats[0], stats[1], stats[2])
	}
	return w.Flush()
}

func (s State) setUserVote(username string, option int) {
	tmp := s.user_statistics[username]
	if option == VOTE_YES {
		s.votes[username] = VOTE_YES
		s.voteCount[VOTE_NONE]--
		s.voteCount[VOTE_YES]++

		tmp[VOTE_YES]++
		tmp[VOTE_NONE]--
		//state.user_statistics[username][VOTE_YES]++
	} else if option == VOTE_NO {
		s.votes[username] = VOTE_NO
		s.voteCount[VOTE_NONE]--
		s.voteCount[VOTE_NO]++

		tmp[VOTE_NO]++
		tmp[VOTE_NONE]--
	} else {
		//if vote was withdrawn
		current_vote := s.votes[username]
		if current_vote == VOTE_YES {
			tmp[VOTE_YES]--
			s.voteCount[VOTE_YES]--
		} else {
			tmp[VOTE_NO]--
			s.voteCount[VOTE_NO]--
		}

		s.votes[username] = VOTE_NONE
		s.voteCount[VOTE_NONE]++
		tmp[VOTE_NONE]++
	}

	s.user_statistics[username] = tmp
}

func (s State) reset() {
	for _, u := range s.users {
		s.votes[u] = VOTE_NONE
	}

	usersCount := len(s.users)

	s.voteCount[VOTE_NONE] = usersCount
	s.voteCount[VOTE_NO] = 0
	s.voteCount[VOTE_YES] = 0
}

func (s State) Init(message *telego.Message) {
	s.reset()
	s.active = true
	s.messageId = message.MessageID
	_, s.startTime = MoscowTime()
}

func (s State) TimeFromStart(t time.Time) int {

	_, currentTime := MoscowTime()
	diff := s.startTime.Sub(currentTime)

	return int(diff.Seconds())
}

func (s State) getIgnored() []string {
	ret := []string{}
	for _, u := range s.users {
		if s.votes[u] == VOTE_NONE {
			ret = append(ret, u)
		}
	}

	return ret
}

func (s State) getVotedYes() []string {
	ret := []string{}
	for _, u := range s.users {
		if s.votes[u] == VOTE_YES {
			ret = append(ret, u)
		}
	}

	return ret
}

func (s State) getVotedNo() []string {
	ret := []string{}
	for _, u := range s.users {
		if s.votes[u] == VOTE_NO {
			ret = append(ret, u)
		}
	}

	return ret
}

// Not sure about this args
func MoscowTime(args ...int) (*time.Location, time.Time) {
	//init the loc
	loc, _ := time.LoadLocation("Europe/Moscow")
	moscowTime := time.Now().In(loc)

	//set timezone,
	if len(args) > 0 {
		year, month, day := moscowTime.Date()
		return loc, time.Date(year, month, day, args[0], 0, 0, 0, loc)
	}

	return loc, time.Now().In(loc)
}

func (s State) LaunchTimeObserver(bot *telego.Bot, chatID telego.ChatID) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, now := MoscowTime()
				hourBeforeEnd := s.endTime.Add(-1 * time.Hour)

				//diff := now.Sub(s.endTime)

				if hourBeforeEnd.Before(now) {
					bot.SendMessage(ctx,
						tu.Message(
							chatID,
							strings.Join(s.users, " ")+"\n Остался час",
						),
					)
				}

				if s.endTime.Before(now) {
					return
				}
			}

			time.Sleep(time.Second * 5)
		}
	}()

	s.cancelSubroutineFunc = cancel
}

type UserStats struct {
	name string
	yes  int
	no   int
	none int
}

func (s State) PrintStats(ctx context.Context, bot *telego.Bot, chatID telego.ChatID) {

	user_statisticsIdx := slices.Collect(maps.Values(s.user_statistics))

	lenCmp := func(a, b [3]int) int {
		return cmp.Or(
			cmp.Compare(a[VOTE_YES], b[VOTE_YES]),
			cmp.Compare(a[VOTE_NO], b[VOTE_NO]),
			cmp.Compare(a[VOTE_NONE], b[VOTE_NONE]),
		)
	}

	slices.SortFunc(user_statisticsIdx, lenCmp)

	bot.SendMessage(ctx,
		tu.Message(
			chatID,
			"Готовы играть: "+fmt.Sprintf("%v", s.voteCount[VOTE_YES])+"\n"+
				"Геи: "+fmt.Sprintf("%v", s.voteCount[VOTE_NO])+"\n"+
				"Курят бамбук: "+fmt.Sprintf("%v", s.voteCount[VOTE_NONE]),
		),
	)
	fmt.Println(user_statisticsIdx)

	//ask about the problem

	//intoduce yourself

	//math + programming

	//higher standards
	//story of mkrf when you insisted on moving to a descriptive architecture
	//in general work after hours to deliver better solutions
	//are right a lot
	//making of an api for the media forensic
	//adam lvl7
	//bar raiser

	//logical maintainable coding !!!

	//daniel lvl6
	//deliver results
	//that time when while doing a large samba scan the program started to fail,
	//ownership
	//
	//problem solving

	//chip carry //lvl6 coding data str
	//bias for action
	//going to insurance company to negotiate with infosec

	//james
	//dive deep
	//debugging of a php interpreter
	//earn trust
	//

	//system design
	//load balanser
	//sharding

	//bar rasing
	//complex projects
	//expedited developement
	//updated tech
}

// TODO redo the logic
func runBot() {

	// Get Bot token from environment variables
	//botToken := os.Getenv("7113196065:AAEenTOKBuC1FnTrw5K8koozuaNlKe2UKdY")

	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	state := NewState()

	ctx, _ := context.WithCancel(context.Background())
	//inited := false

	//
	//fmt.Println(state.users)

	// Get updates channel
	updates, _ := bot.UpdatesViaLongPolling(ctx, nil)

	bh, _ := th.NewBotHandler(bot, updates)

	// Stop handling updates
	defer func() { _ = bh.Stop() }()

	// Loop through all updates when they came
	for update := range updates {

		//If we have an active vote we want to register it inside state
		if state.active && update.PollAnswer != nil {
			username := update.PollAnswer.User.Username

			//Update the vote count
			state.setUserVote(username, update.PollAnswer.OptionIDs[0])
		}

		// Check if update contains a message
		if update.Message != nil {

			// Get chat ID and username from the message
			chatID := tu.ID(update.Message.Chat.ID)
			username := update.Message.From.Username

			messageParams := strings.Split(update.Message.Text, " ")

			if messageParams[0] == "/help" {

			}
			//show stats
			if messageParams[0] == "/stats" {
				state.PrintStats(ctx, bot, chatID)
			}

			if state.active {
				for _, word := range messageParams {
					if word == "+" {
						state.setUserVote(username, VOTE_YES)
					}

					if word == "-" {
						state.setUserVote(username, VOTE_NO)
					}
				}
			}

			if messageParams[0] == "/setendtime" && username == "adventurer_v" {

			}

			if messageParams[0] == "/start" && username == "adventurer_v" {

				//Init state variables
				state.Init(update.Message)

				//Mention everyone in the first message
				bot.SendMessage(ctx,
					tu.Message(
						chatID,
						strings.Join(state.users, " "),
					),
				)

				//Post a poll for gaming
				bot.SendPoll(ctx,
					&telego.SendPollParams{
						ChatID:      chatID,
						Question:    "Сыграем?",
						Options:     []telego.InputPollOption{tu.PollOption("Да"), tu.PollOption("Нет, Я Гей")},
						IsAnonymous: &[]bool{false}[0],
					},
				)
			}

			if messageParams[0] == "/stop" && username == "adventurer_v" {

				//Init state variables
				state.reset()
				state.active = false
				state.messageId = 0

				//write stats
				// for user, stats := range state.user_statistics {
				// 	stats[state.votes[user]]++
				// }
			}
		}
	}

	defer Exit(*state)
}

func Exit(s State) {
	if s.active {
		s.WriteStats()
	}
	fmt.Println("Exiting")
	os.Exit(0)
}
