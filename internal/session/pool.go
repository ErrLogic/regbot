// Package session manages Appium session lifecycle for RegBot.
package session

import (
	"context"
	"fmt"
	"sync"

	"github.com/ErrLogic/regbot/internal/appium"
	"github.com/ErrLogic/regbot/internal/config"
)

// Pool manages Appium session lifecycle, reusing sessions per device.
type Pool struct {
	mu       sync.Mutex
	sessions map[string]*appium.Driver // keyed by device serial
	cfg      config.AppiumConfig
}

// NewPool creates a session pool.
func NewPool(cfg config.AppiumConfig) *Pool {
	return &Pool{sessions: make(map[string]*appium.Driver), cfg: cfg}
}

// Acquire returns an existing session for the given device serial, or creates one.
func (p *Pool) Acquire(ctx context.Context, serial string, caps appium.Capabilities) (*appium.Driver, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if d, ok := p.sessions[serial]; ok {
		return d, nil
	}

	d, err := appium.NewDriver(ctx, p.cfg.ServerURL, caps)
	if err != nil {
		return nil, fmt.Errorf("session pool: create session for %s: %w", serial, err)
	}
	p.sessions[serial] = d
	return d, nil
}

// Release closes and removes the session for the given device.
func (p *Pool) Release(ctx context.Context, serial string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if d, ok := p.sessions[serial]; ok {
		_ = d.Quit(ctx)
		delete(p.sessions, serial)
	}
}

// ReleaseAll closes all sessions. Call during graceful shutdown.
func (p *Pool) ReleaseAll(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for serial, d := range p.sessions {
		_ = d.Quit(ctx)
		delete(p.sessions, serial)
	}
}

// Has returns whether a session exists for the given serial.
func (p *Pool) Has(serial string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.sessions[serial]
	return ok
}
