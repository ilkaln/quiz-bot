//go:build !solution

package telegram

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ilkaln/quiz-bot/internal/quiz"
)

// Bot реализует Telegram бота для квизов.
type Bot struct {
	client      Client
	engine      quiz.QuizEngine
	botUsername string // Username бота для формирования ссылок (например, "my_quiz_bot")
	// TODO: добавьте необходимые поля
	sessions     map[int64]*Session
	session_mute sync.RWMutex

	userRunID      map[int64]string
	userRunID_mute sync.RWMutex

	authorRunMsg      map[string]*Message
	authorRunMsg_mute sync.RWMutex
}

// NewBot создаёт нового бота.
// botUsername — username бота без @ (например, "my_quiz_bot").
// Используется для формирования ссылок: https://t.me/<botUsername>?start=join_<runID>
func NewBot(client Client, engine quiz.QuizEngine, botUsername string) *Bot {
	return &Bot{
		client:       client,
		engine:       engine,
		botUsername:  botUsername,
		sessions:     make(map[int64]*Session),
		userRunID:    make(map[int64]string),
		authorRunMsg: make(map[string]*Message),
	}
}

// Run запускает бота (long polling).
func (b *Bot) Run() error {

	offset := 0
	timeout := 30 // Секунды ожидания на сервере Telegram

	for {
		updates, err := b.client.GetUpdates(offset, timeout)
		if err != nil {
			continue
		}

		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}

			go func(u Update) {
				if err := b.HandleUpdate(u); err != nil {
					fmt.Printf("Error of handle update with ID %d: %v\n", u.UpdateID, err)
				}
			}(update)
		}
	}

}

// HandleUpdate обрабатывает одно обновление.
func (b *Bot) HandleUpdate(update Update) error {
	if update.Message != nil {
		return b.HandleMessage(update.Message)
	}

	if update.CallbackQuery != nil {
		return b.HandleCallbackQuery(update.CallbackQuery)
	}

	return nil
}

func (b *Bot) BuildQuizzesList(quizzes []*quiz.Quiz) string {
	var buffer strings.Builder

	counter := 0
	for _, q := range quizzes {
		if q != nil {
			if counter == 0 {
				buffer.Grow(512)
				buffer.WriteString(quizzesListTitle)
			}
			counter++
			buffer.WriteString(fmt.Sprintf(quizzesListDescription, counter, q.Title, q.ID, b.botUsername, q.ID))
		}
	}

	if counter == 0 {
		return emptyOwnerQuizzes
	}

	return buffer.String()
}

func (b *Bot) BuildQuizRunsList(quizzes []*quiz.Quiz) string {
	var buffer strings.Builder

	counter := 0
	for _, q := range quizzes {
		if q != nil {
			if counter == 0 {
				buffer.Grow(512)
				buffer.WriteString(quizRunsTitle)
			}
			counter++
			buffer.WriteString(fmt.Sprintf(quizRunsListTitle, counter, q.Title, q.ID))
			list, err := b.engine.GetListRuns(q.ID)
			if err != nil {
				buffer.WriteString(emptyQuizRunsList)
			}
			for i, qr := range list {
				if qr == nil {
					buffer.WriteString(fmt.Sprintf(underfindQuizRunDescription, counter, i+1))
				} else if qr.Status != quiz.RunStatusFinished {
					buffer.WriteString(fmt.Sprintf(quizRunDescription, counter, i+1, qr.ID))
				} else {
					buffer.WriteString(fmt.Sprintf(finishedQuizRunDescription, counter, i+1, qr.ID, b.botUsername, qr.ID))
				}
			}
		}
	}

	if counter == 0 {
		return emptyOwnerQuizzes
	}

	return buffer.String()
}

