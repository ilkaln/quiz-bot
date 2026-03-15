//go:build !solution

package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/ilkaln/quiz-bot/internal/quiz"
)

// MemoryStorage реализует Storage в памяти.
type MemoryStorage struct {
	// TODO: добавьте необходимые поля
	QuizIDMap    map[string]*quiz.Quiz
	OwnerIDmap   map[int64][]*quiz.Quiz
	RunIDMap     map[string]*quiz.QuizRun
	RunQuizIDMap map[string][]*quiz.QuizRun

	mute sync.Mutex
}

// NewMemoryStorage создаёт новый MemoryStorage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		QuizIDMap:    make(map[string]*quiz.Quiz),
		OwnerIDmap:   make(map[int64][]*quiz.Quiz),
		RunIDMap:     make(map[string]*quiz.QuizRun),
		RunQuizIDMap: make(map[string][]*quiz.QuizRun),
	}
}

// SaveQuiz сохраняет квиз.
func (s *MemoryStorage) SaveQuiz(ctx context.Context, q *quiz.Quiz) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mute.Lock()
	defer s.mute.Unlock()

	if q == nil {
		return fmt.Errorf("uaveQuiz got nil quiz")
	}

	s.QuizIDMap[q.ID] = q

	return nil
}

// GetQuiz возвращает квиз по ID.
func (s *MemoryStorage) GetQuiz(ctx context.Context, id string) (*quiz.Quiz, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mute.Lock()
	defer s.mute.Unlock()

	result, ok := s.QuizIDMap[id]
	if !ok {
		return nil, fmt.Errorf("getQuiz got underfind ID: %s", id)
	}

	return result, nil
}

// ListQuizzes возвращает список квизов пользователя.
func (s *MemoryStorage) ListQuizzes(ctx context.Context, ownerID int64) ([]*quiz.Quiz, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mute.Lock()
	defer s.mute.Unlock()

	result, ok := s.OwnerIDmap[ownerID]
	if !ok {
		return nil, fmt.Errorf("listQuizzes got underfind ownerID: %d", ownerID)
	}

	return result, nil
}

// DeleteQuiz удаляет квиз.
func (s *MemoryStorage) DeleteQuiz(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mute.Lock()
	defer s.mute.Unlock()

	quizToDelete, ok := s.QuizIDMap[id]
	if !ok {
		return nil
	}

	if slice, ok := s.OwnerIDmap[quizToDelete.OwnerID]; ok {
		for i, ptr := range slice {
			if ptr == quizToDelete {
				slice[i] = slice[len(slice)-1]
				slice = slice[:len(slice)-1]

				break
			}
		}

		s.OwnerIDmap[quizToDelete.OwnerID] = slice
	}

	delete(s.QuizIDMap, id)

	return nil
}

// SaveRun сохраняет запуск квиза.
func (s *MemoryStorage) SaveRun(ctx context.Context, run *quiz.QuizRun) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mute.Lock()
	defer s.mute.Unlock()

	if run == nil {
		return fmt.Errorf("saveRun got nil run")
	}

	if _, ok := s.RunQuizIDMap[run.QuizID]; !ok {
		s.RunQuizIDMap[run.QuizID] = make([]*quiz.QuizRun, 0)
	}

	s.RunIDMap[run.ID] = run
	s.RunQuizIDMap[run.QuizID] = append(s.RunQuizIDMap[run.QuizID], run)

	return nil
}

// GetRun возвращает запуск по ID.
func (s *MemoryStorage) GetRun(_ context.Context, id string) (*quiz.QuizRun, error) {
	s.mute.Lock()
	defer s.mute.Unlock()

	result, ok := s.RunIDMap[id]
	if !ok {
		return nil, fmt.Errorf("getRun got undefind runID: %s", id)
	}

	return result, nil
}

// ListRuns возвращает список запусков квиза.
func (s *MemoryStorage) ListRuns(_ context.Context, quizID string) ([]*quiz.QuizRun, error) {
	s.mute.Lock()
	defer s.mute.Unlock()

	result, ok := s.RunQuizIDMap[quizID]
	if !ok {
		return nil, fmt.Errorf("listRuns got undefind quizID: %s", quizID)
	}

	return result, nil
}

// UpdateRun обновляет данные запуска.
func (s *MemoryStorage) UpdateRun(ctx context.Context, run *quiz.QuizRun) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mute.Lock()
	defer s.mute.Unlock()

	if run == nil {
		return fmt.Errorf("updateRun got nil run")
	}

	if _, ok := s.RunIDMap[run.ID]; !ok {
		return fmt.Errorf("updateRun got run with underfind id: %s", run.ID)
	}

	s.RunIDMap[run.ID] = run

	return nil
}

func (s *MemoryStorage) GetOwnerQuizzes(ownerID int64) []*quiz.Quiz {
	s.mute.Lock()
	defer s.mute.Unlock()
	slice, ok := s.OwnerIDmap[ownerID]
	if !ok {
		return make([]*quiz.Quiz, 0)
	}
	return slice
}

func (s *MemoryStorage) SetQuizOwner(ownerID int64, q *quiz.Quiz) {
	s.mute.Lock()
	defer s.mute.Unlock()
	if q == nil {
		return
	}

	if _, ok := s.OwnerIDmap[ownerID]; !ok {
		s.OwnerIDmap[ownerID] = make([]*quiz.Quiz, 0)
	}
	s.OwnerIDmap[ownerID] = append(s.OwnerIDmap[ownerID], q)
}
