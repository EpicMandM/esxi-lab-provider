package service

import (
	"context"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type CalendarService struct {
	srv    *calendar.Service
	config CalendarConfig
}

func NewCalendarService(ctx context.Context, config CalendarConfig) (*CalendarService, error) {
	tokenJSON, err := config.LoadServiceAccountToken()
	if err != nil {
		return nil, err
	}

	srv, err := calendar.NewService(ctx, option.WithAuthCredentialsJSON(option.ServiceAccount, tokenJSON))
	if err != nil {
		return nil, err
	}

	return &CalendarService{srv: srv, config: config}, nil
}

func (s *CalendarService) ListEvents(timeMin, timeMax string) ([]*calendar.Event, error) {
	events, err := s.srv.Events.List(s.config.CalendarID).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(timeMin).
		TimeMax(timeMax).
		OrderBy("startTime").
		Do()
	if err != nil {
		return nil, err
	}
	return events.Items, nil
}