func (b *Bot) HandleMessage(message *Message) error {
	b.session_mute.RLock()
	if session, ok := b.sessions[message.Chat.ID]; ok {

		run, err := b.engine.GetRun(session.RunID)
		if err != nil {
			b.client.SendMessage(message.Chat.ID, underfindIDsession, nil)
			b.session_mute.RUnlock()
			return nil
		}

		if session.FieldIdx == len(run.QuizPtr.Settings.Registration) {
			b.client.SendMessage(message.Chat.ID, fmt.Sprintf(alreadyJoined, session.RunID), nil)
			b.session_mute.RUnlock()
			return nil
		}
		b.session_mute.RUnlock()

		value := strings.TrimSpace(message.Text)

		b.session_mute.Lock()
		session.Participant.RegData[run.QuizPtr.Settings.Registration[session.FieldIdx]] = value
		session.FieldIdx++
		b.session_mute.Unlock()

		b.session_mute.RLock()

		defer b.session_mute.RUnlock()

		if session.FieldIdx == len(run.QuizPtr.Settings.Registration) {

			defer delete(b.sessions, message.Chat.ID)

			if err := b.engine.JoinRun(context.Background(), session.RunID, session.Participant); err != nil {
				b.client.SendMessage(message.Chat.ID, fmt.Sprintf(errorJoined, session.RunID, err), nil)
				return nil
			}
			b.client.SendMessage(message.Chat.ID, fmt.Sprintf(successfulReg, run.QuizPtr.Title, session.RunID), nil)
			b.userRunID_mute.Lock()
			b.userRunID[message.Chat.ID] = session.RunID
			b.userRunID_mute.Unlock()
		} else {
			b.client.SendMessage(message.Chat.ID, fmt.Sprintf(regDataReq, run.QuizPtr.Settings.Registration[session.FieldIdx]), nil)
		}
		return nil
	}
	b.session_mute.RUnlock()

	if message.Document != nil {
		return b.HandleDocument(message)
	}

	if runID, ok := strings.CutPrefix(message.Text, "/start join_"); ok {
		runID = strings.TrimSpace(runID)

		if runID == "" {
			b.client.SendMessage(message.Chat.ID, emptyRunID, nil)
			return nil
		}

		run, err := b.engine.GetRun(runID)
		if err != nil {
			b.client.SendMessage(message.Chat.ID, underfindID, nil)
			return nil
		}

		if _, ok := run.Participants[message.From.ID]; ok {
			b.client.SendMessage(message.Chat.ID, fmt.Sprintf(alreadyJoined, runID), nil)
			return nil
		}

		participant := quiz.Participant{
			TelegramID: message.From.ID,
			EditMsgID:  -1,
			Username:   message.From.Username,
			FirstName:  message.From.FirstName,
			LastName:   message.From.LastName,
			RegData:    make(map[string]string),
			JoinedAt:   time.Now(),
		}

		if len(run.QuizPtr.Settings.Registration) > 0 {
			session := Session{
				RunID:       runID,
				Participant: &participant,
				FieldIdx:    0,
			}
			b.session_mute.Lock()
			b.sessions[message.Chat.ID] = &session
			b.session_mute.Unlock()
			b.client.SendMessage(message.Chat.ID, fmt.Sprintf(regDataReq, run.QuizPtr.Settings.Registration[0]), nil)
			return nil
		}
		if err := b.engine.JoinRun(context.Background(), runID, &participant); err != nil {
			b.client.SendMessage(message.Chat.ID, fmt.Sprintf(errorJoined, runID, err), nil)
			return nil
		}
		b.client.SendMessage(message.Chat.ID, fmt.Sprintf(successfulReg, run.QuizPtr.Title, runID), nil)
		b.userRunID_mute.Lock()
		b.userRunID[message.Chat.ID] = runID
		b.userRunID_mute.Unlock()
		return nil
	}

	if quizID, ok := strings.CutPrefix(message.Text, "/start get_quiz_"); ok {
		quizID = strings.TrimSpace(quizID)

		if quizID == "" {
			b.client.SendMessage(message.Chat.ID, emptyQuizID, nil)
			return nil
		}

		quiz, err := b.engine.GetQuiz(quizID)
		if err != nil {
			b.client.SendMessage(message.Chat.ID, underfindQuizID, nil)
			return nil
		}

		b.client.SendDocument(message.Chat.ID, fmt.Sprintf(quizJSONFileName, quiz.Title, quiz.ID), *quiz.BytesJSONPtr)
		return nil
	}

	if runID, ok := strings.CutPrefix(message.Text, "/start get_runres_"); ok {
		runID = strings.TrimSpace(runID)

		if runID == "" {
			b.client.SendMessage(message.Chat.ID, emptyRunResID, nil)
			return nil
		}

		run, err := b.engine.GetRun(runID)
		if err != nil {
			b.client.SendMessage(message.Chat.ID, underfindRunResID, nil)
			return nil
		}

		if run.ResultsCSVBytes == nil {
			file, err := b.engine.ExportCSV(runID)
			if err != nil {
				fmt.Printf("invalid export csv for runID: %s", runID)
				b.client.SendMessage(message.Chat.ID, fmt.Sprintf(errorExportCSV, err), nil)
				return nil
			}
			run.ResultsCSVBytes = &file
		}

		b.client.SendDocument(message.Chat.ID, fmt.Sprintf(resultsCSVDocName, run.ID, run.FinishedAt.Format("2006-01-02_15-04-05")), *run.ResultsCSVBytes)
		return nil
	}
	if text, ok := strings.CutPrefix(message.Text, "/start"); ok && text == "" {

		b.client.SendMessage(message.Chat.ID, welcomeMsg, nil)
		return nil
	}
	if text, ok := strings.CutPrefix(message.Text, "/myquizzes"); ok && text == "" {
		quizzes := b.engine.GetPersQuizzes(message.From.ID)
		text := b.BuildQuizzesList(quizzes)
		b.client.SendMessage(message.Chat.ID, text, nil)
		return nil
	}
	if text, ok := strings.CutPrefix(message.Text, "/myquizruns"); ok && text == "" {
		quizzes := b.engine.GetPersQuizzes(message.From.ID)
		text := b.BuildQuizRunsList(quizzes)
		b.client.SendMessage(message.Chat.ID, text, nil)
		return nil
	}
	b.client.SendMessage(message.Chat.ID, underfindMsg, nil)

	return nil
}

