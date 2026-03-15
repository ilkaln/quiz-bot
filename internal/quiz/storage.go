//go:build !change

package quiz

import (
	"context"
)

// Storage определяет интерфейс для хранения данных квизов.
type Storage interface {
	// SaveQuiz сохраняет квиз.
	SaveQuiz(ctx context.Context, q *Quiz) error

	// GetQuiz возвращает квиз по ID.
	GetQuiz(ctx context.Context, id string) (*Quiz, error)

	// ListQuizzes возвращает список квизов пользователя.
	ListQuizzes(ctx context.Context, ownerID int64) ([]*Quiz, error)

	// DeleteQuiz удаляет квиз.
	DeleteQuiz(ctx context.Context, id string) error

	// SaveRun сохраняет запуск квиза.
	SaveRun(ctx context.Context, run *QuizRun) error

	// GetRun возвращает запуск по ID.
	GetRun(ctx context.Context, id string) (*QuizRun, error)

	// ListRuns возвращает список запусков квиза.
	ListRuns(ctx context.Context, quizID string) ([]*QuizRun, error)

	// UpdateRun обновляет данные запуска.
	UpdateRun(ctx context.Context, run *QuizRun) error

	GetOwnerQuizzes(ownerID int64) []*Quiz

	SetQuizOwner(ownerID int64, quiz *Quiz)
}
