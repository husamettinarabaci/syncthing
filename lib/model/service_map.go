// Copyright (C) 2023 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package model

import (
	"context"
	"fmt"
	"time"

	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/svcutil"
	"github.com/thejerf/suture/v4"
)

// A serviceMap is a utility map of arbitrary keys to a suture.Service of
// some kind, where adding and removing services ensures they are properly
// started and stopped on the given Supervisor. The serviceMap is itself a
// suture.Service and should be added to a Supervisor.
// Not safe for concurrent use.
type serviceMap[K comparable, S suture.Service] struct {
	services    map[K]S
	tokens      map[K]suture.ServiceToken
	supervisor  *suture.Supervisor
	eventLogger events.Logger
}

func newServiceMap[K comparable, S suture.Service](eventLogger events.Logger) *serviceMap[K, S] {
	m := &serviceMap[K, S]{
		services:    make(map[K]S),
		tokens:      make(map[K]suture.ServiceToken),
		eventLogger: eventLogger,
	}
	m.supervisor = suture.New(m.String(), svcutil.SpecWithDebugLogger(l))
	return m
}

// Add adds a service to the map, starting it on the supervisor. If there is
// already a service at the given key, it is removed first.
func (s *serviceMap[K, S]) Add(k K, v S) {
	if tok, ok := s.tokens[k]; ok {
		// There is already a service at this key, remove it first.
		s.supervisor.Remove(tok)
		s.eventLogger.Log(events.Failure, fmt.Sprintf("%s replaced service at key %v", s, k))
	}
	s.services[k] = v
	s.tokens[k] = s.supervisor.Add(v)
}

// Get returns the service at the given key, or the empty value and false if
// there is no service at that key.
func (s *serviceMap[K, S]) Get(k K) (v S, ok bool) {
	v, ok = s.services[k]
	return
}

// Remove removes the service at the given key, stopping it on the supervisor.
// If there is no service at the given key, nothing happens. The return value
// indicates whether a service was removed.
func (s *serviceMap[K, S]) Remove(k K) (found bool) {
	if tok, ok := s.tokens[k]; ok {
		found = true
		s.supervisor.Remove(tok)
	}
	delete(s.services, k)
	delete(s.tokens, k)
	return
}

// RemoveAndWait removes the service at the given key, stopping it on the
// supervisor. If there is no service at the given key, nothing happens. The
// return value indicates whether a service was removed.
func (s *serviceMap[K, S]) RemoveAndWait(k K, timeout time.Duration) (found bool) {
	if tok, ok := s.tokens[k]; ok {
		found = true
		s.supervisor.RemoveAndWait(tok, timeout)
	}
	delete(s.services, k)
	delete(s.tokens, k)
	return found
}

// Each calls the given function for each service in the map.
func (s *serviceMap[K, S]) Each(fn func(K, S)) {
	for key, svc := range s.services {
		fn(key, svc)
	}
}

// Suture implementation

func (s *serviceMap[K, S]) Serve(ctx context.Context) error {
	return s.supervisor.Serve(ctx)
}

func (s *serviceMap[K, S]) String() string {
	var kv K
	var sv S
	return fmt.Sprintf("serviceMap[%T, %T]@%p", kv, sv, s)
}