func (b *Bot) LobbyProc(quizObj *quiz.Quiz, chatID int64, relode bool) error {
	run, err := b.engine.StartRun(context.Background(), quizObj)
	if err != nil {
		b.client.SendMessage(chatID, fmt.Sprintf(errorStartLobby, err), nil)
		return err
	}

	link := fmt.Sprintf(linkPattern, b.botUsername, run.ID)

	optsStart := &SendOptions{
		ReplyMarkup: &InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{
				{
					{Text: "Начать квиз", CallbackData: "start_quiz_" + run.ID},
					{Text: "Отменить", CallbackData: "cancel_lobby_" + run.ID},
				},
			},
		},
	}

	optsStop := &SendOptions{
		ReplyMarkup: &InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{
				{
					{Text: "Отменить", CallbackData: "cancel_quiz_" + run.ID},
				},
			},
		},
	}

	lastCount := 0
	var sendedMsg *Message

	if relode {
		sendedMsg, err = b.client.SendMessage(chatID, fmt.Sprintf(successfulRestartRun, quizObj.Title, run.ID, link, lastCount), optsStart)
	} else {
		sendedMsg, err = b.client.SendMessage(chatID, fmt.Sprintf(successfulFileLoad, quizObj.Title, run.ID, link, lastCount), optsStart)
	}
	b.authorRunMsg_mute.Lock()
	b.authorRunMsg[run.ID] = sendedMsg
	b.authorRunMsg_mute.Unlock()

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-(*run.LobbyRun.Context).Done():
				run, err = b.engine.GetRun(run.ID)
				if err == nil && run.Status == quiz.RunStatusRunning {
					count := b.engine.GetParticipantCount(run.ID)
					b.authorRunMsg_mute.RLock()
					b.client.EditMessage(chatID, b.authorRunMsg[run.ID].MessageID, fmt.Sprintf(successfulFileLoad, quizObj.Title, run.ID, link, count), optsStop)
					b.authorRunMsg_mute.RUnlock()
				}
				return
			case <-ticker.C:
				count := b.engine.GetParticipantCount(run.ID)
				if count != lastCount {
					b.authorRunMsg_mute.RLock()
					b.client.EditMessage(chatID, b.authorRunMsg[run.ID].MessageID, fmt.Sprintf(successfulFileLoad, quizObj.Title, run.ID, link, count), optsStart)
					b.authorRunMsg_mute.RUnlock()
				}
			}
		}
	}()
	return nil
}

