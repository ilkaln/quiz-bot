//go:build !solution

package quiz

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	mrand "math/rand"
	"sort"
	"strconv"
	"time"
)

// Engine реализует QuizEngine.
type Engine struct {
	// TODO: добавьте необходимые поля
	storage Storage
}

// NewEngine создаёт новый QuizEngine.
func NewEngine() *Engine {
	return &Engine{
		storage: NewMemoryStorage(),
	}
}

func generateID(size int) (string, error) {
	cntBytes := size / 2

	bytes := make([]byte, cntBytes)

	if _, err := crand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

// LoadQuiz парсит JSON и создаёт квиз.
func (e *Engine) LoadQuiz(data []byte) (*Quiz, error) {
	parsedJSON := QuizInputData{
		Settings: Settings{
			MaxParticipants: math.MaxInt,
			Registration:    []string{},
		},
	}

	err := json.Unmarshal(data, &parsedJSON)
	if err != nil {
		return nil, err
	}

	if parsedJSON.Title == "" {
		return nil, fmt.Errorf("quiz title is nil")
	}

	if parsedJSON.Settings.TimePerQuestion <= 0 {
		return nil, fmt.Errorf("invalid time per question: %d", parsedJSON.Settings.TimePerQuestion)
	}

	if len(parsedJSON.Questions) == 0 {
		return nil, fmt.Errorf("quiz did not got any questions")
	}

	for i := range parsedJSON.Questions {
		q := &parsedJSON.Questions[i]
		if q.Text == "" {
			return nil, fmt.Errorf("the question %d should have title", i)
		}

		if len(q.Options) < 2 || 6 < len(q.Options) {
			return nil, fmt.Errorf(
				"the question %d should have from 2 to 6 optons, but given %d",
				i,
				len(q.Options),
			)
		}

		if q.Correct < 0 || q.Correct >= len(q.Options) {
			return nil, fmt.Errorf("the question %d has invalid correct answer index: %q. Should be from 0 to (%d - 1)", i, q.Correct, len(q.Options))
		}
		if q.Points <= 0 {
			q.Points = 1
		}

		if q.Time <= 0 {
			q.Time = parsedJSON.Settings.TimePerQuestion
		}
	}

	quizID, err := generateID(16)
	if err != nil {
		return nil, err
	}

	result := Quiz{
		ID:           quizID,
		OwnerID:      0,
		Title:        parsedJSON.Title,
		Settings:     parsedJSON.Settings,
		Questions:    parsedJSON.Questions,
		CreatedAt:    time.Now(),
		BytesJSONPtr: &data,
	}

	if err := e.storage.SaveQuiz(context.Background(), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// StartRun создаёт новый запуск квиза.
func (e *Engine) StartRun(ctx context.Context, quiz *Quiz) (*QuizRun, error) {

	runID, err := generateID(16)
	if err != nil {
		return nil, err
	}

	ctxLobby, cancelFunc := context.WithCancel(ctx)

	run := QuizRun{
		ID:      runID,
		QuizID:  quiz.ID,
		QuizPtr: quiz,
		Status:  RunStatusLobby,
		LobbyRun: &LobbyRun{
			CancelFunc: &cancelFunc,
			Context:    &ctxLobby},
		Participants: make(map[int64]*Participant),
		Answers:      make(map[int64][]Answer),
		ChStopRun:    make(chan struct{}),
	}

	if err := e.storage.SaveRun(ctx, &run); err != nil {
		return nil, err
	}

	return &run, nil
}

// JoinRun добавляет участника в запуск квиза.
func (e *Engine) JoinRun(ctx context.Context, runID string, participant *Participant) error {
	if participant == nil {
		return fmt.Errorf("joinrun got nil participant")
	}

	run, err := e.storage.GetRun(ctx, runID)
	if err != nil {
		return err
	}

	quiz, err := e.storage.GetQuiz(ctx, run.QuizID)
	if err != nil {
		return err
	}

	if len(run.Participants) >= quiz.Settings.MaxParticipants {
		return fmt.Errorf("maximum count of particioant by %d reached", quiz.Settings.MaxParticipants)
	}

	if run.Status != RunStatusLobby {
		return fmt.Errorf("invalid status %q during the joinrun. should be %q", run.Status, RunStatusLobby)
	}

	if _, ok := run.Participants[participant.TelegramID]; ok {
		return fmt.Errorf("participant with ID %v was already joined", participant.TelegramID)
	}
	run.Participants[participant.TelegramID] = participant

	return e.storage.UpdateRun(ctx, run)
}

// GetParticipantCount возвращает текущее количество участников.
func (e *Engine) GetParticipantCount(runID string) int {
	run, err := e.storage.GetRun(context.Background(), runID)
	if err != nil {
		return 0
	}
	return len(run.Participants)
}

// StartQuiz запускает квиз.
func (e *Engine) StartQuiz(ctx context.Context, runID string) (<-chan QuizEvent, error) {

	run, err := e.storage.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	if run.Status != RunStatusLobby {
		return nil, fmt.Errorf("invalid status %q of run. should be %q", run.Status, RunStatusLobby)
	}
	run.Status = RunStatusRunning
	e.storage.UpdateRun(ctx, run)

	quiz, err := e.storage.GetQuiz(ctx, run.QuizID)
	if err != nil {
		return nil, err
	}

	questions := quiz.Questions
	if quiz.Settings.ShuffleQuestions {
		mrand.Shuffle(len(questions), func(i int, j int) {
			questions[i], questions[j] = questions[j], questions[i]
		})
	}

	if quiz.Settings.ShuffleAnswers {
		for k := range questions {
			mrand.Shuffle(len(questions[k].Options), func(i, j int) {
				questions[k].Options[i], questions[k].Options[j] = questions[k].Options[j], questions[k].Options[i]
				if i == questions[k].Correct {
					questions[k].Correct = j
				} else if j == questions[k].Correct {
					questions[k].Correct = i
				}
			})
		}
	}

	chEvent := make(chan QuizEvent)
	(*run.LobbyRun.CancelFunc)()

	go func() {
		defer close(chEvent)

		run.StartedAt = time.Now()
		run.QuestionsSlicePtr = &questions
		run.QuestionsStartsAt = make([]time.Time, len(questions))
		run.setQuestionReaponses = make([]map[int64]struct{}, len(questions))
		e.storage.UpdateRun(ctx, run)

		timePerQuastion := quiz.Settings.TimePerQuestion

		for i, curQuastion := range questions {
			if curQuastion.Time <= 0 {
				questions[i].Time = timePerQuastion
			}
			curPerTime := curQuastion.Time
			run.CurrentQuestionIdx = i
			run.setQuestionReaponses[i] = make(map[int64]struct{}, len(run.Participants))
			e.storage.UpdateRun(ctx, run)

			run.QuestionsStartsAt[i] = time.Now()
			chEvent <- QuizEvent{
				Type:        EventTypeQuestion,
				QuestionIdx: i,
				Question:    &curQuastion,
			}
			select {
			case <-time.After(time.Second * time.Duration(curPerTime)):
				if len(run.setQuestionReaponses[i]) < len(run.Participants) {
					chEvent <- QuizEvent{
						Type:        EventTypeTimeUp,
						QuestionIdx: i,
					}
				}
			case <-ctx.Done():
				return
			}
		}

		chEvent <- QuizEvent{
			Type: EventTypeFinished,
		}

		run.FinishedAt = time.Now()
		run.Status = RunStatusFinished
		e.storage.UpdateRun(ctx, run)
	}()

	return chEvent, nil
}

// SubmitAnswer регистрирует ответ участника.
func (e *Engine) SubmitAnswer(ctx context.Context, runID string, participantID int64, questionIdx int, answerIdx int) error {
	run, err := e.storage.GetRun(ctx, runID)
	if err != nil {
		return err
	}

	if run.Status != RunStatusRunning {
		return fmt.Errorf("invalid status %q of run. should be %q", run.Status, RunStatusRunning)
	}

	if questionIdx >= len(*(run.QuestionsSlicePtr)) {
		return fmt.Errorf("question index %q out of range", questionIdx)
	}

	if answerIdx >= len((*run.QuestionsSlicePtr)[questionIdx].Options) {
		return fmt.Errorf("answer index %q out of range", answerIdx)
	}

	if len(run.Answers[participantID]) > 0 && run.Answers[participantID][len(run.Answers[participantID])-1].QuestionIdx == questionIdx {
		return fmt.Errorf("answer with question index %q already submited", questionIdx)
	}

	if run.Answers == nil {
		run.Answers = make(map[int64][]Answer)
	}

	newAnswer := Answer{
		QuestionIdx: questionIdx,
		AnswerIdx:   answerIdx,
		AnsweredAt:  time.Now(),
	}

	run.Answers[participantID] = append(run.Answers[participantID], newAnswer)
	run.setQuestionReaponses[questionIdx][participantID] = struct{}{}

	return e.storage.UpdateRun(ctx, run)
}

// SubmitAnswerByLetter регистрирует ответ участника по букве.
func (e *Engine) SubmitAnswerByLetter(ctx context.Context, runID string, participantID int64, letter string) error {
	idx, ok := LetterToIndex(letter)
	if !ok {
		return fmt.Errorf("invalid letter %q for SubmitAnswerByLetter", letter)
	}

	run, err := e.storage.GetRun(ctx, runID)
	if err != nil {
		return err
	}

	return e.SubmitAnswer(ctx, runID, participantID, run.CurrentQuestionIdx, idx)
}

// GetCurrentQuestion возвращает текущий номер вопроса.
func (e *Engine) GetCurrentQuestion(runID string) int {
	run, err := e.storage.GetRun(context.Background(), runID)
	if err != nil || run.Status != RunStatusRunning {
		return -1
	}
	return run.CurrentQuestionIdx
}

// GetResults возвращает результаты квиза.
func (e *Engine) GetResults(runID string) (*QuizResults, error) {
	ctx := context.Background()

	run, err := e.storage.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	quiz, err := e.storage.GetQuiz(ctx, run.QuizID)
	if err != nil {
		return nil, err
	}

	leaderboard := make([]LeaderboardEntry, 0)

	for _, p := range run.Participants {
		leaderboard = append(leaderboard, LeaderboardEntry{
			Participant:  p,
			Score:        0,
			CorrectCount: 0,
			TotalTime:    0,
		})
	}

	for i := range leaderboard {
		pID := leaderboard[i].Participant.TelegramID
		if answers, ok := run.Answers[pID]; ok {
			for _, ans := range answers {
				if ans.QuestionIdx >= 0 && ans.QuestionIdx < len(quiz.Questions) {
					question := (*run.QuestionsSlicePtr)[ans.QuestionIdx]
					if ans.AnswerIdx == question.Correct {
						leaderboard[i].Score += question.Points
						leaderboard[i].CorrectCount++
					}
					leaderboard[i].TotalTime += ans.AnsweredAt.Sub(run.QuestionsStartsAt[ans.QuestionIdx])
				}
			}
		}
	}

	for i := range leaderboard {
		pID := leaderboard[i].Participant.TelegramID
		for j, q := range *run.QuestionsSlicePtr {
			if _, responded := run.setQuestionReaponses[j][pID]; !responded {
				leaderboard[i].TotalTime += time.Duration(q.Time) * time.Second
			}
		}
	}

	sort.Slice(leaderboard, func(i, j int) bool {
		a, b := leaderboard[i], leaderboard[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.CorrectCount != b.CorrectCount {
			return a.CorrectCount > b.CorrectCount
		}
		return a.TotalTime < b.TotalTime
	})

	for i := range leaderboard {
		leaderboard[i].Rank = i + 1
	}

	result := QuizResults{
		RunID:       run.ID,
		QuizTitle:   quiz.Title,
		Leaderboard: leaderboard,
		TotalTime:   run.FinishedAt.Sub(run.StartedAt),
	}

	return &result, nil
}

// ExportCSV экспортирует результаты в CSV.
func (e *Engine) ExportCSV(runID string) ([]byte, error) {
	results, err := e.GetResults(runID)
	if err != nil {
		return nil, err
	}

	run, err := e.storage.GetRun(context.Background(), runID)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)
	header := []string{"Rank", "Username", "Score", "Correct Answers", "Total Time", "Name", "Surname"}
	for _, field := range run.QuizPtr.Settings.Registration {
		header = append(header, field)
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	for _, pers := range results.Leaderboard {
		row := []string{
			strconv.Itoa(pers.Rank),
			pers.Participant.Username,
			strconv.Itoa(pers.Score),
			strconv.Itoa(pers.CorrectCount),
			fmt.Sprintf("%.2f", pers.TotalTime.Seconds()),
			pers.Participant.FirstName,
			pers.Participant.LastName,
		}

		for _, field := range run.QuizPtr.Settings.Registration {
			if val, ok := pers.Participant.RegData[field]; ok {
				row = append(row, val)
			} else {
				row = append(row, "-")
			}
		}

		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	bytes := buf.Bytes()
	run.ResultsCSVBytes = &bytes

	return bytes, nil
}

// GetRun возвращает запуск по ID.
func (e *Engine) GetRun(runID string) (*QuizRun, error) {
	run, err := e.storage.GetRun(context.Background(), runID)
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (e *Engine) GetPersQuizzes(persID int64) []*Quiz {
	return e.storage.GetOwnerQuizzes(persID)
}

func (e *Engine) SetQuizOwner(ownerID int64, quiz *Quiz) {
	e.storage.SetQuizOwner(ownerID, quiz)
}

func (e *Engine) GetQuiz(quizID string) (*Quiz, error) {
	return e.storage.GetQuiz(context.Background(), quizID)
}

func (e *Engine) GetListRuns(quizID string) ([]*QuizRun, error) {
	return e.storage.ListRuns(context.Background(), quizID)
}
