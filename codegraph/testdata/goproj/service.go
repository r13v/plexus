package main

type Base struct {
	id int
}

type Service struct {
	Base
	name string
}

type Runner interface {
	Run() string
}

type ExtendedRunner interface {
	Runner
	Stop()
}

func NewService(name string) *Service {
	return &Service{name: name}
}

func (s *Service) Run() string {
	return s.name + ": running"
}

func (s *Service) Stop() {
	cleanup()
}

func cleanup() {
	// no-op
}