func (b *Bot) HandleCallbackQuery(callback *CallbackQuery) error {

	if strings.HasPrefix(callback.Data, "start_quiz_") {
		runID := strings.TrimPrefix(callback.Data, "start_quiz_")

		events, err := b.engine.StartQuiz(context.Background(), runID)
		if err != nil {
			b.client.AnswerCallback(callback.ID, fmt.Sprintf(unsuccessfulRunQuiz, err))
			return nil
		}

		go b.EventsProces(runID, events)

		b.client.AnswerCallback(callback.ID, successfulRunQuiz)
		return nil
	}

	if strings.HasPrefix(callback.Data, "cancel_lobby_") {
		runID := strings.TrimPrefix(callback.Data, "cancel_lobby_")

		run, err := b.engine.GetRun(runID)
		if err != nil {
			fmt.Printf("Underfind runID for cancel_lobby: %s", runID)
			b.client.AnswerCallback(callback.ID, logicalError)
			return nil
		}

		b.authorRunMsg_mute.RLock()
		msg, ok := b.authorRunMsg[runID]
		if !ok || msg == nil {
			fmt.Printf("Underfind runID for cancel_lobby: %s", runID)
			b.client.AnswerCallback(callback.ID, logicalError)
			return nil
		}
		b.authorRunMsg_mute.RUnlock()

		opts := &SendOptions{
			ReplyMarkup: &InlineKeyboardMarkup{
				InlineKeyboard: [][]InlineKeyboardButton{
					{
						{Text: "Повторить квиз", CallbackData: "restart_quiz_" + run.ID},
					},
				},
			},
		}

		(*run.LobbyRun.CancelFunc)()
		b.client.EditMessage(msg.Chat.ID, msg.MessageID, fmt.Sprintf(cancelLobby, run.QuizPtr.Title, runID), opts)

		b.client.AnswerCallback(callback.ID, successfulStopQuiz)
		return nil
	}

	if strings.HasPrefix(callback.Data, "cancel_quiz_") {
		runID := strings.TrimPrefix(callback.Data, "cancel_quiz_")

		run, err := b.engine.GetRun(runID)
		if err != nil {
			fmt.Printf("Underfind runID for cancel_quiz: %s", runID)
			return nil
		}

		b.authorRunMsg_mute.RLock()
		msg, ok := b.authorRunMsg[runID]
		if !ok || msg == nil {
			fmt.Printf("Underfind runID for cancel_lobby: %s", runID)
			b.client.AnswerCallback(callback.ID, logicalError)
			return nil
		}
		b.authorRunMsg_mute.RUnlock()

		opts := &SendOptions{
			ReplyMarkup: &InlineKeyboardMarkup{
				InlineKeyboard: [][]InlineKeyboardButton{
					{
						{Text: "Повторить квиз", CallbackData: "restart_quiz_" + run.ID},
					},
				},
			},
		}

		close(run.ChStopRun)
		b.client.EditMessage(msg.Chat.ID, msg.MessageID, fmt.Sprintf(cancelRunning, run.QuizPtr.Title, runID), opts)

		b.client.AnswerCallback(callback.ID, successfulStopQuiz)
		return nil
	}

	if strings.HasPrefix(callback.Data, "download_csv_") {
		runID := strings.TrimPrefix(callback.Data, "download_csv_")

		run, err := b.engine.GetRun(runID)
		if err != nil {
			fmt.Printf("Underfind runID for download_csv: %s", runID)
			return nil
		}

		b.authorRunMsg_mute.RLock()
		msg, ok := b.authorRunMsg[runID]
		if !ok || msg == nil {
			fmt.Printf("Underfind runID for download_csv: %s", runID)
			b.client.AnswerCallback(callback.ID, logicalError)
			return nil
		}
		b.authorRunMsg_mute.RUnlock()

		file, err := b.engine.ExportCSV(runID)
		if err != nil {
			fmt.Printf("invalid export csv for runID: %s", runID)
			b.client.AnswerCallback(callback.ID, fmt.Sprintf(errorExportCSV, err))
			return nil
		}

		b.client.SendDocument(
			msg.Chat.ID,
			fmt.Sprintf(resultsCSVDocName, runID, run.FinishedAt.Format("2006-01-02_15-04-05")),
			file,
		)

		b.client.AnswerCallback(callback.ID, successfulExportCSV)
		return nil
	}

	if strings.HasPrefix(callback.Data, "restart_quiz_") {
		runID := strings.TrimPrefix(callback.Data, "restart_quiz_")

		prevRun, err := b.engine.GetRun(runID)
		if err != nil {
			fmt.Printf("Underfind runID for restart_quiz: %s", runID)
			b.client.AnswerCallback(callback.ID, fmt.Sprintf(unsuccessfulRestartQuiz, err))
			return nil
		}

		quizObj := prevRun.QuizPtr

		err = b.LobbyProc(quizObj, callback.From.ID, true)
		if err == nil {
			b.client.AnswerCallback(callback.ID, successfulRestartRunAnswer)
		}
		return err
	}

	b.userRunID_mute.RLock()
	defer b.userRunID_mute.RUnlock()

	runID, ok := b.userRunID[callback.Message.Chat.ID]
	if !ok {
		b.client.AnswerCallback(callback.ID, answerWithoutConnectToRun)
		return nil
	}

	if err := b.engine.SubmitAnswerByLetter(context.Background(), runID, callback.Message.Chat.ID, callback.Data); err != nil {
		b.client.AnswerCallback(callback.ID, fmt.Sprintf(unsuccessfulSubmit, err))
		return nil
	}

	run, err := b.engine.GetRun(runID)
	if err == nil {
		question := (*run.QuestionsSlicePtr)[run.CurrentQuestionIdx]

		if question.Explanation != "" {
			text := b.BuildQuestion(run.CurrentQuestionIdx+1, &question, false) + fmt.Sprintf(questionExplanation, question.Explanation)
			b.client.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, text, nil)
		}
	}

	b.client.AnswerCallback(callback.ID, successfulSubmit)
	return nil
}

