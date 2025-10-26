package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"gorm.io/gorm"

	"perfugo/internal/config"
	"perfugo/internal/server"
)

type stubServer struct {
	startErr       error
	stopErr        error
	blockUntilStop bool

	startCalled bool
	stopCalled  bool

	startGate   chan struct{}
	startNotify chan struct{}
}

func newStubServer(startErr, stopErr error, block bool) *stubServer {
	s := &stubServer{
		startErr:       startErr,
		stopErr:        stopErr,
		blockUntilStop: block,
		startNotify:    make(chan struct{}),
	}
	if block {
		s.startGate = make(chan struct{})
	}
	return s
}

func (s *stubServer) Start() error {
	s.startCalled = true
	close(s.startNotify)
	if s.blockUntilStop {
		<-s.startGate
	}
	return s.startErr
}

func (s *stubServer) Stop() error {
	s.stopCalled = true
	if s.blockUntilStop {
		close(s.startGate)
	}
	return s.stopErr
}

func TestRunUsesMockDatabaseWhenConfigured(t *testing.T) {
	originalLoadConfig := loadConfigFunc
	originalSetLogLevel := setLogLevelFunc
	originalMock := newMockDatabaseFunc
	originalConfigure := configureDatabase
	originalNewServer := newServerFunc
	originalSubscribe := subscribeShutdownSig

	t.Cleanup(func() {
		loadConfigFunc = originalLoadConfig
		setLogLevelFunc = originalSetLogLevel
		newMockDatabaseFunc = originalMock
		configureDatabase = originalConfigure
		newServerFunc = originalNewServer
		subscribeShutdownSig = originalSubscribe
	})

	cfg := config.Config{
		Server: config.ServerConfig{Addr: ":8080"},
		Database: config.DatabaseConfig{
			UseMock: true,
		},
		Logging: config.LoggingConfig{Level: "debug"},
		Auth: config.AuthConfig{
			Session: config.SessionConfig{
				Lifetime:     time.Hour,
				CookieName:   "test",
				CookieSecure: true,
			},
		},
	}

	var mockCalled bool
	loadConfigFunc = func() (config.Config, error) { return cfg, nil }
	setLogLevelFunc = func(level string) error { return nil }
	newMockDatabaseFunc = func(ctx context.Context) (*gorm.DB, error) {
		mockCalled = true
		return &gorm.DB{}, nil
	}
	configureDatabase = func(config.DatabaseConfig) (*gorm.DB, error) {
		t.Fatal("configureDatabase should not be called when mock is enabled")
		return nil, nil
	}

	serverStub := newStubServer(http.ErrServerClosed, nil, true)
	newServerFunc = func(server.Config) (serverLifecycle, error) {
		return serverStub, nil
	}

	shutdownCh := make(chan os.Signal, 1)
	subscribeShutdownSig = func() (<-chan os.Signal, func()) {
		return shutdownCh, func() {}
	}

	go func() {
		<-serverStub.startNotify
		shutdownCh <- syscall.SIGTERM
	}()

	code := run(context.Background())
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !mockCalled {
		t.Fatal("expected mock database to be used")
	}
	if !serverStub.startCalled || !serverStub.stopCalled {
		t.Fatal("expected server start and stop to be invoked")
	}
}

func TestRunReturnsErrorWhenServerStartFails(t *testing.T) {
	originalLoadConfig := loadConfigFunc
	originalSetLogLevel := setLogLevelFunc
	originalMock := newMockDatabaseFunc
	originalNewServer := newServerFunc
	originalSubscribe := subscribeShutdownSig

	t.Cleanup(func() {
		loadConfigFunc = originalLoadConfig
		setLogLevelFunc = originalSetLogLevel
		newMockDatabaseFunc = originalMock
		newServerFunc = originalNewServer
		subscribeShutdownSig = originalSubscribe
	})

	cfg := config.Config{
		Server: config.ServerConfig{Addr: ":8080"},
		Database: config.DatabaseConfig{
			UseMock: true,
		},
		Logging: config.LoggingConfig{Level: "info"},
		Auth:    config.AuthConfig{Session: config.SessionConfig{Lifetime: time.Hour}},
	}

	loadConfigFunc = func() (config.Config, error) { return cfg, nil }
	setLogLevelFunc = func(string) error { return nil }
	newMockDatabaseFunc = func(context.Context) (*gorm.DB, error) { return &gorm.DB{}, nil }

	serverStub := newStubServer(errors.New("listener failure"), nil, false)
	newServerFunc = func(server.Config) (serverLifecycle, error) {
		return serverStub, nil
	}

	subscribeShutdownSig = func() (<-chan os.Signal, func()) {
		return make(chan os.Signal), func() {}
	}

	code := run(context.Background())
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if serverStub.stopCalled {
		t.Fatal("server stop should not be called on start error")
	}
}

func TestRunHandlesDatabaseConfigurationError(t *testing.T) {
	originalLoadConfig := loadConfigFunc
	originalSetLogLevel := setLogLevelFunc
	originalMock := newMockDatabaseFunc
	originalConfigure := configureDatabase

	t.Cleanup(func() {
		loadConfigFunc = originalLoadConfig
		setLogLevelFunc = originalSetLogLevel
		newMockDatabaseFunc = originalMock
		configureDatabase = originalConfigure
	})

	cfg := config.Config{
		Server:   config.ServerConfig{Addr: ":8080"},
		Database: config.DatabaseConfig{URL: "postgres://example", UseMock: false},
		Logging:  config.LoggingConfig{Level: "info"},
	}

	loadConfigFunc = func() (config.Config, error) { return cfg, nil }
	setLogLevelFunc = func(string) error { return nil }
	newMockDatabaseFunc = func(context.Context) (*gorm.DB, error) {
		t.Fatal("mock database should not be used when URL is configured")
		return nil, nil
	}
	configureDatabase = func(config.DatabaseConfig) (*gorm.DB, error) {
		return nil, errors.New("db connection refused")
	}

	code := run(context.Background())
	if code != 1 {
		t.Fatalf("expected exit code 1 on database configuration failure, got %d", code)
	}
}

func TestRunReturnsErrorWhenLogLevelInvalid(t *testing.T) {
	originalLoadConfig := loadConfigFunc
	originalSetLogLevel := setLogLevelFunc

	t.Cleanup(func() {
		loadConfigFunc = originalLoadConfig
		setLogLevelFunc = originalSetLogLevel
	})

	cfg := config.Config{Logging: config.LoggingConfig{Level: "invalid"}}
	loadConfigFunc = func() (config.Config, error) { return cfg, nil }
	setLogLevelFunc = func(string) error { return errors.New("invalid level") }

	code := run(context.Background())
	if code != 1 {
		t.Fatalf("expected exit code 1 for invalid log level, got %d", code)
	}
}