func (b *Bot) HandleDocument(message *Message) error {
	if !strings.HasSuffix(message.Document.FileName, ".json") {
		b.client.SendMessage(message.Chat.ID, invalidFormatOfFile, nil)
		return nil
	}

	path, err := b.client.GetFile(message.Document.FileID)
	if err != nil {
		b.client.SendMessage(message.Chat.ID, fmt.Sprintf(errorFile, err), nil)
		return err
	}

	data, err := b.client.DownloadFile(path)
	if err != nil {
		b.client.SendMessage(message.Chat.ID, fmt.Sprintf(errorFile, err), nil)
		return err
	}

	quizObj, err := b.engine.LoadQuiz(data)
	if err != nil {
		b.client.SendMessage(message.Chat.ID, errorStructFile, nil)
		return nil
	}

	b.engine.SetQuizOwner(message.From.ID, quizObj)

	return b.LobbyProc(quizObj, message.Chat.ID, false)
}

func (b *Bot) BuildQuestion(num int, question *quiz.Question, withOpts bool) string {
	var buffer strings.Builder

	buffer.Grow(512)

	buffer.WriteString(fmt.Sprintf(questionNum, num))
	buffer.WriteString(question.Text)
	buffer.WriteString("\n")

	for i, ans := range question.Options {
		buffer.WriteString(fmt.Sprintf(answerLetter, letters[i:i+1]))
		buffer.WriteString(ans)
	}

	buffer.WriteString("\n\n")

	buffer.WriteString(fmt.Sprintf(timePerQuestion, question.Time))
	if question.Time%10 == 1 && question.Time%100 != 11 {
		buffer.WriteString(endingA)
	} else if 0 < question.Time%10 && question.Time%10 <= 4 &&
		question.Time%100 != 12 && question.Time%100 != 13 &&
		question.Time%100 != 14 {
		buffer.WriteString(endingB)
	} else {
		buffer.WriteString("\n\n")
	}
	if withOpts {
		buffer.WriteString(actionsIndication)
		for i := range question.Options {
			buffer.WriteString(letters[i : i+1])
			buffer.WriteString(" ")
			if i+2 == len(question.Options) {
				buffer.WriteString(penultimateOne)
			}
		}
	}
	return buffer.String()
}

func (b *Bot) BuildWordEnding(v int) string {
	if v%10 == 1 && v%100 != 11 {
		return "\n"
	}
	if 1 < v%10 && v%10 < 5 && v%100 != 12 &&
		v%100 != 13 && v%100 != 14 {
		return endingD
	}
	return endingC
}

func (b *Bot) BuildLeadbordTop(res *quiz.QuizResults) string {
	if len(res.Leaderboard) < 1 {
		return emptyRes
	}
	var buffer strings.Builder

	buffer.Grow(512)

	buffer.WriteString(fmt.Sprintf(leadboardTopTitle, min(10, len(res.Leaderboard))))

	for i, entry := range res.Leaderboard[:min(11, len(res.Leaderboard))] {
		switch i {
		case 0:
			buffer.WriteString("🥇 1. ")
			buffer.WriteString(fmt.Sprintf(personalLeadData, entry.Participant.Username, entry.Score))
			buffer.WriteString(b.BuildWordEnding(entry.Score))
		case 1:
			buffer.WriteString("🥈 2. ")
			buffer.WriteString(fmt.Sprintf(personalLeadData, entry.Participant.Username, entry.Score))
			buffer.WriteString(b.BuildWordEnding(entry.Score))
		case 2:
			buffer.WriteString("🥉 3. ")
			buffer.WriteString(fmt.Sprintf(personalLeadData, entry.Participant.Username, entry.Score))
			buffer.WriteString(b.BuildWordEnding(entry.Score))
		default:
			buffer.WriteString(fmt.Sprintf("%d. ", i+1))
			buffer.WriteString(fmt.Sprintf(personalLeadData, entry.Participant.Username, entry.Score))
			buffer.WriteString(b.BuildWordEnding(entry.Score))
		}
	}
	return buffer.String()
}

func (b *Bot) BuildLeaderboard(res *quiz.QuizResults, entry *quiz.LeaderboardEntry, leadTop string) string {
	var buffer strings.Builder

	buffer.Grow(512)
	buffer.WriteString(fmt.Sprintf(quizResultsTitle, res.QuizTitle))
	if entry != nil {
		buffer.WriteString(fmt.Sprintf(personalRes, entry.Score))
		buffer.WriteString(b.BuildWordEnding(entry.Score))
	}
	buffer.WriteString(leadTop)

	return buffer.String()
}

func (b *Bot) EventsProces(runID string, chEvent <-chan (quiz.QuizEvent)) {

	run, err := b.engine.GetRun(runID)
	if err != nil {
		fmt.Printf("Underfind runID in events processing")
		return
	}

	for {
		select {
		case event := <-chEvent:
			if event.Type == quiz.EventTypeQuestion {
				beginSend := make(chan struct{})
				var wg sync.WaitGroup

				row := []InlineKeyboardButton{}

				for i := range event.Question.Options {
					if i >= len(letters) {
						break
					}

					button := InlineKeyboardButton{
						Text:         letters[i : i+1],
						CallbackData: letters[i : i+1],
					}
					row = append(row, button)
				}

				for _, p := range run.Participants {
					wg.Add(1)
					go func(p *quiz.Participant) {
						defer wg.Done()
						opts := &SendOptions{
							ReplyMarkup: &InlineKeyboardMarkup{
								InlineKeyboard: [][]InlineKeyboardButton{row},
							},
						}
						<-beginSend
						if p.EditMsgID == -1 {
							msg, _ := b.client.SendMessage(p.TelegramID, b.BuildQuestion(run.CurrentQuestionIdx+1, event.Question, true), opts)
							p.EditMsgID = msg.MessageID
						} else {
							b.client.EditMessage(p.TelegramID, p.EditMsgID, b.BuildQuestion(run.CurrentQuestionIdx+1, event.Question, true), opts)
						}
					}(p)
				}

				close(beginSend)
				wg.Wait()
			} else if event.Type == quiz.EventTypeFinished {

				results, err := b.engine.GetResults(runID)
				if err != nil {
					fmt.Printf("Invalid runID for GetResults: %s", runID)
					return
				}

				leadbordTop := b.BuildLeadbordTop(results)

				beginSend := make(chan struct{})
				var wg sync.WaitGroup
				for _, entry := range results.Leaderboard {
					wg.Add(1)
					go func(e *quiz.LeaderboardEntry) {
						defer wg.Done()
						text := b.BuildLeaderboard(results, e, leadbordTop)
						<-beginSend
						if e.Participant.EditMsgID == -1 {
							msg, _ := b.client.SendMessage(e.Participant.TelegramID, text, nil)
							e.Participant.EditMsgID = msg.MessageID
						} else {
							b.client.EditMessage(e.Participant.TelegramID, e.Participant.EditMsgID, text, nil)
						}
					}(&entry)
				}
				close(beginSend)
				wg.Wait()

				b.authorRunMsg_mute.RLock()
				message, ok := b.authorRunMsg[runID]
				b.authorRunMsg_mute.RUnlock()

				if !ok {
					fmt.Printf("Invalid runID for request to authorRunMsg: %s", runID)
					return
				}

				text := fmt.Sprintf(authorMsgAthterRun, run.QuizPtr.Title, runID) + b.BuildLeaderboard(results, nil, leadbordTop)

				opts := &SendOptions{
					ReplyMarkup: &InlineKeyboardMarkup{
						InlineKeyboard: [][]InlineKeyboardButton{
							{
								{Text: "Скачать CSV", CallbackData: "download_csv_" + run.ID},
								{Text: "Повторить квиз", CallbackData: "restart_quiz_" + run.ID},
							},
						},
					},
				}

				b.client.EditMessage(message.Chat.ID, message.MessageID, text, opts)
				return
			}
		case <-run.ChStopRun:

			beginSend := make(chan struct{})
			var wg sync.WaitGroup
			for _, p := range run.Participants {
				wg.Add(1)
				go func(p *quiz.Participant) {
					defer wg.Done()
					<-beginSend
					if p.EditMsgID == -1 {
						msg, _ := b.client.SendMessage(p.TelegramID, fmt.Sprintf(cancelParticipantMsg, run.QuizPtr.Title), nil)
						p.EditMsgID = msg.MessageID
					} else {
						b.client.EditMessage(p.TelegramID, p.EditMsgID, fmt.Sprintf(cancelParticipantMsg, run.QuizPtr.Title), nil)
					}
				}(p)
			}

			close(beginSend)
			wg.Wait()
			return
		}
	}
}
